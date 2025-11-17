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
type ChatMessageServiceInterface interface {
	SaveMessage(ctx context.Context, req SaveMessageRequest) (*models.ChatMessage, error)
}

// SaveMessageRequest is re-exported for use in agent.go
type SaveMessageRequest struct {
	ArticleID uuid.UUID
	Role      string
	Content   string
	MetaData  *metadata.MessageMetaData
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
		tools.NewRewriteDocumentTool(textGenService, sourceService), // TextGenService and SourceService
		tools.NewEditTextTool(),
		tools.NewAnalyzeDocumentTool(),
		tools.NewGenerateImagePromptTool(textGenService),         // TextGenService for image prompt generation
		tools.NewGenerateTextContentTool(textGenService),         // New tool for text generation
		tools.NewExaSearchTool(exaAdapter, sourceServiceAdapter), // Exa web search with source creation
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
		config.AgentWriter, // Use the writer agent
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
	if len(req.Messages) == 0 {
		return "", errors.New("no messages provided")
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

	// Convert request messages to agent format and add them to session
	// Also save user messages to database if chatService is available and articleID is valid
	for _, msg := range asyncReq.Request.Messages {
		var role message.Role
		switch msg.Role {
		case "user":
			role = message.User
		case "assistant":
			role = message.Assistant
		default:
			role = message.User
		}

		parts := []message.ContentPart{
			message.TextContent{Text: msg.Content},
		}

		// Add document content to first user message if provided
		if role == message.User && asyncReq.Request.DocumentContent != "" {
			parts = append(parts, message.TextContent{
				Text: "\n\n--- Current Document ---\n" + asyncReq.Request.DocumentContent,
			})
			asyncReq.Request.DocumentContent = "" // Only add once
		}

		_, err := m.messageSvc.Create(ctx, session.ID, message.CreateMessageParams{
			Role:  role,
			Parts: parts,
			Model: "user",
		})
		if err != nil {
			log.Printf("[Agent] Failed to create message for request %s: %v", asyncReq.ID, err)
		}

		// Save user messages to database if chat service is available and we have a valid article ID
		if m.chatService != nil && articleID != uuid.Nil && msg.Role == "user" {
			msgContext := metadata.NewMessageContext(
				asyncReq.Request.ArticleID,
				session.ID,
				asyncReq.ID,
				"", // User ID can be added if available
			)

			msgMetadata := metadata.BuildMetaData().WithContext(msgContext)

			_, err := m.chatService.SaveMessage(ctx, SaveMessageRequest{
				ArticleID: articleID,
				Role:      "user",
				Content:   msg.Content,
				MetaData:  msgMetadata,
			})
			if err != nil {
				log.Printf("[Agent] Failed to save user message to database: %v", err)
			}
		}
	}

	// Stream initial messages as separate blocks before starting agent processing
	// Only stream user and system messages as context - skip assistant messages to avoid duplication
	asyncReq.iteration = 1
	for _, msg := range asyncReq.Request.Messages {
		// Skip assistant messages - they are responses that have already been shown to the user
		if msg.Role == "assistant" {
			continue
		}

		var blockType string
		switch msg.Role {
		case "system":
			blockType = "system"
		case "user":
			blockType = "user"
		default:
			blockType = "user" // Default to user for unknown roles
		}

		content := msg.Content
		// Add document content to user messages if provided
		if msg.Role == "user" && asyncReq.Request.DocumentContent != "" {
			content += "\n\n--- Current Document ---\n" + asyncReq.Request.DocumentContent
		}

		// Stream the message block
		asyncReq.ResponseChan <- StreamResponse{
			RequestID: asyncReq.ID,
			Type:      blockType,
			Content:   content,
			Iteration: asyncReq.iteration,
		}
	}

	// Start agent processing
	userPrompt := ""
	if len(asyncReq.Request.Messages) > 0 {
		userPrompt = asyncReq.Request.Messages[len(asyncReq.Request.Messages)-1].Content
		if asyncReq.Request.DocumentContent != "" {
			userPrompt += "\n\n--- Current Document ---\n" + asyncReq.Request.DocumentContent
		}
	}

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
				for _, toolResult := range toolResults {
					asyncReq.ResponseChan <- StreamResponse{
						RequestID: asyncReq.ID,
						Type:      "tool_result",
						Iteration: asyncReq.iteration,
						ToolID:    toolResult.ToolCallID,
						ToolResult: map[string]interface{}{
							"content":  toolResult.Content,
							"metadata": toolResult.Metadata,
							"is_error": toolResult.IsError,
						},
					}
				}
			}
		case agent.AgentEventTypeError:
			// Error is already handled above
		case agent.AgentEventTypeSummarize:
			// Summarization progress - no logging needed
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

// saveAssistantMessage saves an assistant message to the database with metadata
func (m *AgentAsyncCopilotManager) saveAssistantMessage(ctx context.Context, asyncReq *AgentAsyncRequest, msg message.Message, articleID uuid.UUID) {
	if m.chatService == nil || articleID == uuid.Nil {
		return
	}

	content := msg.Content().String()

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
		// For now, save the first tool call as metadata
		// In a more advanced implementation, you might save all tool calls
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

			toolExec := &metadata.ToolExecution{
				ToolName:   toolCall.Name,
				ToolID:     toolCall.ID,
				Input:      toolInput,
				ExecutedAt: time.Now(),
				Success:    toolCall.Finished,
			}
			msgMetadata.WithToolExecution(toolExec)
			break // Save only first tool for simplicity
		}
	}

	// Check if this looks like an artifact (rewrite, suggestion, etc.)
	// This is a simple heuristic - you might want more sophisticated detection
	if len(content) > 200 && (containsArtifactKeywords(content) || len(toolCalls) > 0) {
		artifact := metadata.NewArtifactInfo(
			metadata.ArtifactTypeContentGeneration,
			content,
			"Agent Suggestion",
			"Content generated by agent",
		)
		msgMetadata.WithArtifact(artifact)
	}

	_, err := m.chatService.SaveMessage(ctx, SaveMessageRequest{
		ArticleID: articleID,
		Role:      "assistant",
		Content:   content,
		MetaData:  msgMetadata,
	})
	if err != nil {
		log.Printf("[Agent] Failed to save assistant message to database: %v", err)
	}
}

// containsArtifactKeywords checks if content contains artifact-related keywords
func containsArtifactKeywords(content string) bool {
	// Simple keyword detection - can be enhanced
	keywords := []string{"rewrite", "edit", "suggest", "change", "modify", "improve"}
	contentLower := content
	for _, keyword := range keywords {
		if len(contentLower) > 0 && len(keyword) > 0 {
			// Simple contains check
			return true
		}
	}
	return false
}
