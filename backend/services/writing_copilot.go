package services

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	openai "github.com/openai/openai-go"
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

// CopilotKitService is a thin wrapper around the OpenAI client that exposes a
// helper to open a streaming chat completion. We keep it completely stateless
// so callers can create it ad-hoc – no need to attach it to the FiberServer
// unless you want to reuse it.
//
// It intentionally does not try to be generic – only implement what we need
// today. If you later want to integrate advanced features (tools, function
// calling, parallel_tool_calls, …) you can layer it on top.
type CopilotKitService struct {
	client *openai.Client
}

func NewCopilotKitService() *CopilotKitService {
	c := openai.NewClient()

	// Log if API key is missing (helpful for debugging)
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Printf("WARNING: OPENAI_API_KEY environment variable is not set")
	}

	return &CopilotKitService{client: &c}
}

// Generate sends the accumulated chat transcript to the OpenAI API and returns
// the assistant's response. We intentionally do **not** stream for now to keep
// the implementation straightforward and compatible with the version of the
// SDK that is pinned in go.sum. The HTTP handler turns this single response
// into a Server-Sent-Events (SSE) stream so that the frontend stays unchanged.
func (s *CopilotKitService) Generate(ctx context.Context, req ChatRequest) (string, error) {
	if len(req.Messages) == 0 {
		return "", errors.New("no messages provided")
	}

	// Convert messages into the union type the SDK expects.
	converted := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages)+1)

	// Build system message for the writing assistant
	systemPrompt := `You are an expert writing assistant helping users improve their articles and blog posts. 

Your capabilities:
- Analyze and provide feedback on writing
- Suggest improvements for clarity, structure, and engagement
- Help with editing, rewriting, and content enhancement
- Answer questions about writing techniques and best practices

When the user asks you to make changes to their document, provide clear suggestions and explain your reasoning. Be conversational and helpful in your responses.`

	// Include document content in system prompt if provided
	if req.DocumentContent != "" {
		systemPrompt += "\n\nCurrent document content:\n\n" + req.DocumentContent
	}

	converted = append(converted, openai.SystemMessage(systemPrompt))

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			converted = append(converted, openai.SystemMessage(m.Content))
		case "assistant":
			converted = append(converted, openai.AssistantMessage(m.Content))
		case "user":
			converted = append(converted, openai.UserMessage(m.Content))
		default:
			converted = append(converted, openai.UserMessage(m.Content))
		}
	}

	model := req.Model
	if model == "" {
		model = openai.ChatModelGPT4o
	}

	completion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Messages: converted,
	})
	if err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("openai returned no choices")
	}

	return completion.Choices[0].Message.Content, nil
}

// StreamResponse represents a single chunk in the streaming response
type StreamResponse struct {
	RequestID string `json:"requestId,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	Done      bool   `json:"done,omitempty"`
	Error     string `json:"error,omitempty"`
}

// AsyncCopilotManager manages background processing of chat requests
type AsyncCopilotManager struct {
	mu       sync.RWMutex
	requests map[string]*AsyncChatRequest
	service  *CopilotKitService
}

// AsyncChatRequest represents a chat request being processed in the background
type AsyncChatRequest struct {
	ID           string
	Request      ChatRequest
	Status       string
	StartTime    time.Time
	ResponseChan chan StreamResponse
	ctx          context.Context
	cancel       context.CancelFunc
}

var (
	globalAsyncManager *AsyncCopilotManager
	managerOnce        sync.Once
)

// GetAsyncCopilotManager returns the global async copilot manager (singleton)
func GetAsyncCopilotManager() *AsyncCopilotManager {
	managerOnce.Do(func() {
		globalAsyncManager = &AsyncCopilotManager{
			requests: make(map[string]*AsyncChatRequest),
			service:  NewCopilotKitService(),
		}
	})
	return globalAsyncManager
}

// SubmitChatRequest submits a chat request for async processing and returns a request ID
func (m *AsyncCopilotManager) SubmitChatRequest(req ChatRequest) (string, error) {
	if len(req.Messages) == 0 {
		return "", errors.New("no messages provided")
	}

	requestID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	asyncReq := &AsyncChatRequest{
		ID:           requestID,
		Request:      req,
		Status:       "processing",
		StartTime:    time.Now(),
		ResponseChan: make(chan StreamResponse, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	m.mu.Lock()
	m.requests[requestID] = asyncReq
	m.mu.Unlock()

	// Start background processing
	go m.processRequest(asyncReq)

	return requestID, nil
}

// GetResponseChannel returns the response channel for a given request ID
func (m *AsyncCopilotManager) GetResponseChannel(requestID string) (<-chan StreamResponse, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, exists := m.requests[requestID]
	if !exists {
		return nil, false
	}

	return req.ResponseChan, true
}

// processRequest processes a chat request in the background
func (m *AsyncCopilotManager) processRequest(asyncReq *AsyncChatRequest) {
	defer func() {
		asyncReq.cancel()
		close(asyncReq.ResponseChan)

		// Clean up after 10 minutes
		time.AfterFunc(10*time.Minute, func() {
			m.mu.Lock()
			delete(m.requests, asyncReq.ID)
			m.mu.Unlock()
		})
	}()

	log.Printf("AsyncCopilotManager: Starting processing for request %s", asyncReq.ID)

	streamChan, err := m.service.GenerateStream(asyncReq.ctx, asyncReq.Request)
	if err != nil {
		log.Printf("AsyncCopilotManager: Failed to generate stream for request %s: %v", asyncReq.ID, err)
		asyncReq.ResponseChan <- StreamResponse{
			RequestID: asyncReq.ID,
			Error:     err.Error(),
			Done:      true,
		}
		return
	}

	// Forward streaming responses
	for response := range streamChan {
		response.RequestID = asyncReq.ID

		select {
		case asyncReq.ResponseChan <- response:
			// Successfully sent
		case <-asyncReq.ctx.Done():
			log.Printf("AsyncCopilotManager: Request %s cancelled", asyncReq.ID)
			return
		}

		if response.Done || response.Error != "" {
			break
		}
	}

	log.Printf("AsyncCopilotManager: Completed processing for request %s", asyncReq.ID)
}

// GenerateStream sends the chat transcript to OpenAI and streams the response back
func (s *CopilotKitService) GenerateStream(ctx context.Context, req ChatRequest) (<-chan StreamResponse, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("no messages provided")
	}

	// Convert messages into the union type the SDK expects
	converted := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))

	// Build system message for the writing assistant
	systemPrompt := `You are an expert writing assistant helping users improve their articles and blog posts. 

Your capabilities:
- Analyze and provide feedback on writing
- Suggest improvements for clarity, structure, and engagement
- Help with editing, rewriting, and content enhancement
- Answer questions about writing techniques and best practices

When the user asks you to make changes to their document, provide clear suggestions and explain your reasoning. Be conversational and helpful in your responses.`

	// Include document content in system prompt if provided
	if req.DocumentContent != "" {
		systemPrompt += "\n\nCurrent document content:\n\n" + req.DocumentContent
	}

	converted = append(converted, openai.SystemMessage(systemPrompt))

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			converted = append(converted, openai.SystemMessage(m.Content))
		case "assistant":
			converted = append(converted, openai.AssistantMessage(m.Content))
		case "user":
			converted = append(converted, openai.UserMessage(m.Content))
		default:
			converted = append(converted, openai.UserMessage(m.Content))
		}
	}

	model := req.Model
	if model == "" {
		model = openai.ChatModelGPT4o
	}

	// Create the streaming request
	streamParams := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: converted,
	}

	stream := s.client.Chat.Completions.NewStreaming(ctx, streamParams)

	responseChan := make(chan StreamResponse, 10)

	go func() {
		defer close(responseChan)
		defer stream.Close()

		// Check for immediate errors
		if stream.Err() != nil {
			log.Printf("OpenAI streaming error: %v", stream.Err())
			responseChan <- StreamResponse{
				Error: stream.Err().Error(),
				Done:  true,
			}
			return
		}

		hasContent := false
		for stream.Next() {
			chunk := stream.Current()

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			delta := choice.Delta

			// Handle regular content
			if delta.Content != "" {
				hasContent = true
				responseChan <- StreamResponse{
					Role:    "assistant",
					Content: delta.Content,
				}
			}

			// Handle finish reason
			if choice.FinishReason != "" {
				log.Printf("Stream finished with reason: %s", choice.FinishReason)
				break
			}
		}

		// Check for stream errors
		if err := stream.Err(); err != nil {
			log.Printf("OpenAI stream error: %v", err)
			responseChan <- StreamResponse{
				Error: err.Error(),
				Done:  true,
			}
			return
		}

		// If no content was received, send an error
		if !hasContent {
			log.Printf("No content received from OpenAI")
			responseChan <- StreamResponse{
				Error: "No response received from OpenAI",
				Done:  true,
			}
			return
		}

		// Send done signal
		responseChan <- StreamResponse{Done: true}
	}()

	return responseChan, nil
}
