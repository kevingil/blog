package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"backend/pkg/core/agent/metadata"
	"backend/pkg/core/ml/llm/agent"
	"backend/pkg/core/ml/llm/config"
	"backend/pkg/core/ml/llm/message"
	"backend/pkg/core/ml/llm/session"
	"backend/pkg/core/ml/llm/tools"
	"backend/pkg/core/ml/text"
	"backend/pkg/database/models"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/google/uuid"
)

// ChatMessageServiceInterface defines the interface for chat message operations
// This interface is satisfied by core/chat.MessageService
type ChatMessageServiceInterface interface {
	SaveMessage(ctx context.Context, articleID uuid.UUID, role, content string, metaData *metadata.MessageMetaData) (*models.ChatMessage, error)
	GetConversationHistory(ctx context.Context, articleID uuid.UUID, limit int) ([]models.ChatMessage, error)
}

// SourceServiceInterface defines the interface for source operations
// This interface is satisfied by core/source functions
type SourceServiceInterface interface {
	Create(ctx context.Context, req interface{}) (interface{}, error)
	GetByArticleID(ctx context.Context, articleID uuid.UUID) (interface{}, error)
	SearchSimilar(ctx context.Context, articleID uuid.UUID, query string, limit int) (interface{}, error)
}

// ExaServiceInterface defines the interface for Exa search operations
type ExaServiceInterface interface {
	Search(ctx context.Context, query string, options map[string]interface{}) (interface{}, error)
	Answer(ctx context.Context, question string) (interface{}, error)
}

// AgentAsyncCopilotManager - LLM Agent Framework powered copilot manager
type AgentAsyncCopilotManager struct {
	requests    map[string]*AgentAsyncRequest
	mu          sync.RWMutex
	agent       agent.Service
	sessionSvc  session.Service
	messageSvc  message.Service
	chatService ChatMessageServiceInterface
	config      Config
}

// AgentAsyncRequest represents an async chat request
type AgentAsyncRequest struct {
	ID           string
	Request      ChatRequest
	Status       string
	StartTime    time.Time
	ResponseChan chan StreamResponse
	ctx          context.Context
	cancel       context.CancelFunc
	SessionID    string
	iteration    int
	reasoning    string // Accumulated reasoning content from reasoning models
}

// Global singleton for backward compatibility
var (
	globalAgentManager *AgentAsyncCopilotManager
	agentManagerOnce   sync.Once
)

// NewAgentAsyncCopilotManager creates a new agent manager with configuration
func NewAgentAsyncCopilotManager(cfg Config, agentSvc agent.Service, sessionSvc session.Service, messageSvc message.Service, chatService ChatMessageServiceInterface) *AgentAsyncCopilotManager {
	return &AgentAsyncCopilotManager{
		requests:    make(map[string]*AgentAsyncRequest),
		agent:       agentSvc,
		sessionSvc:  sessionSvc,
		messageSvc:  messageSvc,
		chatService: chatService,
		config:      cfg,
	}
}

// GetAgentAsyncCopilotManager returns the singleton agent-based async manager
func GetAgentAsyncCopilotManager() *AgentAsyncCopilotManager {
	if globalAgentManager == nil {
		globalAgentManager = &AgentAsyncCopilotManager{
			requests: make(map[string]*AgentAsyncRequest),
			config:   LoadConfig(),
		}
	}
	return globalAgentManager
}

// SetGlobalAgentManager sets the global agent manager instance
func SetGlobalAgentManager(manager *AgentAsyncCopilotManager) {
	globalAgentManager = manager
}

// ExaAdapter is a combined adapter for Exa services
type ExaAdapter interface {
	tools.ExaSearchService
	tools.ExaAnswerService
}

// InitializeAgentCopilotManager initializes the agent copilot manager with real services
func InitializeAgentCopilotManager(sourceService tools.ArticleSourceService, chatService ChatMessageServiceInterface, exaAdapter ExaAdapter, sourceServiceAdapter tools.ExaSourceService) error {
	// Load agent configuration
	cfg := LoadConfig()

	// Create session and message services
	sessionSvc := session.NewInMemorySessionService()
	messageSvc := message.NewInMemoryMessageService()

	// Create text generation service for tools that need it
	textGenService := text.NewGenerationService()

	// Create writing tools for the agent
	writingTools := []tools.BaseTool{
		tools.NewReadDocumentTool(),
		tools.NewEditTextTool(),
		tools.NewAnalyzeDocumentTool(),
		tools.NewGenerateImagePromptTool(textGenService),
		tools.NewGenerateTextContentTool(textGenService),
	}

	// Add Exa tools if adapter is provided
	if exaAdapter != nil && sourceServiceAdapter != nil {
		writingTools = append(writingTools,
			tools.NewExaSearchTool(exaAdapter, sourceServiceAdapter),
			tools.NewExaAnswerTool(exaAdapter),
		)
	}

	// Add source-related tools if source service is available
	if sourceService != nil {
		writingTools = append(writingTools,
			tools.NewGetRelevantSourcesTool(sourceService),
			tools.NewAddContextFromSourcesTool(sourceService),
		)
	}

	// Create the agent using the LLM framework
	agentSvc, err := agent.NewAgent(
		config.AgentCopilot,
		sessionSvc,
		messageSvc,
		writingTools,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Create and set the global manager with configuration
	manager := NewAgentAsyncCopilotManager(cfg, agentSvc, sessionSvc, messageSvc, chatService)
	SetGlobalAgentManager(manager)

	log.Printf("[Agent] Initialized with configuration (max_concurrent=%d, timeout=%v)", cfg.MaxConcurrentRequests, cfg.RequestTimeout)
	return nil
}

// InitializeWithDefaults initializes the agent copilot manager with default services
// This is a convenience function that creates all necessary adapters
func InitializeWithDefaults(chatService ChatMessageServiceInterface) error {
	// For now, we initialize without source service and exa adapters
	// These can be added later when proper adapters are set up
	return InitializeAgentCopilotManager(nil, chatService, nil, nil)
}

func (m *AgentAsyncCopilotManager) SubmitChatRequest(req ChatRequest) (string, error) {
	if req.Message == "" {
		return "", errors.New("message is required")
	}

	if req.ArticleID == "" {
		return "", errors.New("articleId is required")
	}

	if m.agent == nil {
		return "", errors.New("agent service not initialized")
	}

	// Check concurrent request limit
	m.mu.RLock()
	currentRequests := len(m.requests)
	m.mu.RUnlock()

	if currentRequests >= m.config.MaxConcurrentRequests {
		return "", fmt.Errorf("maximum concurrent requests reached (%d)", m.config.MaxConcurrentRequests)
	}

	requestID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), m.config.RequestTimeout)

	asyncReq := &AgentAsyncRequest{
		ID:           requestID,
		Request:      req,
		Status:       "processing",
		StartTime:    time.Now(),
		ResponseChan: make(chan StreamResponse, m.config.ChannelBuffer),
		ctx:          ctx,
		cancel:       cancel,
		SessionID:    requestID,
	}

	m.mu.Lock()
	m.requests[requestID] = asyncReq
	m.mu.Unlock()

	go m.processAgentRequest(asyncReq)

	return requestID, nil
}

func (m *AgentAsyncCopilotManager) GetResponseChannel(requestID string) (<-chan StreamResponse, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, exists := m.requests[requestID]
	if !exists {
		return nil, false
	}

	return req.ResponseChan, true
}

// Shutdown gracefully shuts down the agent manager
func (m *AgentAsyncCopilotManager) Shutdown(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	log.Printf("[Agent] Shutting down, waiting for %d in-flight requests...", m.ActiveRequests())

	m.mu.RLock()
	for _, req := range m.requests {
		req.cancel()
	}
	m.mu.RUnlock()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if m.ActiveRequests() == 0 {
			log.Printf("[Agent] All requests completed, shutdown successful")
			return nil
		}
		<-ticker.C
	}

	log.Printf("[Agent] Shutdown timeout reached, %d requests still active", m.ActiveRequests())
	return fmt.Errorf("shutdown timeout: %d requests still active", m.ActiveRequests())
}

// ActiveRequests returns the number of active requests
func (m *AgentAsyncCopilotManager) ActiveRequests() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.requests)
}

// CancelRequest cancels a specific request by ID
func (m *AgentAsyncCopilotManager) CancelRequest(requestID string) error {
	m.mu.RLock()
	req, exists := m.requests[requestID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("request not found")
	}

	req.cancel()
	return nil
}

func (m *AgentAsyncCopilotManager) processAgentRequest(asyncReq *AgentAsyncRequest) {
	defer func() {
		asyncReq.cancel()
		close(asyncReq.ResponseChan)

		time.AfterFunc(m.config.CleanupDelay, func() {
			m.mu.Lock()
			delete(m.requests, asyncReq.ID)
			m.mu.Unlock()
		})
	}()

	log.Printf("[Agent] Starting request %s", asyncReq.ID)

	timeoutCtx, timeoutCancel := context.WithCancel(asyncReq.ctx)
	defer timeoutCancel()

	go m.monitorTimeout(timeoutCtx, asyncReq)

	sess, err := m.sessionSvc.Create(asyncReq.ctx, "Writing Copilot Session")
	if err != nil {
		log.Printf("[Agent] Failed to create session for request %s: %v", asyncReq.ID, err)
		asyncReq.ResponseChan <- StreamResponse{
			RequestID: asyncReq.ID,
			Type:      "error",
			Error:     "Failed to create session: " + err.Error(),
			Done:      true,
		}
		return
	}

	ctx := asyncReq.ctx
	var articleID uuid.UUID
	if asyncReq.Request.ArticleID != "" {
		ctx = tools.WithArticleID(ctx, asyncReq.Request.ArticleID)
		if parsedID, err := uuid.Parse(asyncReq.Request.ArticleID); err == nil {
			articleID = parsedID
		}
	}

	log.Printf("[Agent] Loading conversation context from database...")
	dbMessages, err := m.loadConversationContext(ctx, articleID, 12)
	if err != nil {
		log.Printf("[Agent] Failed to load conversation context: %v", err)
		dbMessages = []message.Message{}
	}
	log.Printf("[Agent] ✅ Loaded %d messages from database as context", len(dbMessages))

	for _, msg := range dbMessages {
		_, err := m.messageSvc.Create(ctx, sess.ID, message.CreateMessageParams{
			Role:  msg.Role,
			Parts: msg.Parts,
			Model: "loaded",
		})
		if err != nil {
			log.Printf("[Agent] Warning: Failed to add loaded message to session: %v", err)
		}
	}

	log.Printf("[Agent] 📝 Saving NEW user message to database...")

	msgContext := metadata.NewMessageContext(
		asyncReq.Request.ArticleID,
		sess.ID,
		asyncReq.ID,
		"",
	)

	msgMetadata := metadata.BuildMetaData().WithContext(msgContext)

	savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "user", asyncReq.Request.Message, msgMetadata)
	if err != nil {
		log.Printf("[Agent] ❌ Failed to save user message to database: %v", err)
	} else {
		log.Printf("[Agent] ✅ Saved user message (ID: %s) to database for article %s", savedMsg.ID, articleID)
	}

	userPrompt := asyncReq.Request.Message
	if asyncReq.Request.DocumentContent != "" {
		ctx = tools.WithDocumentContent(ctx, asyncReq.Request.DocumentContent, "")
		outline := generateHTMLOutline(asyncReq.Request.DocumentContent)
		userPrompt += "\n\n--- Document Outline (use read_document for full content) ---\n" + outline
		log.Printf("[Agent] Document outline generated (%d lines), full HTML content stored in context", strings.Count(outline, "\n")+1)
	}

	asyncReq.iteration = 1

	resultChan, err := m.agent.Run(ctx, sess.ID, userPrompt)

	startTime := time.Now()

	if err != nil {
		log.Printf("[Agent] Failed to start request %s: %v", asyncReq.ID, err)
		asyncReq.ResponseChan <- StreamResponse{
			RequestID: asyncReq.ID,
			Type:      "error",
			Error:     "Failed to start agent: " + err.Error(),
			Done:      true,
		}
		return
	}

	for event := range resultChan {
		if event.Error != nil {
			log.Printf("[Agent] Error processing request %s: %v", asyncReq.ID, event.Error)
			asyncReq.ResponseChan <- StreamResponse{
				RequestID: asyncReq.ID,
				Type:      "error",
				Error:     event.Error.Error(),
				Done:      true,
			}
			return
		}

		switch event.Type {
		case agent.AgentEventTypeThinking:
			asyncReq.ResponseChan <- StreamResponse{
				RequestID:       asyncReq.ID,
				Type:            "thinking",
				ThinkingMessage: event.ThinkingMessage,
				Iteration:       event.Iteration,
			}
		case agent.AgentEventTypeReasoningDelta:
			// Accumulate reasoning for persistence
			asyncReq.reasoning += event.ReasoningDelta
			asyncReq.ResponseChan <- StreamResponse{
				RequestID:       asyncReq.ID,
				Type:            StreamTypeReasoningDelta,
				ThinkingContent: event.ReasoningDelta,
				Iteration:       asyncReq.iteration,
			}
		case agent.AgentEventTypeContentDelta:
			asyncReq.ResponseChan <- StreamResponse{
				RequestID: asyncReq.ID,
				Type:      "content_delta",
				Content:   event.ContentDelta,
				Iteration: asyncReq.iteration,
			}
		case agent.AgentEventTypeResponse:
			if event.Message.ID != "" {
				asyncReq.iteration++
				m.saveAssistantMessage(ctx, asyncReq, event.Message, articleID)

				toolCalls := event.Message.ToolCalls()
				if len(toolCalls) > 0 {
					textContent := event.Message.Content().String()
					if textContent != "" {
						asyncReq.ResponseChan <- StreamResponse{
							RequestID: asyncReq.ID,
							Type:      "text",
							Content:   textContent,
							Iteration: asyncReq.iteration,
						}
					}

					for _, toolCall := range toolCalls {
						var toolInput interface{}
						if toolCall.Input != "" {
							var jsonInput map[string]interface{}
							if err := json.Unmarshal([]byte(toolCall.Input), &jsonInput); err == nil {
								toolInput = jsonInput
							} else {
								toolInput = toolCall.Input
							}
						}

						asyncReq.ResponseChan <- StreamResponse{
							RequestID: asyncReq.ID,
							Type:      "tool_use",
							Iteration: asyncReq.iteration,
							ToolID:    toolCall.ID,
							ToolName:  toolCall.Name,
							ToolInput: toolInput,
						}
					}
				} else {
					asyncReq.ResponseChan <- StreamResponse{
						RequestID: asyncReq.ID,
						Type:      "text",
						Content:   event.Message.Content().String(),
						Iteration: asyncReq.iteration,
					}
				}
			}
		case agent.AgentEventTypeTool:
			if event.Message.ID != "" {
				toolResults := event.Message.ToolResults()

				groupID := uuid.New().String()
				toolCalls := make([]ToolCallPayload, 0, len(toolResults))

				for _, toolResult := range toolResults {
					var resultData map[string]interface{}
					var toolName string
					if !toolResult.IsError {
						if err := json.Unmarshal([]byte(toolResult.Content), &resultData); err == nil {
							if name, ok := resultData["tool_name"].(string); ok {
								toolName = name
							}
						}
					}

					status := "completed"
					if toolResult.IsError {
						status = "error"
					}

					toolCalls = append(toolCalls, ToolCallPayload{
						ID:     toolResult.ToolCallID,
						Name:   toolName,
						Status: status,
						Result: resultData,
					})
				}

				savedMsg := m.saveToolResultMessage(ctx, asyncReq, event.Message, toolResults, articleID)
				if savedMsg != nil {
					var metaDataMap map[string]interface{}
					if err := json.Unmarshal(savedMsg.MetaData, &metaDataMap); err != nil {
						log.Printf("[Agent] ⚠️ Failed to unmarshal meta_data for streaming: %v", err)
						metaDataMap = make(map[string]interface{})
					}

					asyncReq.ResponseChan <- StreamResponse{
						RequestID: asyncReq.ID,
						Type:      StreamTypeFullMessage,
						Iteration: asyncReq.iteration,
						FullMessage: &FullMessagePayload{
							ID:        savedMsg.ID.String(),
							ArticleID: savedMsg.ArticleID.String(),
							Role:      savedMsg.Role,
							Content:   savedMsg.Content,
							MetaData:  metaDataMap,
							CreatedAt: savedMsg.CreatedAt.Format(time.RFC3339),
						},
					}
				}

				asyncReq.ResponseChan <- StreamResponse{
					RequestID: asyncReq.ID,
					Type:      StreamTypeToolGroupComplete,
					Iteration: asyncReq.iteration,
					ToolGroup: &ToolGroupPayload{
						GroupID: groupID,
						Status:  "completed",
						Calls:   toolCalls,
					},
				}

				for _, toolResult := range toolResults {
					isSearchTool := false
					if !toolResult.IsError {
						var resultData map[string]interface{}
						if err := json.Unmarshal([]byte(toolResult.Content), &resultData); err == nil {
							if toolName, ok := resultData["tool_name"].(string); ok {
								isSearchTool = toolName == "search_web_sources" || toolName == "ask_question"
							}
						}
					}

					asyncReq.ResponseChan <- StreamResponse{
						RequestID: asyncReq.ID,
						Type:      StreamTypeToolResult,
						Iteration: asyncReq.iteration,
						ToolID:    toolResult.ToolCallID,
						ToolResult: map[string]interface{}{
							"content":   toolResult.Content,
							"metadata":  toolResult.Metadata,
							"is_error":  toolResult.IsError,
							"is_search": isSearchTool,
						},
					}
				}
			}
		case agent.AgentEventTypeError:
			// Error is already handled above
		default:
			log.Println("Unknown event type", event.Type)
		}
	}

	asyncReq.ResponseChan <- StreamResponse{
		RequestID: asyncReq.ID,
		Type:      "done",
		Done:      true,
	}

	duration := time.Since(startTime)
	log.Printf("[Agent] Completed request %s in %v", asyncReq.ID, duration)
}

func (m *AgentAsyncCopilotManager) loadConversationContext(ctx context.Context, articleID uuid.UUID, limit int) ([]message.Message, error) {
	if m.chatService == nil {
		return []message.Message{}, nil
	}

	dbMessages, err := m.chatService.GetConversationHistory(ctx, articleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation history: %w", err)
	}

	log.Printf("[Agent] 📚 Reconstructing %d messages from database metadata...", len(dbMessages))

	agentMessages := make([]message.Message, 0, len(dbMessages))

	for i, dbMsg := range dbMessages {
		var role message.Role
		switch dbMsg.Role {
		case "user":
			role = message.User
		case "assistant":
			role = message.Assistant
		case "tool":
			role = message.Tool
		default:
			role = message.User
		}

		parts := []message.ContentPart{
			message.TextContent{Text: dbMsg.Content},
		}

		msg := message.Message{
			ID:        dbMsg.ID.String(),
			Role:      role,
			Parts:     parts,
			SessionID: "",
		}

		if len(dbMsg.MetaData) > 2 {
			var metaData metadata.MessageMetaData
			if err := json.Unmarshal(dbMsg.MetaData, &metaData); err == nil {

				if metaData.ToolExecution != nil {
					log.Printf("[Agent]    [%d] Reconstructing tool call: %s", i+1, metaData.ToolExecution.ToolName)

					inputJSON, _ := json.Marshal(metaData.ToolExecution.Input)
					toolCall := message.ToolCall{
						ID:       metaData.ToolExecution.ToolID,
						Name:     metaData.ToolExecution.ToolName,
						Input:    string(inputJSON),
						Finished: metaData.ToolExecution.Success,
					}
					msg.AddToolCall(toolCall)
					msg.FinishToolCall(toolCall.ID)
				}

				if metaData.Artifact != nil {
					log.Printf("[Agent]    [%d] Reconstructing artifact: %s (%s)", i+1, metaData.Artifact.Type, metaData.Artifact.Status)

					if metaData.ToolExecution != nil && metaData.ToolExecution.Output != nil {
						outputJSON, _ := json.Marshal(metaData.ToolExecution.Output)
						toolResult := message.ToolResult{
							ToolCallID: metaData.ToolExecution.ToolID,
							Content:    string(outputJSON),
							IsError:    !metaData.ToolExecution.Success,
						}

						toolMsg := message.Message{
							Role: message.Tool,
							Parts: []message.ContentPart{
								toolResult,
							},
						}
						agentMessages = append(agentMessages, msg)
						agentMessages = append(agentMessages, toolMsg)
						continue
					}
				}
			}
		}

		agentMessages = append(agentMessages, msg)
	}

	log.Printf("[Agent] ✅ Reconstructed %d messages (%d from DB)", len(agentMessages), len(dbMessages))

	return agentMessages, nil
}

func (m *AgentAsyncCopilotManager) saveAssistantMessage(ctx context.Context, asyncReq *AgentAsyncRequest, msg message.Message, articleID uuid.UUID) {
	if m.chatService == nil || articleID == uuid.Nil {
		return
	}

	content := msg.Content().String()

	log.Printf("[Agent] 💾 Saving assistant message...")

	msgContext := metadata.NewMessageContext(
		asyncReq.Request.ArticleID,
		asyncReq.SessionID,
		asyncReq.ID,
		"",
	)

	msgMetadata := metadata.BuildMetaData().WithContext(msgContext)

	// Include reasoning/thinking content if present
	if asyncReq.reasoning != "" {
		log.Printf("[Agent]    Has reasoning content: %d chars", len(asyncReq.reasoning))
		msgMetadata.WithThinking(&metadata.ThinkingBlock{
			Content: asyncReq.reasoning,
			Visible: true,
		})
		// Reset reasoning for next turn
		asyncReq.reasoning = ""
	}

	toolCalls := msg.ToolCalls()
	if len(toolCalls) > 0 {
		log.Printf("[Agent]    Has %d tool call(s): %v", len(toolCalls), toolCalls[0].Name)

		toolCall := toolCalls[0]

		var toolInput interface{}
		if toolCall.Input != "" {
			var jsonInput map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Input), &jsonInput); err == nil {
				toolInput = jsonInput
			} else {
				toolInput = toolCall.Input
			}
		}

		toolExec := &metadata.ToolExecution{
			ToolName:   toolCall.Name,
			ToolID:     toolCall.ID,
			Input:      toolInput,
			ExecutedAt: time.Now(),
			Success:    toolCall.Finished,
		}
		msgMetadata.WithToolExecution(toolExec)
	}

	savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "assistant", content, msgMetadata.Build())
	if err != nil {
		log.Printf("[Agent] ❌ Failed to save assistant message to database: %v", err)
	} else {
		log.Printf("[Agent] ✅ Saved assistant message (ID: %s) to database", savedMsg.ID)
	}
}

func (m *AgentAsyncCopilotManager) monitorTimeout(ctx context.Context, asyncReq *AgentAsyncRequest) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	lastActivityTime := time.Now()
	updateCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updateCount++
			elapsed := time.Since(lastActivityTime)

			if elapsed > 1*time.Minute && elapsed < 2*time.Minute {
				asyncReq.ResponseChan <- StreamResponse{
					RequestID:       asyncReq.ID,
					Type:            "thinking",
					ThinkingMessage: "Still working on your request...",
					Iteration:       updateCount,
				}
			}

			if elapsed > 2*time.Minute {
				asyncReq.ResponseChan <- StreamResponse{
					RequestID:       asyncReq.ID,
					Type:            "thinking",
					ThinkingMessage: "This is taking longer than expected. You can wait or cancel the request.",
					Iteration:       updateCount,
				}
			}
		}
	}
}

func (m *AgentAsyncCopilotManager) saveToolResultMessage(ctx context.Context, asyncReq *AgentAsyncRequest, msg message.Message, toolResults []message.ToolResult, articleID uuid.UUID) *models.ChatMessage {
	if m.chatService == nil || articleID == uuid.Nil {
		return nil
	}

	log.Printf("[Agent] 🔧 Processing %d tool result(s) for database save...", len(toolResults))

	msgContext := metadata.NewMessageContext(
		asyncReq.Request.ArticleID,
		asyncReq.SessionID,
		asyncReq.ID,
		"",
	)

	msgMetadata := metadata.BuildMetaData().WithContext(msgContext)

	for idx, toolResult := range toolResults {
		log.Printf("[Agent]    Tool Result #%d:", idx+1)
		log.Printf("[Agent]       Call ID: %s", toolResult.ToolCallID)
		log.Printf("[Agent]       Is Error: %v", toolResult.IsError)

		if toolResult.IsError {
			log.Printf("[Agent]       ⚠️  Skipping error result")
			continue
		}

		var toolResultData map[string]interface{}
		if err := json.Unmarshal([]byte(toolResult.Content), &toolResultData); err != nil {
			log.Printf("[Agent]       ⚠️  Failed to parse tool result: %v", err)
			continue
		}

		toolName, _ := toolResultData["tool_name"].(string)
		log.Printf("[Agent]       Tool Name: %s", toolName)

		toolExec := &metadata.ToolExecution{
			ToolName:   toolName,
			ToolID:     toolResult.ToolCallID,
			Output:     toolResultData,
			ExecutedAt: time.Now(),
			Success:    true,
		}
		msgMetadata.WithToolExecution(toolExec)

		if toolName == "edit_text" || toolName == "rewrite_document" {
			log.Printf("[Agent]       ✏️  ARTIFACT TOOL DETECTED")

			artifactID := uuid.New().String()
			artifactType := metadata.ArtifactTypeCodeEdit
			if toolName == "rewrite_document" {
				artifactType = metadata.ArtifactTypeRewrite
			}

			var artifactContent string
			var diffPreview string
			var description string

			if toolName == "edit_text" {
				if newText, ok := toolResultData["new_text"].(string); ok {
					artifactContent = newText
				}
				if oldText, ok := toolResultData["original_text"].(string); ok {
					diffPreview = fmt.Sprintf("Old: %s\nNew: %s", truncate(oldText, 50), truncate(artifactContent, 50))
				}
				if reason, ok := toolResultData["reason"].(string); ok {
					description = reason
				}
			} else if toolName == "rewrite_document" {
				if newContent, ok := toolResultData["new_content"].(string); ok {
					artifactContent = newContent
				}
				if originalContent, ok := toolResultData["original_content"].(string); ok {
					diffPreview = fmt.Sprintf("Original: %s\nNew: %s", truncate(originalContent, 50), truncate(artifactContent, 50))
				}
				if reason, ok := toolResultData["reason"].(string); ok {
					description = reason
				}
			}

			artifact := &metadata.ArtifactInfo{
				ID:          artifactID,
				Type:        artifactType,
				Status:      metadata.ArtifactStatusPending,
				Content:     artifactContent,
				DiffPreview: diffPreview,
				Title:       fmt.Sprintf("%s result", toolName),
				Description: description,
			}

			msgMetadata.WithArtifact(artifact)

			content := fmt.Sprintf("📋 %s: %s", toolName, description)
			savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "assistant", content, msgMetadata)
			if err != nil {
				log.Printf("[Agent] ❌ Failed to save tool result message with artifact: %v", err)
				return nil
			}
			log.Printf("[Agent] ✅ Saved artifact message (ID: %s) with status: %s", savedMsg.ID, metadata.ArtifactStatusPending)
			return savedMsg
		}

		if toolName == "search_web_sources" {
			log.Printf("[Agent]       🔍 SEARCH TOOL DETECTED")

			totalFound := 0
			sourcesCreated := 0
			if val, ok := toolResultData["total_found"].(float64); ok {
				totalFound = int(val)
			}
			if val, ok := toolResultData["sources_successful"].(float64); ok {
				sourcesCreated = int(val)
			}

			content := fmt.Sprintf("🔍 Web search completed: Found %d results, created %d sources", totalFound, sourcesCreated)
			savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "assistant", content, msgMetadata)
			if err != nil {
				log.Printf("[Agent] ❌ Failed to save search tool result message: %v", err)
				return nil
			}
			log.Printf("[Agent] ✅ Saved search result message (ID: %s)", savedMsg.ID)
			return savedMsg
		}

		if toolName == "ask_question" {
			log.Printf("[Agent]       ❓ ASK QUESTION TOOL DETECTED")

			citationCount := 0
			if citations, ok := toolResultData["citations"].([]interface{}); ok {
				citationCount = len(citations)
			}

			content := fmt.Sprintf("❓ Question answered with %d citations", citationCount)
			savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "assistant", content, msgMetadata)
			if err != nil {
				log.Printf("[Agent] ❌ Failed to save ask_question tool result message: %v", err)
				return nil
			}
			log.Printf("[Agent] ✅ Saved ask_question result message (ID: %s)", savedMsg.ID)
			return savedMsg
		}
	}
	return nil
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func convertHTMLToMarkdown(html string) (string, error) {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}
	return markdown, nil
}

func generateHTMLOutline(html string) string {
	lines := strings.Split(html, "\n")
	var outline []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<h1") || strings.HasPrefix(trimmed, "<h2") ||
			strings.HasPrefix(trimmed, "<h3") || strings.HasPrefix(trimmed, "<h4") ||
			strings.HasPrefix(trimmed, "<h5") || strings.HasPrefix(trimmed, "<h6") {
			outline = append(outline, fmt.Sprintf("%4d| %s", i+1, line))
		} else if strings.HasPrefix(trimmed, "<p") && len(trimmed) > 60 {
			preview := trimmed
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			outline = append(outline, fmt.Sprintf("%4d| %s", i+1, preview))
		}
	}

	if len(outline) == 0 {
		return "(empty document)"
	}

	return strings.Join(outline, "\n")
}
