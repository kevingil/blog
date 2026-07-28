package provider

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"backend/pkg/core/ml/llm/message"
)

func TestConvertMessagesUsesOutputTextForAssistantHistory(t *testing.T) {
	client := &openaiClient{}

	input := client.convertMessages([]message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "draft an outline"},
			},
		},
		{
			ID:   "assistant-turn-1",
			Role: message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: "I can help with that."},
			},
		},
	})

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, `"role":"assistant"`) {
		t.Fatalf("expected assistant message in request, got %s", body)
	}
	if !strings.Contains(body, `"type":"output_text"`) {
		t.Fatalf("expected assistant content to use output_text, got %s", body)
	}
	if strings.Contains(body, `"role":"assistant","content":[{"text":"I can help with that.","type":"input_text"}]`) {
		t.Fatalf("assistant history used input_text content: %s", body)
	}
}

func TestIsToolParseErrorDoesNotRetryEveryInvalidRequest(t *testing.T) {
	client := &openaiClient{}

	err := errors.New(`400 Bad Request {"type":"invalid_request_error","message":"Invalid value: 'input_text'."}`)
	if client.isToolParseError(err) {
		t.Fatal("generic invalid_request_error should not be treated as a tool parse error")
	}

	err = errors.New("Failed to parse tool call arguments as JSON")
	if !client.isToolParseError(err) {
		t.Fatal("tool argument parse failures should remain retryable")
	}
}
