package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	agentTypes "blog-agent-go/backend/internal/core/agent"
	"blog-agent-go/backend/internal/core/agent/metadata"
	"blog-agent-go/backend/internal/core/ml/llm/agent"
	"blog-agent-go/backend/internal/core/ml/llm/config"
	"blog-agent-go/backend/internal/core/ml/llm/message"
	"blog-agent-go/backend/internal/core/ml/llm/session"
	"blog-agent-go/backend/internal/core/ml/llm/tools"
	"blog-agent-go/backend/internal/models"

	"github.com/google/uuid"
)

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// """
// Python Reference
// we want to create our own endpoints instead of using CopilotKit APIs
// """

// Type aliases for backward compatibility
type ChatMessage = agentTypes.ChatMessage
type ChatRequest = agentTypes.ChatRequest
type ChatRequestResponse = agentTypes.ChatRequestResponse
type StreamResponse = agentTypes.StreamResponse
type ArtifactUpdate = agentTypes.ArtifactUpdate

// AgentAsyncCopilotManager - LLM Agent Framework powered copilot manager
type AgentAsyncCopilotManager struct {
	requests    map[string]*AgentAsyncRequest
	mu          sync.RWMutex
	agent       agent.Service
	sessionSvc  session.Service
	messageSvc  message.Service
	chatService ChatMessageServiceInterface
	config      agentTypes.Config
}

// ChatMessageServiceInterface defines the interface for chat message operations
// This interface is satisfied by core/chat.MessageService
type ChatMessageServiceInterface interface {
	SaveMessage(ctx context.Context, articleID uuid.UUID, role, content string, metaData *metadata.MessageMetaData) (*models.ChatMessage, error)
	GetConversationHistory(ctx context.Context, articleID uuid.UUID, limit int) ([]models.ChatMessage, error)
}

type AgentAsyncRequest struct {
	ID           string
	Request      ChatRequest
	Status       string
	StartTime    time.Time
	ResponseChan chan StreamResponse
	ctx          context.Context
	cancel       context.CancelFunc
	SessionID    string
	iteration    int // Track iteration number for message blocks
}

// Global singleton for backward compatibility
var (
	globalAgentManager *AgentAsyncCopilotManager
	agentManagerOnce   sync.Once
)

// NewAgentAsyncCopilotManager creates a new agent manager with configuration
func NewAgentAsyncCopilotManager(cfg agentTypes.Config, agentSvc agent.Service, sessionSvc session.Service, messageSvc message.Service, chatService ChatMessageServiceInterface) *AgentAsyncCopilotManager {
	return &AgentAsyncCopilotManager{
		requests:    make(map[string]*AgentAsyncRequest),
		agent:       agentSvc,
		sessionSvc:  sessionSvc,
		messageSvc:  messageSvc,
		chatService: chatService,
		config:      cfg,
	}
}

// GetAgentAsyncCopilotManager returns the singleton agent-based async manager (deprecated)
// Kept for backward compatibility. New code should use NewAgentAsyncCopilotManager.
func GetAgentAsyncCopilotManager() *AgentAsyncCopilotManager {
	if globalAgentManager == nil {
		// Initialize with default config if not set
		globalAgentManager = &AgentAsyncCopilotManager{
			requests: make(map[string]*AgentAsyncRequest),
			config:   agentTypes.LoadConfig(),
		}
	}
	return globalAgentManager
}

// SetGlobalAgentManager sets the global agent manager instance
func SetGlobalAgentManager(manager *AgentAsyncCopilotManager) {
	globalAgentManager = manager
}

// InitializeAgentCopilotManager initializes the agent copilot manager with real services
func InitializeAgentCopilotManager(articleSourceService *ArticleSourceService, chatService ChatMessageServiceInterface) error {
	// Load agent configuration
	cfg := agentTypes.LoadConfig()

	// Create session and message services
	sessionSvc := session.NewInMemorySessionService()
	messageSvc := message.NewInMemoryMessageService()

	// Use the real service directly - it already implements the interface we need
	var sourceService tools.ArticleSourceService = nil
	if articleSourceService != nil {
		sourceService = articleSourceService
	}

	// Create text generation service for tools that need it
	textGenService := NewTextGenerationService()

	// Create Exa search service and adapter for web search capabilities
	exaService := NewExaSearchService()
	exaAdapter := NewExaServiceAdapter(exaService)

	// Create source service adapter for tools interface compatibility
	sourceServiceAdapter := NewSourceServiceAdapter(articleSourceService)

	// Create writing tools for the agent
	writingTools := []tools.BaseTool{
		tools.NewEditTextTool(),
		tools.NewAnalyzeDocumentTool(),
		tools.NewGenerateImagePromptTool(textGenService),         // TextGenService for image prompt generation
		tools.NewGenerateTextContentTool(textGenService),         // New tool for text generation
		tools.NewExaSearchTool(exaAdapter, sourceServiceAdapter), // Exa web search with source creation
		tools.NewExaAnswerTool(exaAdapter),                       // Exa answer for factual Q&A with citations
	}

	// Add source-related tools if source service is available
	if sourceService != nil {
		writingTools = append(writingTools,
			tools.NewGetRelevantSourcesTool(sourceService),
			tools.NewAddContextFromSourcesTool(sourceService), // New tool for context addition
		)
	}

	// Create the agent using the LLM framework
	agentSvc, err := agent.NewAgent(
		config.AgentCopilot, // Use the copilot agent for blog writing
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
		SessionID:    requestID, // Use requestID as sessionID for simplicity
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

// Shutdown gracefully shuts down the agent manager, waiting for in-flight requests
func (m *AgentAsyncCopilotManager) Shutdown(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	log.Printf("[Agent] Shutting down, waiting for %d in-flight requests...", m.ActiveRequests())

	// Cancel all requests
	m.mu.RLock()
	for _, req := range m.requests {
		req.cancel()
	}
	m.mu.RUnlock()

	// Wait for all requests to complete or timeout
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

		// Clean up after configured delay
		time.AfterFunc(m.config.CleanupDelay, func() {
			m.mu.Lock()
			delete(m.requests, asyncReq.ID)
			m.mu.Unlock()
		})
	}()

	log.Printf("[Agent] Starting request %s", asyncReq.ID)

	// Start timeout monitoring goroutine
	timeoutCtx, timeoutCancel := context.WithCancel(asyncReq.ctx)
	defer timeoutCancel()

	go m.monitorTimeout(timeoutCtx, asyncReq)

	// Create session for this request
	session, err := m.sessionSvc.Create(asyncReq.ctx, "Writing Copilot Session")
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

	// Add article ID to context if provided
	ctx := asyncReq.ctx
	var articleID uuid.UUID
	if asyncReq.Request.ArticleID != "" {
		ctx = tools.WithArticleID(ctx, asyncReq.Request.ArticleID)
		// Parse article ID for database operations
		if parsedID, err := uuid.Parse(asyncReq.Request.ArticleID); err == nil {
			articleID = parsedID
		}
	}

	// Load conversation context from database (last 12 messages)
	log.Printf("[Agent] Loading conversation context from database...")
	dbMessages, err := m.loadConversationContext(ctx, articleID, 12)
	if err != nil {
		log.Printf("[Agent] Failed to load conversation context: %v", err)
		// Continue with empty context rather than failing
		dbMessages = []message.Message{}
	}
	log.Printf("[Agent] ✅ Loaded %d messages from database as context", len(dbMessages))

	// Add loaded messages to in-memory session
	for _, msg := range dbMessages {
		_, err := m.messageSvc.Create(ctx, session.ID, message.CreateMessageParams{
			Role:  msg.Role,
			Parts: msg.Parts,
			Model: "loaded",
		})
		if err != nil {
			log.Printf("[Agent] Warning: Failed to add loaded message to session: %v", err)
		}
	}

	// Save the NEW user message to database
	log.Printf("[Agent] 📝 Saving NEW user message to database...")
	log.Printf("[Agent]    Article ID: %s", articleID)
	log.Printf("[Agent]    Content preview: %s", truncate(asyncReq.Request.Message, 100))

	msgContext := metadata.NewMessageContext(
		asyncReq.Request.ArticleID,
		session.ID,
		asyncReq.ID,
		"", // User ID can be added if available
	)

	msgMetadata := metadata.BuildMetaData().WithContext(msgContext)

	savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "user", asyncReq.Request.Message, msgMetadata)
	if err != nil {
		log.Printf("[Agent] ❌ Failed to save user message to database: %v", err)
	} else {
		log.Printf("[Agent] ✅ Saved user message (ID: %s) to database for article %s", savedMsg.ID, articleID)
	}

	// Build user prompt with document content
	userPrompt := asyncReq.Request.Message
	if asyncReq.Request.DocumentContent != "" {
		userPrompt += "\n\n--- Current Document ---\n" + asyncReq.Request.DocumentContent
	}

	asyncReq.iteration = 1

	// Run agent request with article ID context
	resultChan, err := m.agent.Run(ctx, session.ID, userPrompt)

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

	// Stream agent events directly from the result channel
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
			// Stream thinking state to client
			asyncReq.ResponseChan <- StreamResponse{
				RequestID:       asyncReq.ID,
				Type:            "thinking",
				ThinkingMessage: event.ThinkingMessage,
				Iteration:       event.Iteration,
			}
		case agent.AgentEventTypeContentDelta:
			// Stream content chunks in real-time
			asyncReq.ResponseChan <- StreamResponse{
				RequestID: asyncReq.ID,
				Type:      "content_delta",
				Content:   event.ContentDelta,
				Iteration: asyncReq.iteration,
			}
		case agent.AgentEventTypeResponse:
			if event.Message.ID != "" {
				asyncReq.iteration++

				// Save assistant message to database
				m.saveAssistantMessage(ctx, asyncReq, event.Message, articleID)

				// Check if this message has tool calls - if so, stream them separately
				toolCalls := event.Message.ToolCalls()
				if len(toolCalls) > 0 {
					// Stream text content first if there is any
					textContent := event.Message.Content().String()
					if textContent != "" {
						asyncReq.ResponseChan <- StreamResponse{
							RequestID: asyncReq.ID,
							Type:      "text",
							Content:   textContent,
							Iteration: asyncReq.iteration,
						}
					}

					// Stream each tool call as a separate tool_use block
					for _, toolCall := range toolCalls {
						// Parse tool input as JSON if possible
						var toolInput interface{}
						if toolCall.Input != "" {
							// Try to parse as JSON, fallback to string if parsing fails
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
					// Regular text response without tool calls
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
				// This event contains tool results - stream each as a separate tool_result block
				toolResults := event.Message.ToolResults()

				// Build tool group for the new streaming format
				groupID := uuid.New().String()
				toolCalls := make([]agentTypes.ToolCallPayload, 0, len(toolResults))

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

					toolCalls = append(toolCalls, agentTypes.ToolCallPayload{
						ID:     toolResult.ToolCallID,
						Name:   toolName,
						Status: status,
						Result: resultData,
					})
				}

				// Save tool result messages with artifact metadata and stream full message if saved
				savedMsg := m.saveToolResultMessage(ctx, asyncReq, event.Message, toolResults, articleID)
				if savedMsg != nil {
					// Convert MetaData from JSON bytes to map for streaming
					var metaDataMap map[string]interface{}
					if err := json.Unmarshal(savedMsg.MetaData, &metaDataMap); err != nil {
						log.Printf("[Agent] ⚠️ Failed to unmarshal meta_data for streaming: %v", err)
						metaDataMap = make(map[string]interface{})
					}

					// Stream the full message so frontend can render DiffArtifact immediately
					asyncReq.ResponseChan <- StreamResponse{
						RequestID: asyncReq.ID,
						Type:      agentTypes.StreamTypeFullMessage,
						Iteration: asyncReq.iteration,
						FullMessage: &agentTypes.FullMessagePayload{
							ID:        savedMsg.ID.String(),
							ArticleID: savedMsg.ArticleID.String(),
							Role:      savedMsg.Role,
							Content:   savedMsg.Content,
							MetaData:  metaDataMap,
							CreatedAt: savedMsg.CreatedAt.Format(time.RFC3339),
						},
					}
				}

				// Stream tool group complete event (new architecture)
				asyncReq.ResponseChan <- StreamResponse{
					RequestID: asyncReq.ID,
					Type:      agentTypes.StreamTypeToolGroupComplete,
					Iteration: asyncReq.iteration,
					ToolGroup: &agentTypes.ToolGroupPayload{
						GroupID: groupID,
						Status:  "completed",
						Calls:   toolCalls,
					},
				}

				// Also stream individual tool_result events for backward compatibility
				for _, toolResult := range toolResults {
					// Detect if this is a search tool result
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
						Type:      agentTypes.StreamTypeToolResult,
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
			// Unknown event type
			log.Println("Unknown event type", event.Type)
		}
	}

	// Send completion signal
	asyncReq.ResponseChan <- StreamResponse{
		RequestID: asyncReq.ID,
		Type:      "done",
		Done:      true,
	}

	duration := time.Since(startTime)
	log.Printf("[Agent] Completed request %s in %v", asyncReq.ID, duration)
}

// loadConversationContext loads the last N messages from database and reconstructs them for agent context
func (m *AgentAsyncCopilotManager) loadConversationContext(ctx context.Context, articleID uuid.UUID, limit int) ([]message.Message, error) {
	if m.chatService == nil {
		return []message.Message{}, nil
	}

	// Get messages from database
	dbMessages, err := m.chatService.GetConversationHistory(ctx, articleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation history: %w", err)
	}

	log.Printf("[Agent] 📚 Reconstructing %d messages from database metadata...", len(dbMessages))

	// Convert to agent message format
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
			SessionID: "", // Will be set when added to session
		}

		// Reconstruct tool calls from metadata
		if len(dbMsg.MetaData) > 2 {
			var metaData metadata.MessageMetaData
			if err := json.Unmarshal(dbMsg.MetaData, &metaData); err == nil {

				// If this message has tool execution metadata, it means it called a tool
				if metaData.ToolExecution != nil {
					log.Printf("[Agent]    [%d] Reconstructing tool call: %s", i+1, metaData.ToolExecution.ToolName)

					// Add tool call to message
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

				// If this message has artifact, reconstruct the tool result
				if metaData.Artifact != nil {
					log.Printf("[Agent]    [%d] Reconstructing artifact: %s (%s)", i+1, metaData.Artifact.Type, metaData.Artifact.Status)

					// Create a tool result message for the artifact
					// This will be added as a separate message after the assistant message
					if metaData.ToolExecution != nil && metaData.ToolExecution.Output != nil {
						outputJSON, _ := json.Marshal(metaData.ToolExecution.Output)
						toolResult := message.ToolResult{
							ToolCallID: metaData.ToolExecution.ToolID,
							Content:    string(outputJSON),
							IsError:    !metaData.ToolExecution.Success,
						}

						// Add as next message
						toolMsg := message.Message{
							Role: message.Tool,
							Parts: []message.ContentPart{
								toolResult,
							},
						}
						agentMessages = append(agentMessages, msg)     // Add assistant message first
						agentMessages = append(agentMessages, toolMsg) // Then tool result
						continue                                       // Skip adding msg again below
					}
				}
			}
		}

		agentMessages = append(agentMessages, msg)
	}

	log.Printf("[Agent] ✅ Reconstructed %d messages (%d from DB)", len(agentMessages), len(dbMessages))

	return agentMessages, nil
}

// saveAssistantMessage saves an assistant message to the database with metadata
func (m *AgentAsyncCopilotManager) saveAssistantMessage(ctx context.Context, asyncReq *AgentAsyncRequest, msg message.Message, articleID uuid.UUID) {
	if m.chatService == nil || articleID == uuid.Nil {
		return
	}

	content := msg.Content().String()

	log.Printf("[Agent] 💾 Saving assistant message...")
	log.Printf("[Agent]    Article ID: %s", articleID)
	log.Printf("[Agent]    Content preview: %s", truncate(content, 100))

	// Build metadata
	msgContext := metadata.NewMessageContext(
		asyncReq.Request.ArticleID,
		asyncReq.SessionID,
		asyncReq.ID,
		"",
	)

	msgMetadata := metadata.BuildMetaData().WithContext(msgContext)

	// Add tool execution metadata if there are tool calls
	toolCalls := msg.ToolCalls()
	if len(toolCalls) > 0 {
		log.Printf("[Agent]    Has %d tool call(s): %v", len(toolCalls), toolCalls[0].Name)

		// Save the first tool call as metadata (simplified approach)
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

	savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "assistant", content, msgMetadata)
	if err != nil {
		log.Printf("[Agent] ❌ Failed to save assistant message to database: %v", err)
	} else {
		log.Printf("[Agent] ✅ Saved assistant message (ID: %s) to database", savedMsg.ID)
	}
}

// monitorTimeout sends periodic thinking updates and timeout warnings
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

			// Send "still working" message after 1 minute
			if elapsed > 1*time.Minute && elapsed < 2*time.Minute {
				asyncReq.ResponseChan <- StreamResponse{
					RequestID:       asyncReq.ID,
					Type:            "thinking",
					ThinkingMessage: "Still working on your request...",
					Iteration:       updateCount,
				}
			}

			// Send timeout warning after 2 minutes
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

// saveToolResultMessage saves tool result messages with artifact metadata
// Returns the saved message if one was created (for artifact tools), nil otherwise
func (m *AgentAsyncCopilotManager) saveToolResultMessage(ctx context.Context, asyncReq *AgentAsyncRequest, msg message.Message, toolResults []message.ToolResult, articleID uuid.UUID) *models.ChatMessage {
	if m.chatService == nil || articleID == uuid.Nil {
		return nil
	}

	log.Printf("[Agent] 🔧 Processing %d tool result(s) for database save...", len(toolResults))

	// Build metadata context
	msgContext := metadata.NewMessageContext(
		asyncReq.Request.ArticleID,
		asyncReq.SessionID,
		asyncReq.ID,
		"",
	)

	msgMetadata := metadata.BuildMetaData().WithContext(msgContext)

	// Process each tool result to detect artifacts and save tool execution metadata
	for idx, toolResult := range toolResults {
		log.Printf("[Agent]    Tool Result #%d:", idx+1)
		log.Printf("[Agent]       Call ID: %s", toolResult.ToolCallID)
		log.Printf("[Agent]       Is Error: %v", toolResult.IsError)

		if toolResult.IsError {
			log.Printf("[Agent]       ⚠️  Skipping error result")
			continue
		}

		// Parse tool result content
		var toolResultData map[string]interface{}
		if err := json.Unmarshal([]byte(toolResult.Content), &toolResultData); err != nil {
			log.Printf("[Agent]       ⚠️  Failed to parse tool result: %v", err)
			continue
		}

		toolName, _ := toolResultData["tool_name"].(string)
		log.Printf("[Agent]       Tool Name: %s", toolName)

		// Save tool execution metadata for ALL tools
		toolExec := &metadata.ToolExecution{
			ToolName:   toolName,
			ToolID:     toolResult.ToolCallID,
			Output:     toolResultData,
			ExecutedAt: time.Now(),
			Success:    true,
		}
		msgMetadata.WithToolExecution(toolExec)

		// Create artifact for edit_text and rewrite_document tools
		if toolName == "edit_text" || toolName == "rewrite_document" {
			log.Printf("[Agent]       ✏️  ARTIFACT TOOL DETECTED")

			artifactID := uuid.New().String()
			artifactType := metadata.ArtifactTypeCodeEdit
			if toolName == "rewrite_document" {
				artifactType = metadata.ArtifactTypeRewrite
			}

			// Extract content and diff information
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

			log.Printf("[Agent]          Artifact ID: %s", artifactID)
			log.Printf("[Agent]          Type: %s", artifactType)
			log.Printf("[Agent]          Status: %s", metadata.ArtifactStatusPending)
			log.Printf("[Agent]          Description: %s", description)

			// Create artifact info
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

			// Save message with artifact metadata
			content := fmt.Sprintf("📋 %s: %s", toolName, description)
			savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "assistant", content, msgMetadata)
			if err != nil {
				log.Printf("[Agent] ❌ Failed to save tool result message with artifact: %v", err)
				return nil
			}
			log.Printf("[Agent] ✅ Saved artifact message (ID: %s) with status: %s", savedMsg.ID, metadata.ArtifactStatusPending)
			return savedMsg
		}

		// Save search tool results with metadata (no artifact, but save tool execution)
		if toolName == "search_web_sources" {
			log.Printf("[Agent]       🔍 SEARCH TOOL DETECTED")

			// Build a summary message
			totalFound := 0
			sourcesCreated := 0
			query := ""
			if val, ok := toolResultData["total_found"].(float64); ok {
				totalFound = int(val)
			}
			if val, ok := toolResultData["sources_successful"].(float64); ok {
				sourcesCreated = int(val)
			}
			if val, ok := toolResultData["query"].(string); ok {
				query = val
			}

			log.Printf("[Agent]          Query: %s", query)
			log.Printf("[Agent]          Results Found: %d", totalFound)
			log.Printf("[Agent]          Sources Created: %d", sourcesCreated)

			content := fmt.Sprintf("🔍 Web search completed: Found %d results, created %d sources", totalFound, sourcesCreated)
			savedMsg, err := m.chatService.SaveMessage(ctx, articleID, "assistant", content, msgMetadata)
			if err != nil {
				log.Printf("[Agent] ❌ Failed to save search tool result message: %v", err)
				return nil
			}
			log.Printf("[Agent] ✅ Saved search result message (ID: %s)", savedMsg.ID)
			return savedMsg
		}

		// Handle ask_question tool results
		if toolName == "ask_question" {
			log.Printf("[Agent]       ❓ ASK QUESTION TOOL DETECTED")

			// Extract answer and citations
			answer := ""
			citationCount := 0
			if val, ok := toolResultData["answer"].(string); ok {
				answer = val
			}
			if citations, ok := toolResultData["citations"].([]interface{}); ok {
				citationCount = len(citations)
			}

			log.Printf("[Agent]          Answer preview: %s", truncate(answer, 100))
			log.Printf("[Agent]          Citations: %d", citationCount)

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
