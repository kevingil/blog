package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"blog-agent-go/backend/internal/llm/agent"
	"blog-agent-go/backend/internal/llm/config"
	"blog-agent-go/backend/internal/llm/message"
	"blog-agent-go/backend/internal/llm/session"
	"blog-agent-go/backend/internal/llm/tools"

	"github.com/google/uuid"
)

// """
// Python Reference
// we want to create our own endpoints instead of using CopilotKit APIs
// """

// ChatMessage is a simplified representation of a chat message that we
// receive from the CopilotKit frontend. It intentionally mirrors the
// OpenAI message schema but without the more advanced fields (tool calls, etc.)
// We only expose what we currently need. If in the future you want to surface
// tool/function-call information, simply extend this struct.
//
// NOTE: CopilotKit always sends role + content – function calls are encoded
// inside the assistant messages as required by the OpenAI wire-format.
// That means we can round-trip the messages without losing any information.
// For now, we map the three common roles to the corresponding helpers that
// ship with the official openai-go SDK (SystemMessage, UserMessage, AssistantMessage).
// Everything else is treated as a plain user message.
//
// We deliberately keep this struct extremely small – avoid over-abstraction as
// requested in the user rules.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is what the /api/copilotkit endpoint receives from the React
// runtime.
//
//   - Messages – full chat transcript
//   - Model    – allow the caller to pick a model (optional, defaults to GPT-4o)
//   - DocumentContent – the current article content to provide context (optional)
//
// In the reference CopilotKit implementation there are also fields for
//
//	"actions", "state" and so on. We do not need them yet for a minimal viable
//
// prototype, so they are omitted. Feel free to extend later.
//
// Keeping the shape small reduces the amount of JSON unmarshalling code we
// have to maintain without losing forward compatibility (unknown fields are
// ignored by encoding/json).
type ChatRequest struct {
	Messages        []ChatMessage `json:"messages"`
	Model           string        `json:"model"`
	DocumentContent string        `json:"documentContent,omitempty"`
	ArticleID       string        `json:"articleId,omitempty"`
}

// ChatRequestResponse is the immediate response returned when a chat request is submitted
type ChatRequestResponse struct {
	RequestID string `json:"requestId"`
	Status    string `json:"status"`
}

// WebSocket streaming types - Block-based streaming for structured agent responses
//
// This struct supports streaming agent responses as individual message blocks, where each
// block represents a specific part of the agent's response (text, tool calls, tool results).
// Blocks belonging to the same request are grouped by RequestID on the frontend.
//
// Supported block types:
// - "text": Assistant text responses
// - "tool_use": Tool/function calls made by the agent
// - "tool_result": Results returned from tool executions
// - "user": User messages (streamed as initial context)
// - "system": System messages (streamed as initial context)
// - "error": Error messages
// - "done": Completion signal
type StreamResponse struct {
	RequestID string `json:"requestId,omitempty"`
	Type      string `json:"type"` // "text", "tool_use", "tool_result", "error", "done"
	Content   string `json:"content,omitempty"`
	Iteration int    `json:"iteration,omitempty"`

	// Tool-specific fields for tool_use blocks
	ToolID    string      `json:"tool_id,omitempty"`
	ToolName  string      `json:"tool_name,omitempty"`
	ToolInput interface{} `json:"tool_input,omitempty"`

	// Tool result fields for tool_result blocks
	ToolResult interface{} `json:"tool_result,omitempty"`

	// Legacy fields for backward compatibility
	Role  string `json:"role,omitempty"`
	Data  any    `json:"data,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Error string `json:"error,omitempty"`
}

// Artifact represents tool execution status shown to user
type ArtifactUpdate struct {
	ToolName string      `json:"tool_name"`
	Status   string      `json:"status"` // "starting", "in_progress", "completed", "error"
	Message  string      `json:"message"`
	Result   interface{} `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// AgentAsyncCopilotManager - LLM Agent Framework powered copilot manager
type AgentAsyncCopilotManager struct {
	requests   map[string]*AgentAsyncRequest
	mu         sync.RWMutex
	agent      agent.Service
	sessionSvc session.Service
	messageSvc message.Service
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

var (
	globalAgentManager *AgentAsyncCopilotManager
	agentManagerOnce   sync.Once
)

// GetAgentAsyncCopilotManager returns the singleton agent-based async manager
func GetAgentAsyncCopilotManager() *AgentAsyncCopilotManager {
	agentManagerOnce.Do(func() {
		globalAgentManager = &AgentAsyncCopilotManager{
			requests: make(map[string]*AgentAsyncRequest),
			// Services will be injected when needed
		}
	})
	return globalAgentManager
}

func (m *AgentAsyncCopilotManager) SetAgentServices(agentSvc agent.Service, sessionSvc session.Service, messageSvc message.Service) {
	m.agent = agentSvc
	m.sessionSvc = sessionSvc
	m.messageSvc = messageSvc
}

// InitializeAgentCopilotManager initializes the agent copilot manager with real services
func InitializeAgentCopilotManager(articleSourceService *ArticleSourceService) error {
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

	// Create writing tools for the agent
	writingTools := []tools.BaseTool{
		tools.NewRewriteDocumentTool(textGenService, sourceService), // TextGenService and SourceService
		tools.NewEditTextTool(),
		tools.NewAnalyzeDocumentTool(),
		tools.NewGenerateImagePromptTool(textGenService),         // TextGenService for image prompt generation
		tools.NewGenerateTextContentTool(textGenService),         // New tool for text generation
		tools.NewExaSearchTool(exaAdapter, articleSourceService), // Exa web search with source creation
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

	// Get the manager and set the services directly
	manager := GetAgentAsyncCopilotManager()
	manager.SetAgentServices(agentSvc, sessionSvc, messageSvc)

	log.Printf("AgentAsyncCopilotManager: Initialized with real LLM agent framework")
	return nil
}

func (m *AgentAsyncCopilotManager) SubmitChatRequest(req ChatRequest) (string, error) {
	if len(req.Messages) == 0 {
		return "", errors.New("no messages provided")
	}

	if m.agent == nil {
		return "", errors.New("agent service not initialized")
	}

	requestID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	asyncReq := &AgentAsyncRequest{
		ID:           requestID,
		Request:      req,
		Status:       "processing",
		StartTime:    time.Now(),
		ResponseChan: make(chan StreamResponse, 100),
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

func (m *AgentAsyncCopilotManager) processAgentRequest(asyncReq *AgentAsyncRequest) {
	defer func() {
		asyncReq.cancel()
		close(asyncReq.ResponseChan)

		// Clean up after 15 minutes
		time.AfterFunc(15*time.Minute, func() {
			m.mu.Lock()
			delete(m.requests, asyncReq.ID)
			m.mu.Unlock()
		})
	}()

	log.Printf("AgentAsyncCopilotManager: Starting agent processing for request %s", asyncReq.ID)
	log.Printf("AgentAsyncCopilotManager: Request details - Article ID: %q, Messages: %d", asyncReq.Request.ArticleID, len(asyncReq.Request.Messages))

	// Create session for this request
	session, err := m.sessionSvc.Create(asyncReq.ctx, "Writing Copilot Session")
	if err != nil {
		log.Printf("AgentAsyncCopilotManager: Failed to create session: %v", err)
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
	if asyncReq.Request.ArticleID != "" {
		ctx = tools.WithArticleID(ctx, asyncReq.Request.ArticleID)
		log.Printf("AgentAsyncCopilotManager: Added article ID %s to context", asyncReq.Request.ArticleID)

		// Verify the article ID was set correctly
		testArticleID := tools.GetArticleIDFromContext(ctx)
		log.Printf("AgentAsyncCopilotManager: Verified article ID in context: %q", testArticleID)
	} else {
		log.Printf("AgentAsyncCopilotManager: No article ID provided in request")
	}

	// Convert request messages to agent format and add them to session
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
			log.Printf("AgentAsyncCopilotManager: Failed to create message: %v", err)
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

	// Log the final prompt that will be sent to the LLM
	log.Printf("📝 [FinalPrompt] Complete prompt being sent to LLM:")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("%s", userPrompt)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Run agent request with article ID context
	// Final verification before running agent
	finalArticleID := tools.GetArticleIDFromContext(ctx)
	log.Printf("AgentAsyncCopilotManager: Final context check - Article ID: %q", finalArticleID)

	resultChan, err := m.agent.Run(ctx, session.ID, userPrompt)

	startTime := time.Now()
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🚀 [Agent] Starting request %s for session %s", asyncReq.ID, session.ID)
	log.Printf("   ⏰ Started at: %s", startTime.Format("15:04:05.000"))
	log.Printf("   👤 Session Title: %s", session.Title)
	if len(asyncReq.Request.Messages) > 0 {
		log.Printf("   💬 Message Count: %d", len(asyncReq.Request.Messages))
	}
	if asyncReq.Request.DocumentContent != "" {
		log.Printf("   📄 Document Content: %d characters", len(asyncReq.Request.DocumentContent))
	}
	log.Printf("   📝 User prompt: %.100s%s", userPrompt, func() string {
		if len(userPrompt) > 100 {
			return "..."
		}
		return ""
	}())
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err != nil {
		duration := time.Since(startTime)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("❌ [Agent] Failed to start agent for request %s", asyncReq.ID)
		log.Printf("   🚨 Error: %v", err)
		log.Printf("   ⏱️  Failed after: %v", duration)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
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
			duration := time.Since(startTime)
			log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			log.Printf("❌ [Agent] Error during processing for request %s", asyncReq.ID)
			log.Printf("   🚨 Error: %v", event.Error)
			log.Printf("   ⏱️  Failed after: %v", duration)
			log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
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
				m.logMessageDetails(event.Message, asyncReq.ID)
				asyncReq.iteration++

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
				m.logMessageDetails(event.Message, asyncReq.ID)

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
			log.Printf("📊 [Agent] Summarization progress: %s", event.Progress)
		}
	}

	// Send completion signal
	asyncReq.ResponseChan <- StreamResponse{
		RequestID: asyncReq.ID,
		Type:      "done",
		Done:      true,
	}

	duration := time.Since(startTime)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ [Agent] Completed processing for request %s", asyncReq.ID)
	log.Printf("   ⏱️  Total duration: %v", duration)
	log.Printf("   🏁 Finished at: %s", time.Now().Format("15:04:05.000"))
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// logMessageDetails provides comprehensive logging for agent messages
func (m *AgentAsyncCopilotManager) logMessageDetails(msg message.Message, requestID string) {
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📨 [Agent] Message Details for Request: %s", requestID)
	log.Printf("   📋 Message ID: %s", msg.ID)
	log.Printf("   🤖 Model: %s", func() string {
		if msg.Model != "" {
			return msg.Model
		}
		return "default"
	}())
	log.Printf("   🏁 Finish Reason: %s", msg.FinishReason())

	// Log tool calls if present
	toolCalls := msg.ToolCalls()
	if len(toolCalls) > 0 {
		log.Printf("   🔧 Tool Calls (%d):", len(toolCalls))
		for i, toolCall := range toolCalls {
			log.Printf("      %d. 🛠️  %s", i+1, toolCall.Name)
			log.Printf("         📝 ID: %s", toolCall.ID)
			if len(toolCall.Input) > 200 {
				log.Printf("         📊 Input: %.200s...", toolCall.Input)
			} else {
				log.Printf("         📊 Input: %s", toolCall.Input)
			}
			log.Printf("         ✅ Finished: %t", toolCall.Finished)
		}
	}

	// Log tool results if present
	toolResults := msg.ToolResults()
	if len(toolResults) > 0 {
		log.Printf("   📋 Tool Results (%d):", len(toolResults))
		for i, result := range toolResults {
			status := "✅"
			if result.IsError {
				status = "❌"
			}
			log.Printf("      %d. %s Tool Call ID: %s", i+1, status, result.ToolCallID)
			if len(result.Content) > 300 {
				log.Printf("         📄 Content: %.300s...", result.Content)
			} else {
				log.Printf("         📄 Content: %s", result.Content)
			}
			if result.Metadata != "" {
				log.Printf("         🏷️  Metadata: %s", result.Metadata)
			}
		}
	}

	// Log message content
	content := msg.Content().String()
	if content != "" {
		log.Printf("   💬 Response Content:")
		if len(content) > 500 {
			log.Printf("      %.500s...", content)
			log.Printf("      📏 Total length: %d characters", len(content))
		} else {
			log.Printf("      %s", content)
		}
	}

	// Log binary content if present
	binaryContent := msg.BinaryContent()
	if len(binaryContent) > 0 {
		log.Printf("   📎 Binary Attachments (%d):", len(binaryContent))
		for i, binary := range binaryContent {
			log.Printf("      %d. 📁 %s (%s) - %d bytes", i+1, binary.Path, binary.MIMEType, len(binary.Data))
		}
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
