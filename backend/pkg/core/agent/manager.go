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

// ArticleDraftService provides draft content persistence for the agent.
// A pre-turn snapshot enables "Undo All", and per-edit updates keep the DB in sync.
type ArticleDraftService interface {
	// CreateDraftSnapshot saves the current draft as a version record (for "Undo All").
	// Returns the version ID so the frontend can revert to it on reject.
	CreateDraftSnapshot(ctx context.Context, articleID uuid.UUID) (*uuid.UUID, error)
	// UpdateDraftContent updates only draft_content + updated_at (no version record).
	// Used during agent turns to persist each edit without creating version spam.
	UpdateDraftContent(ctx context.Context, articleID uuid.UUID, htmlContent string) error
}

// AgentAsyncCopilotManager - LLM Agent Framework powered copilot manager
type AgentAsyncCopilotManager struct {
	requests     map[string]*AgentAsyncRequest
	mu           sync.RWMutex
	agent        agent.Service
	sessionSvc   session.Service
	messageSvc   message.Service
	chatService  ChatMessageServiceInterface
	draftService ArticleDraftService // nil-safe: version snapshots and DB persistence are skipped if nil
	config       Config
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
	// Chain of thought step tracking
	steps           []TurnStep // Ordered steps in this turn
	currentStepType string     // "reasoning", "tool", "content", or ""
	currentStepIdx  int        // Index of current step being built
}

// Global singleton for backward compatibility
var (
	globalAgentManager *AgentAsyncCopilotManager
	agentManagerOnce   sync.Once
)

// NewAgentAsyncCopilotManager creates a new agent manager with configuration
func NewAgentAsyncCopilotManager(cfg Config, agentSvc agent.Service, sessionSvc session.Service, messageSvc message.Service, chatService ChatMessageServiceInterface, draftService ArticleDraftService) *AgentAsyncCopilotManager {
	return &AgentAsyncCopilotManager{
		requests:     make(map[string]*AgentAsyncRequest),
		agent:        agentSvc,
		sessionSvc:   sessionSvc,
		messageSvc:   messageSvc,
		chatService:  chatService,
		draftService: draftService,
		config:       cfg,
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

// ExaClient interface combines ExaSearchService and ExaAnswerService
// Satisfied directly by exa.Client - no adapter needed
type ExaClient interface {
	tools.ExaSearchService
	tools.ExaAnswerService
}

// InitializeAgentCopilotManager initializes the agent copilot manager with real services
func InitializeAgentCopilotManager(sourceService tools.ArticleSourceService, chatService ChatMessageServiceInterface, exaClient ExaClient, sourceCreator tools.SourceCreator, draftService ArticleDraftService) error {
	// Load agent configuration
	cfg := LoadConfig()

	// Create session and message services
	sessionSvc := session.NewInMemorySessionService()
	messageSvc := message.NewInMemoryMessageService()

	// Create text generation service for tools that need it
	textGenService := text.NewGenerationService()

	// Create a DraftSaver adapter for the EditTextTool (bridges string articleID to UUID-based ArticleDraftService)
	var draftSaver tools.DraftSaver
	if draftService != nil {
		draftSaver = &draftSaverAdapter{draftService: draftService}
	}

	// Create writing tools for the agent
	writingTools := []tools.BaseTool{
		tools.NewReadDocumentTool(),
		tools.NewEditTextTool(draftSaver),
		tools.NewRewriteSectionTool(draftSaver),
		tools.NewGenerateImagePromptTool(textGenService),
		tools.NewGenerateTextContentTool(textGenService),
	}

	// Add Exa tools if client is provided
	if exaClient != nil && sourceCreator != nil {
		writingTools = append(writingTools,
			tools.NewExaSearchTool(exaClient, sourceCreator),
			tools.NewExaAnswerTool(exaClient),
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
	manager := NewAgentAsyncCopilotManager(cfg, agentSvc, sessionSvc, messageSvc, chatService, draftService)
	SetGlobalAgentManager(manager)

	log.Printf("[Agent] Initialized with configuration (max_concurrent=%d, timeout=%v)", cfg.MaxConcurrentRequests, cfg.RequestTimeout)
	return nil
}

// InitializeWithDefaults initializes the agent copilot manager with default services
// This is a convenience function that initializes without optional services
func InitializeWithDefaults(chatService ChatMessageServiceInterface, draftService ArticleDraftService) error {
	// Initialize without source service and exa client
	// These can be provided via InitializeAgentCopilotManager when available
	return InitializeAgentCopilotManager(nil, chatService, nil, nil, draftService)
}

// draftSaverAdapter bridges the tools.DraftSaver interface (string article IDs)
// to the ArticleDraftService interface (uuid.UUID article IDs).
type draftSaverAdapter struct {
	draftService ArticleDraftService
}

func (a *draftSaverAdapter) UpdateDraftContent(ctx context.Context, articleID string, htmlContent string) error {
	parsed, err := uuid.Parse(articleID)
	if err != nil {
		return fmt.Errorf("invalid article ID %q: %w", articleID, err)
	}
	return a.draftService.UpdateDraftContent(ctx, parsed, htmlContent)
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
	dbMessages, err := m.loadConversationContext(ctx, articleID, 30)
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
	if asyncReq.Request.DocumentContent != "" || asyncReq.Request.DocumentMarkdown != "" {
		ctx = tools.WithDocumentContent(ctx, asyncReq.Request.DocumentContent, asyncReq.Request.DocumentMarkdown)

		// Generate outline from markdown if available, otherwise fall back to HTML
		var layout string
		if asyncReq.Request.DocumentMarkdown != "" {
			layout = generateMarkdownOutline(asyncReq.Request.DocumentMarkdown)
		} else {
			layout = generateHTMLOutline(asyncReq.Request.DocumentContent)
		}
		docLen := len(asyncReq.Request.DocumentMarkdown)
		if docLen == 0 {
			docLen = len(asyncReq.Request.DocumentContent)
		}
		lineCount := strings.Count(asyncReq.Request.DocumentMarkdown, "\n") + 1
		userPrompt += fmt.Sprintf("\n\n--- Document Info: %d chars, %d lines ---\n", docLen, lineCount)
		userPrompt += layout
		log.Printf("[Agent] Document layout generated (%d headers, %d chars, %d lines)", strings.Count(layout, "\n")+1, docLen, lineCount)
	}

	// Create a pre-turn version snapshot so the frontend can "Undo All" agent changes.
	// This is done before the agent runs so the snapshot captures the state before any edits.
	if m.draftService != nil && articleID != uuid.Nil {
		snapshotID, snapErr := m.draftService.CreateDraftSnapshot(ctx, articleID)
		if snapErr != nil {
			log.Printf("[Agent] ⚠️ Failed to create pre-turn snapshot: %v", snapErr)
		} else if snapshotID != nil {
			log.Printf("[Agent] 📸 Created pre-turn snapshot (version %s) for article %s", snapshotID.String(), articleID)
			asyncReq.ResponseChan <- StreamResponse{
				RequestID: asyncReq.ID,
				Type:      StreamTypeTurnStarted,
				Data: map[string]interface{}{
					"snapshot_version_id": snapshotID.String(),
				},
			}
		}
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
			// If current step is not reasoning, start a new reasoning step
			if asyncReq.currentStepType != "reasoning" {
				asyncReq.currentStepIdx = len(asyncReq.steps)
				asyncReq.steps = append(asyncReq.steps, TurnStep{
					Type:      "reasoning",
					Reasoning: &ReasoningStep{Content: "", Visible: true},
				})
				asyncReq.currentStepType = "reasoning"
			}
			// Append to current reasoning step
			asyncReq.steps[asyncReq.currentStepIdx].Reasoning.Content += event.ReasoningDelta

			asyncReq.ResponseChan <- StreamResponse{
				RequestID:       asyncReq.ID,
				Type:            StreamTypeReasoningDelta,
				ThinkingContent: event.ReasoningDelta,
				Iteration:       asyncReq.iteration,
				StepIndex:       asyncReq.currentStepIdx,
			}
		case agent.AgentEventTypeContentDelta:
			// If current step is not content, start a new content step
			if asyncReq.currentStepType != "content" {
				asyncReq.currentStepIdx = len(asyncReq.steps)
				asyncReq.steps = append(asyncReq.steps, TurnStep{
					Type:    "content",
					Content: "",
				})
				asyncReq.currentStepType = "content"
			}
			// Append to current content step
			asyncReq.steps[asyncReq.currentStepIdx].Content += event.ContentDelta

			asyncReq.ResponseChan <- StreamResponse{
				RequestID: asyncReq.ID,
				Type:      "content_delta",
				Content:   event.ContentDelta,
				Iteration: asyncReq.iteration,
				StepIndex: asyncReq.currentStepIdx,
			}
		case agent.AgentEventTypeResponse:
			if event.Message.ID != "" {
				asyncReq.iteration++
				toolCalls := event.Message.ToolCalls()
				
				// Only save message when there are NO tool calls (final response)
				// When there are tool calls, we continue accumulating steps
				if len(toolCalls) == 0 {
					m.saveAssistantMessage(ctx, asyncReq, event.Message, articleID)
				}

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
						var toolInput map[string]interface{}
						if toolCall.Input != "" {
							if err := json.Unmarshal([]byte(toolCall.Input), &toolInput); err != nil {
								toolInput = map[string]interface{}{"raw": toolCall.Input}
							}
						}

						// Create tool step in chain of thought
						asyncReq.currentStepIdx = len(asyncReq.steps)
						asyncReq.steps = append(asyncReq.steps, TurnStep{
							Type: "tool",
							Tool: &ToolStepPayload{
								ToolID:    toolCall.ID,
								ToolName:  toolCall.Name,
								Input:     toolInput,
								Status:    "running",
								StartedAt: time.Now().Format(time.RFC3339),
							},
						})
						asyncReq.currentStepType = "tool"

						asyncReq.ResponseChan <- StreamResponse{
							RequestID: asyncReq.ID,
							Type:      "tool_use",
							Iteration: asyncReq.iteration,
							StepIndex: asyncReq.currentStepIdx,
							ToolID:    toolCall.ID,
							ToolName:  toolCall.Name,
							ToolInput: toolInput,
						}
					}
				} else {
					// Final content response - create content step
					content := event.Message.Content().String()
					if content != "" {
						asyncReq.currentStepIdx = len(asyncReq.steps)
						asyncReq.steps = append(asyncReq.steps, TurnStep{
							Type:    "content",
							Content: content,
						})
						asyncReq.currentStepType = "content"
					}

					asyncReq.ResponseChan <- StreamResponse{
						RequestID: asyncReq.ID,
						Type:      "text",
						Content:   content,
						Iteration: asyncReq.iteration,
						StepIndex: asyncReq.currentStepIdx,
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
					errorStr := ""
					if toolResult.IsError {
						status = "error"
						errorStr = toolResult.Content
					}

					toolCalls = append(toolCalls, ToolCallPayload{
						ID:     toolResult.ToolCallID,
						Name:   toolName,
						Status: status,
						Result: resultData,
					})

					// Update the matching tool step with result
					for i := range asyncReq.steps {
						if asyncReq.steps[i].Type == "tool" && asyncReq.steps[i].Tool != nil &&
							asyncReq.steps[i].Tool.ToolID == toolResult.ToolCallID {
							asyncReq.steps[i].Tool.Status = status
							asyncReq.steps[i].Tool.Output = resultData
							asyncReq.steps[i].Tool.CompletedAt = time.Now().Format(time.RFC3339)
							asyncReq.steps[i].Tool.Error = errorStr
							break
						}
					}
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
					resultToolName := ""
					if !toolResult.IsError {
						var resultData map[string]interface{}
						if err := json.Unmarshal([]byte(toolResult.Content), &resultData); err == nil {
							if tn, ok := resultData["tool_name"].(string); ok {
								resultToolName = tn
								isSearchTool = tn == "search_web_sources" || tn == "ask_question"
							}
						}
					}

					asyncReq.ResponseChan <- StreamResponse{
						RequestID: asyncReq.ID,
						Type:      StreamTypeToolResult,
						Iteration: asyncReq.iteration,
						ToolID:    toolResult.ToolCallID,
						ToolName:  resultToolName,
						ToolResult: map[string]interface{}{
							"content":   toolResult.Content,
							"metadata":  toolResult.Metadata,
							"is_error":  toolResult.IsError,
							"is_search": isSearchTool,
							"tool_name": resultToolName,
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

	// Include chain of thought steps if present
	if len(asyncReq.steps) > 0 {
		log.Printf("[Agent]    Has %d chain of thought steps", len(asyncReq.steps))
		
		// Convert TurnStep to ChainOfThoughtStep
		cotSteps := make([]metadata.ChainOfThoughtStep, len(asyncReq.steps))
		for i, step := range asyncReq.steps {
			cotSteps[i] = metadata.ChainOfThoughtStep{
				Type:    step.Type,
				Content: step.Content,
			}
			if step.Reasoning != nil {
				cotSteps[i].Reasoning = &metadata.ThinkingBlock{
					Content:    step.Reasoning.Content,
					DurationMs: step.Reasoning.DurationMs,
					Visible:    step.Reasoning.Visible,
				}
			}
			if step.Tool != nil {
				cotSteps[i].Tool = &metadata.ToolStepInfo{
					ToolID:      step.Tool.ToolID,
					ToolName:    step.Tool.ToolName,
					Input:       step.Tool.Input,
					Output:      step.Tool.Output,
					Status:      step.Tool.Status,
					Error:       step.Tool.Error,
					StartedAt:   step.Tool.StartedAt,
					CompletedAt: step.Tool.CompletedAt,
					DurationMs:  step.Tool.DurationMs,
				}
			}
		}
		msgMetadata.WithSteps(cotSteps)
		
		// Also set legacy Thinking field for backward compatibility (combine all reasoning)
		var allReasoning string
		for _, step := range asyncReq.steps {
			if step.Type == "reasoning" && step.Reasoning != nil {
				allReasoning += step.Reasoning.Content
			}
		}
		if allReasoning != "" {
			msgMetadata.WithThinking(&metadata.ThinkingBlock{
				Content: allReasoning,
				Visible: true,
			})
		}
		
		// Reset steps for next turn
		asyncReq.steps = nil
		asyncReq.currentStepType = ""
		asyncReq.currentStepIdx = 0
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

		// Don't skip error results -- save them so they appear in conversation history
		isError := toolResult.IsError

		var toolResultData map[string]interface{}
		if err := json.Unmarshal([]byte(toolResult.Content), &toolResultData); err != nil {
			if isError {
				// Error results are often plain text, not JSON -- wrap them
				toolResultData = map[string]interface{}{
					"content":  toolResult.Content,
					"is_error": true,
				}
				log.Printf("[Agent]       ⚠️  Error result (plain text): %s", truncate(toolResult.Content, 100))
			} else {
				log.Printf("[Agent]       ⚠️  Failed to parse tool result: %v", err)
				continue
			}
		}

		toolName, _ := toolResultData["tool_name"].(string)
		log.Printf("[Agent]       Tool Name: %s (isError: %v)", toolName, isError)

		toolExec := &metadata.ToolExecution{
			ToolName:   toolName,
			ToolID:     toolResult.ToolCallID,
			Output:     toolResultData,
			ExecutedAt: time.Now(),
			Success:    !isError,
		}
		msgMetadata.WithToolExecution(toolExec)

		if toolName == "edit_text" || toolName == "rewrite_section" || toolName == "rewrite_document" {
			log.Printf("[Agent]       ✏️  ARTIFACT TOOL DETECTED: %s (isError: %v)", toolName, isError)

			artifactID := uuid.New().String()
			artifactType := metadata.ArtifactTypeCodeEdit
			if toolName == "rewrite_document" {
				artifactType = metadata.ArtifactTypeRewrite
			}

			var artifactContent string
			var diffPreview string
			var description string

			if toolName == "edit_text" || toolName == "rewrite_section" {
				// Both edit_text and rewrite_section use old_str/new_str format
				if newStr, ok := toolResultData["new_str"].(string); ok {
					artifactContent = newStr
				} else if newText, ok := toolResultData["new_text"].(string); ok {
					artifactContent = newText
				}
				if oldStr, ok := toolResultData["old_str"].(string); ok {
					diffPreview = fmt.Sprintf("Old: %s\nNew: %s", truncate(oldStr, 50), truncate(artifactContent, 50))
				} else if oldText, ok := toolResultData["original_text"].(string); ok {
					diffPreview = fmt.Sprintf("Old: %s\nNew: %s", truncate(oldText, 50), truncate(artifactContent, 50))
				}
				if reason, ok := toolResultData["reason"].(string); ok {
					description = reason
				}
				// For error results, use the error content as description if no reason
				if isError && description == "" {
					description = toolResult.Content
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

			artifactStatus := metadata.ArtifactStatusPending
			if isError {
				artifactStatus = "error"
			}

			artifact := &metadata.ArtifactInfo{
				ID:          artifactID,
				Type:        artifactType,
				Status:      artifactStatus,
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

// generateMarkdownOutline extracts headings from markdown to show document structure
func generateMarkdownOutline(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var outline []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Match markdown headings (## Heading, ### Heading, etc.)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Count the heading level
		level := 0
		for _, ch := range trimmed {
			if ch == '#' {
				level++
			} else {
				break
			}
		}
		if level < 1 || level > 6 {
			continue
		}

		headerText := strings.TrimSpace(trimmed[level:])
		if headerText == "" {
			continue
		}

		indent := ""
		if level > 2 {
			indent = strings.Repeat("  ", level-2)
		}

		outline = append(outline, fmt.Sprintf("%s- %s (line %d)", indent, headerText, i+1))
	}

	if len(outline) == 0 {
		return "(empty document)"
	}

	return strings.Join(outline, "\n")
}

// generateHTMLOutline extracts only headers from HTML to show document structure
// Returns a tree-like layout with line numbers for navigation
func generateHTMLOutline(html string) string {
	lines := strings.Split(html, "\n")
	var outline []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Extract header level and text
		var level int
		var headerText string
		
		if strings.HasPrefix(trimmed, "<h1") {
			level = 1
		} else if strings.HasPrefix(trimmed, "<h2") {
			level = 2
		} else if strings.HasPrefix(trimmed, "<h3") {
			level = 3
		} else if strings.HasPrefix(trimmed, "<h4") {
			level = 4
		} else if strings.HasPrefix(trimmed, "<h5") {
			level = 5
		} else if strings.HasPrefix(trimmed, "<h6") {
			level = 6
		} else {
			continue
		}
		
		// Extract text content from header tag
		headerText = extractHeaderText(trimmed)
		if headerText == "" {
			continue
		}
		
		// Create indentation based on level (h2 = no indent, h3 = 2 spaces, etc.)
		indent := ""
		if level > 2 {
			indent = strings.Repeat("  ", level-2)
		}
		
		outline = append(outline, fmt.Sprintf("%s- %s (line %d)", indent, headerText, i+1))
	}

	if len(outline) == 0 {
		return "(empty document)"
	}

	return strings.Join(outline, "\n")
}

// extractHeaderText removes HTML tags and returns just the text content
func extractHeaderText(header string) string {
	// Find the closing > of the opening tag
	start := strings.Index(header, ">")
	if start == -1 {
		return ""
	}
	// Find the opening < of the closing tag
	end := strings.LastIndex(header, "</")
	if end == -1 {
		end = len(header)
	}
	
	text := header[start+1 : end]
	// Clean up any nested tags (like <strong>, <em>, etc.)
	text = stripHTMLTags(text)
	return strings.TrimSpace(text)
}

// stripHTMLTags removes all HTML tags from a string
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
