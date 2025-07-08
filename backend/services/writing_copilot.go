package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
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

// Tool definitions
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// Planning phase structured output
type AgentPlan struct {
	Strategy    string        `json:"strategy"`     // "respond_only", "use_tools"
	Reasoning   string        `json:"reasoning"`    // Why this strategy was chosen
	Tools       []PlannedTool `json:"tools"`        // Tools to execute in order
	ResponseMsg string        `json:"response_msg"` // Initial response to user
}

type PlannedTool struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
	Message    string                 `json:"message"` // Message to show in artifact while executing
}

// Agent memory for storing intermediary information
type AgentMemory struct {
	SessionID   string                 `json:"session_id"`
	Context     map[string]interface{} `json:"context"`
	ToolResults []ToolExecutionResult  `json:"tool_results"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ToolExecutionResult struct {
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result"`
	Error      string                 `json:"error,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
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

// Enhanced Writing Copilot Service
type WritingCopilotService struct {
	client      *openai.Client
	textGenSvc  *TextGenerationService
	writerAgent *WriterAgent
	imageGenSvc *ImageGenerationService
	storageSvc  *StorageService
	tools       map[string]ToolDefinition
	memory      map[string]*AgentMemory
	memoryMutex sync.RWMutex
}

func NewWritingCopilotService(textGenSvc *TextGenerationService, writerAgent *WriterAgent, imageGenSvc *ImageGenerationService, storageSvc *StorageService) *WritingCopilotService {
	c := openai.NewClient()

	// Log if API key is missing (helpful for debugging)
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Printf("WARNING: OPENAI_API_KEY environment variable is not set")
	}

	service := &WritingCopilotService{
		client:      &c,
		textGenSvc:  textGenSvc,
		writerAgent: writerAgent,
		imageGenSvc: imageGenSvc,
		storageSvc:  storageSvc,
		memory:      make(map[string]*AgentMemory),
		tools:       make(map[string]ToolDefinition),
	}

	service.initializeTools()
	return service
}

// Initialize available tools
func (s *WritingCopilotService) initializeTools() {
	s.tools["rewrite_document"] = ToolDefinition{
		Name:        "rewrite_document",
		Description: "Completely rewrite or significantly edit the document content",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"new_content": map[string]interface{}{
					"type":        "string",
					"description": "The new document content in markdown format",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Brief explanation of the changes made",
				},
			},
			"required": []string{"new_content", "reason"},
		},
	}

	s.tools["generate_image_prompt"] = ToolDefinition{
		Name:        "generate_image_prompt",
		Description: "Generate an image prompt based on document content",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The document content to generate image prompt for",
				},
			},
			"required": []string{"content"},
		},
	}

	s.tools["analyze_document"] = ToolDefinition{
		Name:        "analyze_document",
		Description: "Analyze document and provide improvement suggestions. Can focus on specific areas or provide general analysis.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"focus_area": map[string]interface{}{
					"type":        "string",
					"description": "Optional: What aspect to focus on (structure, clarity, engagement, grammar, flow, technical_accuracy). If not provided, will analyze overall document quality.",
				},
				"user_request": map[string]interface{}{
					"type":        "string",
					"description": "The user's original request to help understand what they want to improve",
				},
			},
			"required": []string{"user_request"},
		},
	}

	s.tools["edit_text"] = ToolDefinition{
		Name:        "edit_text",
		Description: "Edit specific text in the document while preserving the rest. Use this for targeted edits, improvements, or changes to specific sections.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"original_text": map[string]interface{}{
					"type":        "string",
					"description": "The exact text to find and replace in the document",
				},
				"new_text": map[string]interface{}{
					"type":        "string",
					"description": "The new text to replace the original text with",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Brief explanation of why this edit is being made",
				},
			},
			"required": []string{"original_text", "new_text", "reason"},
		},
	}
}

// Planning phase - decides what tools to use
func (s *WritingCopilotService) createPlan(ctx context.Context, req ChatRequest) (*AgentPlan, error) {
	planningPrompt := `You are a writing assistant planning agent. Analyze the user's request and current document to decide what action to take.

Your response must be valid JSON with this exact structure:
{
  "strategy": "respond_only" or "use_tools",
  "reasoning": "Brief explanation of why this strategy was chosen",
  "tools": [
    {
      "name": "tool_name",
      "parameters": {"param": "value"},
      "message": "What to show user while executing (e.g., 'Rewriting document...', 'Analyzing content...')"
    }
  ],
  "response_msg": "Brief, natural initial response to user"
}

Available tools:
- edit_text: For targeted edits to specific parts of the document (preferred for small changes)
  Parameters: {"original_text": "exact text to replace", "new_text": "replacement text", "reason": "..."}
- rewrite_document: For major content changes or complete rewrites (use when >50% of content changes)
  Parameters: {"new_content": "...", "reason": "..."}
- generate_image_prompt: To create image prompts from content
  Parameters: {"content": "document content"}
- analyze_document: To analyze and provide improvement suggestions
  Parameters: {"user_request": "user's original request", "focus_area": "optional: engagement|clarity|structure|grammar|flow|technical_accuracy"}

Strategy guidelines:
- Use "respond_only" for: questions, simple advice, explanations, small suggestions
- Use "use_tools" for: actual document editing, generating content, creating prompts, analyzing documents

Tool selection guidelines:
- Use "edit_text" for: fixing typos, improving specific sentences/paragraphs, changing tone of specific sections, grammar fixes
- Use "rewrite_document" for: major restructuring, complete rewrites, changing the entire document's approach
- Use "analyze_document" for: providing suggestions without making changes, reviewing content quality

Important guidelines for response_msg:
- Keep initial responses SHORT and conversational (1-2 sentences max)
- For edit_text: Use phrases like "I'll make that edit for you" or "Let me fix that text"
- For analyze_document: Use phrases like "Let me analyze that for you" or "I'll take a look at some suggestions for improvement"
- For rewrite_document: Use phrases like "I'll rewrite that for you" or "Let me improve that content"
- The detailed work will be shown after tool execution, so the initial response should just acknowledge the request

Message guidelines for tools:
- For edit_text: Use messages like "Editing: [brief description]" or "Improving: [section name]"

Always include "user_request" parameter when using analyze_document.

Current document:` + req.DocumentContent

	// Create planning messages
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(planningPrompt),
	}

	// Add conversation history
	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		}
	}

	model := req.Model
	if model == "" {
		model = openai.ChatModelGPT4o
	}

	completion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}

	if len(completion.Choices) == 0 {
		return nil, errors.New("no planning response from OpenAI")
	}

	// Parse the JSON response
	var plan AgentPlan
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &plan); err != nil {
		// Fallback to respond_only if JSON parsing fails
		log.Printf("Failed to parse planning response as JSON: %v", err)
		return &AgentPlan{
			Strategy:    "respond_only",
			Reasoning:   "Failed to parse planning response",
			Tools:       []PlannedTool{},
			ResponseMsg: completion.Choices[0].Message.Content,
		}, nil
	}

	return &plan, nil
}

// Execute a single tool
func (s *WritingCopilotService) executeTool(ctx context.Context, tool PlannedTool, memory *AgentMemory) (*ToolExecutionResult, error) {
	log.Printf("WritingCopilot: Starting tool execution - %s with parameters: %+v", tool.Name, tool.Parameters)

	result := &ToolExecutionResult{
		ToolName:   tool.Name,
		Parameters: tool.Parameters,
		Timestamp:  time.Now(),
	}

	switch tool.Name {
	case "rewrite_document":
		content, ok := tool.Parameters["new_content"].(string)
		if !ok {
			result.Error = "new_content parameter is required"
			log.Printf("WritingCopilot: rewrite_document failed - missing new_content parameter")
			return result, errors.New(result.Error)
		}
		reason, _ := tool.Parameters["reason"].(string)

		log.Printf("WritingCopilot: rewrite_document executing - content length: %d, reason: %s", len(content), reason)

		result.Result = map[string]interface{}{
			"new_content": content,
			"reason":      reason,
		}

		log.Printf("WritingCopilot: rewrite_document completed successfully")

	case "generate_image_prompt":
		content, ok := tool.Parameters["content"].(string)
		if !ok {
			result.Error = "content parameter is required"
			log.Printf("WritingCopilot: generate_image_prompt failed - missing content parameter")
			return result, errors.New(result.Error)
		}

		log.Printf("WritingCopilot: generate_image_prompt executing - content length: %d", len(content))

		prompt, err := s.textGenSvc.GenerateImagePrompt(ctx, content)
		if err != nil {
			result.Error = err.Error()
			log.Printf("WritingCopilot: generate_image_prompt failed - %v", err)
			return result, err
		}

		log.Printf("WritingCopilot: generate_image_prompt completed - generated prompt: %s", prompt)

		result.Result = map[string]interface{}{
			"prompt": prompt,
		}

	case "analyze_document":
		userRequest, ok := tool.Parameters["user_request"].(string)
		if !ok {
			result.Error = "user_request parameter is required"
			return result, errors.New(result.Error)
		}

		focusArea, _ := tool.Parameters["focus_area"].(string)
		if focusArea == "" {
			// Infer focus area from user request
			if strings.Contains(strings.ToLower(userRequest), "engaging") || strings.Contains(strings.ToLower(userRequest), "boring") {
				focusArea = "engagement"
			} else if strings.Contains(strings.ToLower(userRequest), "clear") || strings.Contains(strings.ToLower(userRequest), "confusing") {
				focusArea = "clarity"
			} else if strings.Contains(strings.ToLower(userRequest), "structure") || strings.Contains(strings.ToLower(userRequest), "organize") {
				focusArea = "structure"
			} else if strings.Contains(strings.ToLower(userRequest), "grammar") || strings.Contains(strings.ToLower(userRequest), "spelling") {
				focusArea = "grammar"
			} else {
				focusArea = "overall"
			}
		}

		log.Printf("WritingCopilot: Executing analyze_document tool - Focus Area: %s, User Request: %s", focusArea, userRequest)

		// Just capture the analysis context - let the final response generation create the actual analysis
		result.Result = map[string]interface{}{
			"focus_area":    focusArea,
			"user_request":  userRequest,
			"analysis_done": true,
		}

		log.Printf("WritingCopilot: analyze_document completed - will generate contextual analysis in final response")

	case "edit_text":
		originalText, ok := tool.Parameters["original_text"].(string)
		if !ok {
			result.Error = "original_text parameter is required"
			log.Printf("WritingCopilot: edit_text failed - missing original_text parameter")
			return result, errors.New(result.Error)
		}
		newText, ok := tool.Parameters["new_text"].(string)
		if !ok {
			result.Error = "new_text parameter is required"
			log.Printf("WritingCopilot: edit_text failed - missing new_text parameter")
			return result, errors.New(result.Error)
		}
		reason, _ := tool.Parameters["reason"].(string)

		log.Printf("WritingCopilot: edit_text executing - original length: %d, new length: %d, reason: %s",
			len(originalText), len(newText), reason)

		result.Result = map[string]interface{}{
			"original_text": originalText,
			"new_text":      newText,
			"reason":        reason,
			"edit_type":     "replace",
		}

		log.Printf("WritingCopilot: edit_text completed successfully")

	default:
		result.Error = "unknown tool: " + tool.Name
		log.Printf("WritingCopilot: Unknown tool requested: %s", tool.Name)
		return result, errors.New(result.Error)
	}

	log.Printf("WritingCopilot: Tool %s execution completed successfully", tool.Name)
	return result, nil
}

// Get or create agent memory for session
func (s *WritingCopilotService) getMemory(sessionID string) *AgentMemory {
	s.memoryMutex.Lock()
	defer s.memoryMutex.Unlock()

	if memory, exists := s.memory[sessionID]; exists {
		return memory
	}

	memory := &AgentMemory{
		SessionID:   sessionID,
		Context:     make(map[string]interface{}),
		ToolResults: []ToolExecutionResult{},
		UpdatedAt:   time.Now(),
	}
	s.memory[sessionID] = memory
	return memory
}

// Generate final response after tool execution
func (s *WritingCopilotService) generateFinalResponse(ctx context.Context, req ChatRequest, plan *AgentPlan, toolResults []ToolExecutionResult) (string, error) {
	if plan.Strategy == "respond_only" {
		log.Printf("WritingCopilot: Strategy is respond_only, returning initial response: %s", plan.ResponseMsg)
		return plan.ResponseMsg, nil
	}

	log.Printf("WritingCopilot: Generating final response after tool execution")

	// Build context about what tools were executed and their results
	toolContext := "Tools executed and results:\n"
	var analysisRequests []map[string]interface{}
	var documentChanges []string
	var imagePrompts []string

	for _, result := range toolResults {
		log.Printf("WritingCopilot: Processing tool result - %s: %s", result.ToolName, func() string {
			if result.Error != "" {
				return "ERROR - " + result.Error
			}
			return "SUCCESS"
		}())

		if result.Error != "" {
			toolContext += fmt.Sprintf("- %s: ERROR - %s\n", result.ToolName, result.Error)
		} else {
			toolContext += fmt.Sprintf("- %s: SUCCESS\n", result.ToolName)

			// Extract specific results for better integration
			if resultMap, ok := result.Result.(map[string]interface{}); ok {
				switch result.ToolName {
				case "analyze_document":
					log.Printf("WritingCopilot: Found analyze_document result - Focus: %v, User Request: %v",
						resultMap["focus_area"], resultMap["user_request"])
					analysisRequests = append(analysisRequests, resultMap)
				case "rewrite_document":
					if newContent, ok := resultMap["new_content"].(string); ok && newContent != "" {
						documentChanges = append(documentChanges, "Document has been rewritten with improvements")
					}
				case "generate_image_prompt":
					if prompt, ok := resultMap["prompt"].(string); ok && prompt != "" {
						imagePrompts = append(imagePrompts, prompt)
					}
				}
			}
		}
	}

	// Create a more conversational system prompt that generates contextual analysis
	systemPrompt := `You are a writing assistant. The user made a request and tools were executed to help them. 

Based on the conversation history and the document content, provide a natural, helpful response that directly addresses what the user asked for.

` + toolContext

	// Add specific guidance based on what was executed
	if len(analysisRequests) > 0 {
		log.Printf("WritingCopilot: Creating analysis-focused response for %d analysis requests", len(analysisRequests))

		for _, analysis := range analysisRequests {
			focusArea, _ := analysis["focus_area"].(string)
			userRequest, _ := analysis["user_request"].(string)

			systemPrompt += fmt.Sprintf(`

DOCUMENT ANALYSIS REQUEST:
- User asked: "%s"
- Focus area: %s
- Document content: %s

Please analyze the actual document content and provide specific, contextual suggestions for improvement. 
Look at the actual text and provide concrete recommendations based on what you see.
Make your response conversational and directly address the user's request.
Provide numbered suggestions with specific examples from their document where possible.
Your response should flow naturally as if you're having a conversation with the user.`,
				userRequest, focusArea, req.DocumentContent)
		}
	}

	if len(documentChanges) > 0 {
		log.Printf("WritingCopilot: Adding document rewrite context")
		systemPrompt += `

Document rewriting has been completed. Explain what was changed and why.`
	}

	if len(imagePrompts) > 0 {
		log.Printf("WritingCopilot: Adding image prompt context: %s", strings.Join(imagePrompts, "; "))
		systemPrompt += `

Image prompt has been generated: ` + strings.Join(imagePrompts, "; ")
	}

	// Build conversation messages
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	// Add conversation history
	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	model := req.Model
	if model == "" {
		model = openai.ChatModelGPT4o
	}

	log.Printf("WritingCopilot: Calling OpenAI for final response generation")

	completion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		log.Printf("WritingCopilot: OpenAI call failed: %v", err)
		return "", err
	}

	if len(completion.Choices) == 0 {
		log.Printf("WritingCopilot: No response choices returned from OpenAI")
		return "", errors.New("no response from OpenAI")
	}

	finalResponse := completion.Choices[0].Message.Content
	log.Printf("WritingCopilot: Generated final response (length: %d)", len(finalResponse))

	// Add line break separation if there was an initial response
	separatedResponse := finalResponse
	if plan.ResponseMsg != "" {
		separatedResponse = finalResponse
		//log.Printf("WritingCopilot: Added line break separation between initial response and tool response")
	}

	return separatedResponse, nil
}

// Main method for processing chat requests with new architecture
func (s *WritingCopilotService) ProcessChatStream(ctx context.Context, req ChatRequest, sessionID string) (<-chan StreamResponse, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("no messages provided")
	}

	log.Printf("WritingCopilot: Starting ProcessChatStream for session %s with %d messages", sessionID, len(req.Messages))
	if req.DocumentContent != "" {
		log.Printf("WritingCopilot: Document content provided (length: %d)", len(req.DocumentContent))
	}

	responseChan := make(chan StreamResponse, 50)

	go func() {
		defer close(responseChan)

		// Get agent memory
		memory := s.getMemory(sessionID)
		log.Printf("WritingCopilot: Retrieved/created memory for session %s", sessionID)

		// Phase 1: Planning
		log.Printf("WritingCopilot: Starting planning phase for session %s", sessionID)
		// We always plan first so we don't need to send a planning response
		// but it could be useful one day if we have a need.
		// responseChan <- StreamResponse{
		// 	Type:    "artifact",
		// 	Content: "Planning response...",
		// 	Data: ArtifactUpdate{
		// 		ToolName: "planning",
		// 		Status:   "starting",
		// 		Message:  "Analyzing request and creating plan...",
		// 	},
		// }

		plan, err := s.createPlan(ctx, req)
		if err != nil {
			log.Printf("WritingCopilot: Planning failed: %v", err)
			responseChan <- StreamResponse{
				Type:  "error",
				Error: "Planning failed: " + err.Error(),
				Done:  true,
			}
			return
		}

		log.Printf("WritingCopilot: Plan created - Strategy: %s, Tools: %d", plan.Strategy, len(plan.Tools))

		// Send plan to frontend
		responseChan <- StreamResponse{
			Type: "plan",
			Data: plan,
		}

		// Phase 2: Initial Response
		if plan.ResponseMsg != "" {
			log.Printf("WritingCopilot: Sending initial response: %s", plan.ResponseMsg)
			responseChan <- StreamResponse{
				Type:    "chat",
				Role:    "assistant",
				Content: plan.ResponseMsg,
			}
		}

		// Phase 3: Tool Execution (if needed)
		var toolResults []ToolExecutionResult
		if plan.Strategy == "use_tools" && len(plan.Tools) > 0 {
			log.Printf("WritingCopilot: Executing %d tools", len(plan.Tools))

			for i, tool := range plan.Tools {
				log.Printf("WritingCopilot: Executing tool %d/%d: %s", i+1, len(plan.Tools), tool.Name)

				// Send artifact update
				responseChan <- StreamResponse{
					Type: "artifact",
					Data: ArtifactUpdate{
						ToolName: tool.Name,
						Status:   "starting",
						Message:  tool.Message,
					},
				}

				// Execute tool
				result, err := s.executeTool(ctx, tool, memory)
				if err != nil {
					log.Printf("WritingCopilot: Tool %s failed: %v", tool.Name, err)
					responseChan <- StreamResponse{
						Type: "artifact",
						Data: ArtifactUpdate{
							ToolName: tool.Name,
							Status:   "error",
							Message:  "Failed to execute " + tool.Name,
							Error:    err.Error(),
						},
					}
				} else {
					log.Printf("WritingCopilot: Tool %s completed successfully", tool.Name)
					responseChan <- StreamResponse{
						Type: "artifact",
						Data: ArtifactUpdate{
							ToolName: tool.Name,
							Status:   "completed",
							Message:  "Completed " + tool.Name,
							Result:   result.Result,
						},
					}
				}

				toolResults = append(toolResults, *result)
				memory.ToolResults = append(memory.ToolResults, *result)
			}
		}

		// Phase 4: Final Response (if tools were used)
		if plan.Strategy == "use_tools" && len(toolResults) > 0 {
			log.Printf("WritingCopilot: Generating final response after %d tool executions", len(toolResults))
			finalResponse, err := s.generateFinalResponse(ctx, req, plan, toolResults)
			if err != nil {
				log.Printf("WritingCopilot: Final response generation failed: %v", err)
			} else {
				log.Printf("WritingCopilot: Final response generated, sending to frontend")

				responseChan <- StreamResponse{
					Type:    "chat",
					Role:    "assistant",
					Content: finalResponse,
				}
			}
		}

		// Update memory
		memory.UpdatedAt = time.Now()
		if req.DocumentContent != "" {
			memory.Context["last_document"] = req.DocumentContent
		}
		log.Printf("WritingCopilot: Updated memory for session %s", sessionID)

		// Send completion signal
		responseChan <- StreamResponse{
			Type: "done",
			Done: true,
		}

		log.Printf("WritingCopilot: Completed processing for session %s", sessionID)
	}()

	return responseChan, nil
}

// Legacy method for backward compatibility (non-streaming)
func (s *WritingCopilotService) Generate(ctx context.Context, req ChatRequest) (string, error) {
	sessionID := uuid.New().String()
	streamChan, err := s.ProcessChatStream(ctx, req, sessionID)
	if err != nil {
		return "", err
	}

	var finalResponse string
	for response := range streamChan {
		if response.Type == "chat" && response.Role == "assistant" {
			finalResponse += response.Content
		}
		if response.Done || response.Error != "" {
			break
		}
	}

	return finalResponse, nil
}

// AsyncCopilotManager - Updated to use new architecture
type AsyncCopilotManager struct {
	mu       sync.RWMutex
	requests map[string]*AsyncChatRequest
	service  *WritingCopilotService
}

type AsyncChatRequest struct {
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
	globalAsyncManager *AsyncCopilotManager
	managerOnce        sync.Once
)

// Updated to use new services
func GetAsyncCopilotManager() *AsyncCopilotManager {
	managerOnce.Do(func() {
		globalAsyncManager = &AsyncCopilotManager{
			requests: make(map[string]*AsyncChatRequest),
			service:  NewWritingCopilotService(nil, nil, nil, nil), // Services will be injected later
		}
	})
	return globalAsyncManager
}

// Updated to inject services
func (m *AsyncCopilotManager) SetServices(textGenSvc *TextGenerationService, writerAgent *WriterAgent, imageGenSvc *ImageGenerationService, storageSvc *StorageService) {
	m.service = NewWritingCopilotService(textGenSvc, writerAgent, imageGenSvc, storageSvc)
}

func (m *AsyncCopilotManager) SubmitChatRequest(req ChatRequest) (string, error) {
	if len(req.Messages) == 0 {
		return "", errors.New("no messages provided")
	}

	requestID := uuid.New().String()
	sessionID := uuid.New().String() // In real implementation, this should come from user session
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	asyncReq := &AsyncChatRequest{
		ID:           requestID,
		Request:      req,
		Status:       "processing",
		StartTime:    time.Now(),
		ResponseChan: make(chan StreamResponse, 100),
		ctx:          ctx,
		cancel:       cancel,
		SessionID:    sessionID,
	}

	m.mu.Lock()
	m.requests[requestID] = asyncReq
	m.mu.Unlock()

	go m.processRequest(asyncReq)

	return requestID, nil
}

func (m *AsyncCopilotManager) GetResponseChannel(requestID string) (<-chan StreamResponse, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, exists := m.requests[requestID]
	if !exists {
		return nil, false
	}

	return req.ResponseChan, true
}

func (m *AsyncCopilotManager) processRequest(asyncReq *AsyncChatRequest) {
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

	log.Printf("AsyncCopilotManager: Starting processing for request %s", asyncReq.ID)

	streamChan, err := m.service.ProcessChatStream(asyncReq.ctx, asyncReq.Request, asyncReq.SessionID)
	if err != nil {
		log.Printf("AsyncCopilotManager: Failed to process stream for request %s: %v", asyncReq.ID, err)
		asyncReq.ResponseChan <- StreamResponse{
			RequestID: asyncReq.ID,
			Type:      "error",
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
