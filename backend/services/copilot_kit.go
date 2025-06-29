package services

import (
	"context"
	"errors"

	openai "github.com/openai/openai-go"
)

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
	Messages []ChatMessage `json:"messages"`
	Model    string        `json:"model"`
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
	converted := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
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
