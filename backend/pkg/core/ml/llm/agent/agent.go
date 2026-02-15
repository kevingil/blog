package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"backend/pkg/core/ml/llm/config"
	"backend/pkg/core/ml/llm/logging"
	"backend/pkg/core/ml/llm/message"
	"backend/pkg/core/ml/llm/models"
	"backend/pkg/core/ml/llm/prompt"
	"backend/pkg/core/ml/llm/provider"
	"backend/pkg/core/ml/llm/session"
	"backend/pkg/core/ml/llm/tools"
)


// Common errors
var (
	ErrRequestCancelled = errors.New("request cancelled by user")
	ErrSessionBusy      = errors.New("session is currently processing another request")
)

type AgentEventType string

const (
	AgentEventTypeError          AgentEventType = "error"
	AgentEventTypeResponse       AgentEventType = "response"
	AgentEventTypeTool           AgentEventType = "tool"
	AgentEventTypeThinking       AgentEventType = "thinking"
	AgentEventTypeContentDelta   AgentEventType = "content_delta"
	AgentEventTypeReasoningDelta AgentEventType = "reasoning_delta"
)

type AgentEvent struct {
	Type    AgentEventType
	Message message.Message
	Error   error

	// When summarizing
	SessionID string
	Progress  string
	Done      bool

	// When thinking
	ThinkingMessage string
	Iteration       int

	// When streaming content
	ContentDelta string

	// When streaming reasoning (extended thinking)
	ReasoningDelta string
}

type Service interface {
	Model() models.Model
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error)
}

type agent struct {
	sessions session.Service
	messages message.Service

	tools    []tools.BaseTool
	provider provider.Provider

	activeRequests sync.Map
}

func NewAgent(
	agentName config.AgentName,
	sessions session.Service,
	messages message.Service,
	agentTools []tools.BaseTool,
) (Service, error) {
	// Extract tool names so the system prompt only references registered tools
	toolNames := make([]string, len(agentTools))
	for i, t := range agentTools {
		toolNames[i] = t.Info().Name
	}

	agentProvider, err := createAgentProvider(agentName, toolNames)
	if err != nil {
		return nil, err
	}

	agent := &agent{
		provider:       agentProvider,
		messages:       messages,
		sessions:       sessions,
		tools:          agentTools,
		activeRequests: sync.Map{},
	}

	return agent, nil
}

func (a *agent) Model() models.Model {
	return a.provider.Model()
}

func (a *agent) Cancel(sessionID string) {
	// Cancel regular requests
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			logging.InfoPersist(fmt.Sprintf("Request cancellation initiated for session: %s", sessionID))
			cancel()
		}
	}
}

func (a *agent) IsBusy() bool {
	busy := false
	a.activeRequests.Range(func(key, value interface{}) bool {
		if cancelFunc, ok := value.(context.CancelFunc); ok {
			if cancelFunc != nil {
				busy = true
				return false // Stop iterating
			}
		}
		return true // Continue iterating
	})
	return busy
}

func (a *agent) IsSessionBusy(sessionID string) bool {
	_, busy := a.activeRequests.Load(sessionID)
	return busy
}

func (a *agent) err(err error) AgentEvent {
	return AgentEvent{
		Type:  AgentEventTypeError,
		Error: err,
	}
}

func (a *agent) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	if !a.provider.Model().SupportsAttachments && attachments != nil {
		attachments = nil
	}
	events := make(chan AgentEvent)
	if a.IsSessionBusy(sessionID) {
		return nil, ErrSessionBusy
	}

	genCtx, cancel := context.WithCancel(ctx)

	a.activeRequests.Store(sessionID, cancel)
	go func() {
		logging.Debug("Request started", "sessionID", sessionID)
		defer logging.RecoverPanic("agent.Run", func() {
			events <- a.err(fmt.Errorf("panic while running the agent"))
		})
		var attachmentParts []message.ContentPart
		for _, attachment := range attachments {
			attachmentParts = append(attachmentParts, message.BinaryContent{Path: attachment.FilePath, MIMEType: attachment.MimeType, Data: attachment.Content})
		}
		result := a.processGenerationWithEvents(genCtx, sessionID, content, attachmentParts, events)
		if result.Error != nil && !errors.Is(result.Error, ErrRequestCancelled) && !errors.Is(result.Error, context.Canceled) {
			logging.ErrorPersist(result.Error.Error())
		}
		logging.Debug("Request completed", "sessionID", sessionID)
		a.activeRequests.Delete(sessionID)
		cancel()

		events <- result
		close(events)
	}()
	return events, nil
}

func (a *agent) processGenerationWithEvents(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart, events chan<- AgentEvent) AgentEvent {
	cfg := config.Get()

	// List existing messages; if none, start title generation asynchronously.
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to list messages: %w", err))
	}
	// Removed automatic title generation - copilot agent doesn't need this

	userMsg, err := a.createUserMessage(ctx, sessionID, content, attachmentParts)
	if err != nil {
		return a.err(fmt.Errorf("failed to create user message: %w", err))
	}

	msgHistory := append(msgs, userMsg)

	const maxIterations = 25 // Allow room for research-first workflow

	iteration := 0
	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			return a.err(ctx.Err())
		default:
			// Continue processing
		}

		// Emit thinking event at the start of each iteration
		iteration++

		// Guard against infinite loops - force a final response if we hit the limit
		if iteration > maxIterations {
			log.Printf("[Agent] ⚠️ Hit max iterations (%d), forcing completion", maxIterations)
			finalMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
				Role:  message.Assistant,
				Parts: []message.ContentPart{message.TextContent{Text: "I've made several edits to your document. Let me know if you'd like me to continue with additional changes."}},
				Model: string(a.provider.Model().ID),
			})
			if err != nil {
				return a.err(fmt.Errorf("max iterations reached (%d), failed to create final message: %w", maxIterations, err))
			}
			finalMsg.AddFinish(message.FinishReasonEndTurn)
			return AgentEvent{
				Type:    AgentEventTypeResponse,
				Message: finalMsg,
				Done:    true,
			}
		}
		thinkingEvent := AgentEvent{
			Type:            AgentEventTypeThinking,
			ThinkingMessage: "Thinking...",
			Iteration:       iteration,
			Done:            false,
		}
		events <- thinkingEvent

		log.Printf("[Agent] Iteration %d starting (msgs: %d)", iteration, len(msgHistory))
		agentMessage, toolResults, err := a.streamAndHandleEvents(ctx, sessionID, msgHistory, events)
		log.Printf("[Agent] Iteration %d done (finish: %s, hasTools: %v)", iteration, agentMessage.FinishReason(), toolResults != nil)

		if err != nil {
			if errors.Is(err, context.Canceled) {
				agentMessage.AddFinish(message.FinishReasonCanceled)
				a.messages.Update(context.Background(), agentMessage)
				return a.err(ErrRequestCancelled)
			}
			return a.err(fmt.Errorf("failed to process events: %w", err))
		}
		if cfg.Debug {
			seqId := (len(msgHistory) + 1) / 2
			logging.WriteToolResultsJson(sessionID, seqId, toolResults)
		}
		if (agentMessage.FinishReason() == message.FinishReasonToolUse) && toolResults != nil {
			// Stream the tool call message to the manager (even without text content)
			// This is necessary for the manager to track tool steps in chain of thought
			responseEvent := AgentEvent{
				Type:    AgentEventTypeResponse,
				Message: agentMessage,
				Done:    false,
			}
			events <- responseEvent

			// Stream the tool results message
			toolEvent := AgentEvent{
				Type:    AgentEventTypeTool,
				Message: *toolResults,
				Done:    false,
			}
			log.Printf("│ ✅ Tool results: %d", len(toolResults.ToolResults()))
			events <- toolEvent

			// We are not done, we need to respond with the tool response
			msgHistory = append(msgHistory, agentMessage, *toolResults)
			continue
		}
		return AgentEvent{
			Type:    AgentEventTypeResponse,
			Message: agentMessage,
			Done:    true,
		}
	}
}

func (a *agent) createUserMessage(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) (message.Message, error) {
	parts := []message.ContentPart{message.TextContent{Text: content}}
	parts = append(parts, attachmentParts...)
	return a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
}

func (a *agent) streamAndHandleEvents(ctx context.Context, sessionID string, msgHistory []message.Message, events chan<- AgentEvent) (message.Message, *message.Message, error) {
	// Log the complete context being sent to the LLM
	a.logRequestContext(sessionID, msgHistory)

	// Preserve any existing context values (like article ID) and add session ID
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	eventChan := a.provider.StreamResponse(ctx, msgHistory, a.tools)

	assistantMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{},
		Model: string(a.provider.Model().ID),
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create assistant message: %w", err)
	}

	// Add the message ID into the context while preserving existing values (like article ID and session ID)
	ctx = context.WithValue(ctx, tools.MessageIDContextKey, assistantMsg.ID)

	// Process each event in the stream.
	for event := range eventChan {
		if processErr := a.processEvent(ctx, sessionID, &assistantMsg, event, events); processErr != nil {
			a.finishMessage(ctx, &assistantMsg, message.FinishReasonCanceled)
			return assistantMsg, nil, processErr
		}
		if ctx.Err() != nil {
			a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
			return assistantMsg, nil, ctx.Err()
		}
	}

	toolResults := make([]message.ToolResult, len(assistantMsg.ToolCalls()))
	toolCalls := assistantMsg.ToolCalls()

	// Check if any tools are parallelizable
	parallelizableTools := map[string]bool{
		"search_web_sources":       true,
		"ask_question":             true,
		"get_relevant_sources":     true,
		"fetch_url":                true,
		"add_context_from_sources": true,
	}

	// Determine if we can parallelize (all tools must be parallelizable)
	canParallelize := len(toolCalls) > 1
	for _, tc := range toolCalls {
		if !parallelizableTools[tc.Name] {
			canParallelize = false
			break
		}
	}

	if canParallelize {
		// Execute tools in parallel using goroutines
		log.Printf("│ 🔄 Executing %d tools in parallel", len(toolCalls))

		type toolResultWithIndex struct {
			index  int
			result message.ToolResult
		}
		resultChan := make(chan toolResultWithIndex, len(toolCalls))

		for i, toolCall := range toolCalls {
			go func(idx int, tc message.ToolCall) {
				// Check for cancellation
				select {
				case <-ctx.Done():
					resultChan <- toolResultWithIndex{
						index: idx,
						result: message.ToolResult{
							ToolCallID: tc.ID,
							Content:    "Tool execution canceled by user",
							IsError:    true,
						},
					}
					return
				default:
				}

				// Find the tool
				var tool tools.BaseTool
				for _, availableTool := range a.tools {
					if availableTool.Info().Name == tc.Name {
						tool = availableTool
						break
					}
				}

				if tool == nil {
					resultChan <- toolResultWithIndex{
						index: idx,
						result: message.ToolResult{
							ToolCallID: tc.ID,
							Content:    fmt.Sprintf("Tool not found: %s", tc.Name),
							IsError:    true,
						},
					}
					return
				}

				// Execute the tool
				toolResult, toolErr := tool.Run(ctx, tools.ToolCall{
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Input,
				})

				if toolErr != nil {
					resultChan <- toolResultWithIndex{
						index: idx,
						result: message.ToolResult{
							ToolCallID: tc.ID,
							Content:    fmt.Sprintf("Tool execution error: %v", toolErr),
							IsError:    true,
						},
					}
				} else {
					resultChan <- toolResultWithIndex{
						index: idx,
						result: message.ToolResult{
							ToolCallID: tc.ID,
							Content:    toolResult.Content,
							Metadata:   toolResult.Metadata,
							IsError:    toolResult.IsError,
						},
					}
				}
			}(i, toolCall)
		}

		// Collect all results
		for range toolCalls {
			select {
			case <-ctx.Done():
				// Fill remaining with cancelled
				for j := 0; j < len(toolCalls); j++ {
					if toolResults[j].ToolCallID == "" {
						toolResults[j] = message.ToolResult{
							ToolCallID: toolCalls[j].ID,
							Content:    "Tool execution canceled by user",
							IsError:    true,
						}
					}
				}
				a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
				goto out
			case result := <-resultChan:
				toolResults[result.index] = result.result
			}
		}

		log.Printf("│ ✅ All parallel tools completed")
	} else {
		// Execute tools sequentially (original behavior)
		for i, toolCall := range toolCalls {
			select {
			case <-ctx.Done():
				a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
				// Make all future tool calls cancelled
				for j := i; j < len(toolCalls); j++ {
					toolResults[j] = message.ToolResult{
						ToolCallID: toolCalls[j].ID,
						Content:    "Tool execution canceled by user",
						IsError:    true,
					}
				}
				goto out
			default:
				// Continue processing
				var tool tools.BaseTool
				for _, availableTool := range a.tools {
					if availableTool.Info().Name == toolCall.Name {
						tool = availableTool
						break
					}
				}

				// Tool not found
				if tool == nil {
					toolResults[i] = message.ToolResult{
						ToolCallID: toolCall.ID,
						Content:    fmt.Sprintf("Tool not found: %s", toolCall.Name),
						IsError:    true,
					}
					continue
				}
				toolResult, toolErr := tool.Run(ctx, tools.ToolCall{
					ID:    toolCall.ID,
					Name:  toolCall.Name,
					Input: toolCall.Input,
				})
				if toolErr != nil {
					toolResults[i] = message.ToolResult{
						ToolCallID: toolCall.ID,
						Content:    fmt.Sprintf("Tool execution error: %v", toolErr),
						IsError:    true,
					}
			} else {
				toolResults[i] = message.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    toolResult.Content,
					Metadata:   toolResult.Metadata,
					IsError:    toolResult.IsError,
				}
				log.Printf("│   Tool %q → %d chars (error: %v)", toolCall.Name, len(toolResult.Content), toolResult.IsError)
			}
		}
	}
}
out:
	if len(toolResults) == 0 {
		return assistantMsg, nil, nil
	}
	parts := make([]message.ContentPart, 0)
	for _, tr := range toolResults {
		parts = append(parts, tr)
	}
	msg, err := a.messages.Create(context.Background(), assistantMsg.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: parts,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create cancelled tool message: %w", err)
	}

	return assistantMsg, &msg, err
}

func (a *agent) finishMessage(ctx context.Context, msg *message.Message, finishReson message.FinishReason) {
	msg.AddFinish(finishReson)
	_ = a.messages.Update(ctx, *msg)
}

func (a *agent) processEvent(ctx context.Context, sessionID string, assistantMsg *message.Message, event provider.ProviderEvent, events chan<- AgentEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing.
	}

	switch event.Type {
	case provider.EventThinkingDelta:
		// Reasoning is tracked separately in manager.asyncReq.reasoning
		// and saved to message metadata - no need to store in message content
		a.messages.Update(ctx, *assistantMsg)

		// Emit reasoning delta event for real-time streaming
		events <- AgentEvent{
			Type:           AgentEventTypeReasoningDelta,
			ReasoningDelta: event.Thinking,
		}
		return nil
	case provider.EventContentDelta:
		assistantMsg.AppendContent(event.Content)
		a.messages.Update(ctx, *assistantMsg)

		// Emit content delta event for real-time streaming
		events <- AgentEvent{
			Type:         AgentEventTypeContentDelta,
			ContentDelta: event.Content,
		}
		return nil
	case provider.EventContentStart:
		// Content block started
	case provider.EventToolUseStart:
		log.Printf("│ 🔧 Tool call: %s", event.ToolCall.Name)
		assistantMsg.AddToolCall(*event.ToolCall)
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventToolUseStop:
		assistantMsg.FinishToolCall(event.ToolCall.ID)
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventError:
		if errors.Is(event.Error, context.Canceled) {
			log.Printf("│ ⚠️  Canceled")
			return context.Canceled
		}
		log.Printf("│ ❌ Error: %s", event.Error.Error())
		return event.Error
	case provider.EventComplete:
		contentPreview := event.Response.Content
		if len(contentPreview) > 80 {
			contentPreview = contentPreview[:80] + "..."
		}
		log.Println("┌─────────────────────────────────────────────────────────────")
		log.Printf("│ 📥 LLM RESPONSE")
		log.Printf("│    Finish: %s", event.Response.FinishReason)
		log.Printf("│    Tools: %d", len(event.Response.ToolCalls))
		if contentPreview != "" {
			log.Printf("│    Content: %s", contentPreview)
		}
		log.Println("└─────────────────────────────────────────────────────────────")

		assistantMsg.SetToolCalls(event.Response.ToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason)
		if err := a.messages.Update(ctx, *assistantMsg); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		return a.TrackUsage(ctx, sessionID, a.provider.Model(), event.Response.Usage)
	}

	return nil
}

func (a *agent) TrackUsage(ctx context.Context, sessionID string, model models.Model, usage provider.TokenUsage) error {
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
		model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
		model.CostPer1MIn/1e6*float64(usage.InputTokens) +
		model.CostPer1MOut/1e6*float64(usage.OutputTokens)

	sess.Cost += cost
	sess.CompletionTokens = usage.OutputTokens + usage.CacheReadTokens
	sess.PromptTokens = usage.InputTokens + usage.CacheCreationTokens

	_, err = a.sessions.Save(ctx, sess)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

func (a *agent) Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error) {
	if a.IsBusy() {
		return models.Model{}, fmt.Errorf("cannot change model while processing requests")
	}

	if err := config.UpdateAgentModel(agentName, modelID); err != nil {
		return models.Model{}, fmt.Errorf("failed to update config: %w", err)
	}

	toolNames := make([]string, len(a.tools))
	for i, t := range a.tools {
		toolNames[i] = t.Info().Name
	}

	provider, err := createAgentProvider(agentName, toolNames)
	if err != nil {
		return models.Model{}, fmt.Errorf("failed to create provider for model %s: %w", modelID, err)
	}

	a.provider = provider

	return a.provider.Model(), nil
}

// logRequestContext logs a brief summary of the request
func (a *agent) logRequestContext(sessionID string, msgHistory []message.Message) {
	// Count message types
	userMsgs, assistantMsgs, toolMsgs := 0, 0, 0
	for _, msg := range msgHistory {
		switch msg.Role {
		case message.User:
			userMsgs++
		case message.Assistant:
			assistantMsgs++
		case message.Tool:
			toolMsgs++
		}
	}

	// Get last user message preview
	lastUserContent := ""
	for i := len(msgHistory) - 1; i >= 0; i-- {
		if msgHistory[i].Role == message.User {
			lastUserContent = msgHistory[i].Content().String()
			if len(lastUserContent) > 100 {
				lastUserContent = lastUserContent[:100] + "..."
			}
			break
		}
	}

	log.Println("")
	log.Println("┌─────────────────────────────────────────────────────────────")
	log.Printf("│ 📤 SENDING TO LLM")
	log.Printf("│    Messages: %d user, %d assistant, %d tool", userMsgs, assistantMsgs, toolMsgs)
	log.Printf("│    Tokens: ~%d (estimated)", a.estimateTokens(msgHistory))
	log.Printf("│    User: %s", lastUserContent)
	log.Println("└─────────────────────────────────────────────────────────────")
}

// estimateTokens provides a rough token estimate for logging
func (a *agent) estimateTokens(msgs []message.Message) int {
	total := 0
	for _, msg := range msgs {
		// Rough estimate: 1 token per 4 characters
		total += len(msg.Content().String()) / 4
	}
	return total
}

func createAgentProvider(agentName config.AgentName, availableTools []string) (provider.Provider, error) {
	cfg := config.Get()
	agentConfig, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentName)
	}
	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}

	providerCfg, ok := cfg.Providers[model.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not supported", model.Provider)
	}
	if providerCfg.Disabled {
		return nil, fmt.Errorf("provider %s is not enabled", model.Provider)
	}
	maxTokens := model.DefaultMaxTokens
	if agentConfig.MaxTokens > 0 {
		maxTokens = agentConfig.MaxTokens
	}
	opts := []provider.ProviderClientOption{
		provider.WithAPIKey(providerCfg.APIKey),
		provider.WithModel(model),
		provider.WithSystemMessage(prompt.GetAgentPrompt(agentName, model.Provider, availableTools)),
		provider.WithMaxTokens(maxTokens),
	}
	if (model.Provider == models.ProviderOpenAI || model.Provider == models.ProviderGROQ) && model.CanReason {
		reasoningEffort := agentConfig.ReasoningEffort
		if reasoningEffort == "" {
			reasoningEffort = "medium"
		}
		opts = append(
			opts,
			provider.WithOpenAIOptions(
				provider.WithReasoningEffort(reasoningEffort),
			),
		)
	} else if model.Provider == models.ProviderAnthropic && model.CanReason {
		opts = append(
			opts,
			provider.WithAnthropicOptions(
				provider.WithAnthropicShouldThinkFn(provider.DefaultShouldThinkFn),
			),
		)
	}
	agentProvider, err := provider.NewProvider(
		model.Provider,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create provider: %v", err)
	}

	return agentProvider, nil
}
