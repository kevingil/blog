package services

import (
	"context"
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

// import json
// import uuid
// from typing import Optional
// from litellm import completion
// from crewai.flow.flow import Flow, start, router, listen
// from copilotkit.crewai import (
//   copilotkit_stream,
//   copilotkit_predict_state,
//   CopilotKitState
// )

// WRITE_DOCUMENT_TOOL = {
//     "type": "function",
//     "function": {
//         "name": "write_document",
//         "description": " ".join("""
//             Write a document. Use markdown formatting to format the document.
//             It's good to format the document extensively so it's easy to read.
//             You can use all kinds of markdown.
//             However, do not use italic or strike-through formatting, it's reserved for another purpose.
//             You MUST write the full document, even when changing only a few words.
//             When making edits to the document, try to make them minimal - do not change every word.
//             Keep stories SHORT!
//             """.split()),
//         "parameters": {
//             "type": "object",
//             "properties": {
//                 "document": {
//                     "type": "string",
//                     "description": "The document to write"
//                 },
//             },
//         }
//     }
// }

// class AgentState(CopilotKitState):
//     """
//     The state of the agent.
//     """
//     document: Optional[str] = None

// class PredictiveStateUpdatesFlow(Flow[AgentState]):
//     """
//     This is a sample flow that demonstrates predictive state updates.
//     """

//     @start()
//     @listen("route_follow_up")
//     async def start_flow(self):
//         """
//         This is the entry point for the flow.
//         """

//     @router(start_flow)
//     async def chat(self):
//         """
//         Standard chat node.
//         """
//         system_prompt = f"""
//         You are a helpful assistant for writing documents.
//         To write the document, you MUST use the write_document tool.
//         You MUST write the full document, even when changing only a few words.
//         When you wrote the document, DO NOT repeat it as a message.
//         Just briefly summarize the changes you made. 2 sentences max.
//         This is the current state of the document: ----\n {self.state.document}\n-----
//         """

//         # 1. Here we specify that we want to stream the tool call to write_document
//         #    to the frontend as state.
//         await copilotkit_predict_state({
//             "document": {
//                 "tool_name": "write_document",
//                 "tool_argument": "document"
//             }
//         })

//         # 2. Run the model and stream the response
//         #    Note: In order to stream the response, wrap the completion call in
//         #    copilotkit_stream and set stream=True.
//         response = await copilotkit_stream(
//             completion(

//                 # 2.1 Specify the model to use
//                 model="openai/gpt-4o",
//                 messages=[
//                     {
//                         "role": "system",
//                         "content": system_prompt
//                     },
//                     *self.state.messages
//                 ],

//                 # 2.2 Bind the tools to the model
//                 tools=[
//                     *self.state.copilotkit.actions,
//                     WRITE_DOCUMENT_TOOL
//                 ],

//                 # 2.3 Disable parallel tool calls to avoid race conditions,
//                 #     enable this for faster performance if you want to manage
//                 #     the complexity of running tool calls in parallel.
//                 parallel_tool_calls=False,
//                 stream=True
//             )
//         )

//         message = response.choices[0].message

//         # 3. Append the message to the messages in state
//         self.state.messages.append(message)

//         # 4. Handle tool call
//         if message.get("tool_calls"):
//             tool_call = message["tool_calls"][0]
//             tool_call_id = tool_call["id"]
//             tool_call_name = tool_call["function"]["name"]
//             tool_call_args = json.loads(tool_call["function"]["arguments"])

//             if tool_call_name == "write_document":
//                 self.state.document = tool_call_args["document"]

//                 # 4.1 Append the result to the messages in state
//                 self.state.messages.append({
//                     "role": "tool",
//                     "content": "Document written.",
//                     "tool_call_id": tool_call_id
//                 })

//                 # 4.2 Append a tool call to confirm changes
//                 self.state.messages.append({
//                     "role": "assistant",
//                     "content": "",
//                     "tool_calls": [{
//                         "id": str(uuid.uuid4()),
//                         "function": {
//                             "name": "confirm_changes",
//                             "arguments": "{}"
//                         }
//                     }]
//                 })

//                 return "route_end"

//         # 5. If our tool was not called, return to the end route
//         return "route_end"

//     @listen("route_end")
//     async def end(self):
//         """
//         End the flow.
//
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
}

// ChatRequestResponse is the immediate response returned when a chat request is submitted
type ChatRequestResponse struct {
	RequestID string `json:"requestId"`
	Status    string `json:"status"`
}

// WebSocket streaming types
type StreamResponse struct {
	RequestID string `json:"requestId,omitempty"`
	Type      string `json:"type"` // "chat", "artifact", "plan", "error", "done"
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	Data      any    `json:"data,omitempty"`
	Done      bool   `json:"done,omitempty"`
	Error     string `json:"error,omitempty"`
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
func InitializeAgentCopilotManager() error {
	// Create session and message services
	sessionSvc := session.NewInMemorySessionService()
	messageSvc := message.NewInMemoryMessageService()

	// Create writing tools for the agent
	writingTools := []tools.BaseTool{
		tools.NewRewriteDocumentTool(nil), // TextGenService not needed for basic functionality
		tools.NewEditTextTool(),
		tools.NewAnalyzeDocumentTool(),
		tools.NewGenerateImagePromptTool(nil), // TextGenService not needed for basic functionality
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

		_, err := m.messageSvc.Create(asyncReq.ctx, session.ID, message.CreateMessageParams{
			Role:  role,
			Parts: parts,
			Model: "user",
		})
		if err != nil {
			log.Printf("AgentAsyncCopilotManager: Failed to create message: %v", err)
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

	// Run agent request
	resultChan, err := m.agent.Run(asyncReq.ctx, session.ID, userPrompt)

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

				asyncReq.ResponseChan <- StreamResponse{
					RequestID: asyncReq.ID,
					Type:      "chat",
					Role:      "assistant",
					Content:   event.Message.Content().String(),
					Done:      false,
				}
			}
		case agent.AgentEventTypeTool:
			if event.Message.ID != "" {
				m.logMessageDetails(event.Message, asyncReq.ID)

				// Extract tool results and serialize them as JSON content
				toolResults := event.Message.ToolResults()
				var content string
				if len(toolResults) > 0 {
					// For now, take the first tool result content (most common case)
					// In the future, we could combine multiple tool results
					content = toolResults[0].Content
				} else {
					// Fallback to regular text content if no tool results
					content = event.Message.Content().String()
				}

				// Stream tool message as role "tool" with content containing tool results
				asyncReq.ResponseChan <- StreamResponse{
					RequestID: asyncReq.ID,
					Type:      "chat",
					Role:      "tool",
					Content:   content,
					Done:      false,
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
