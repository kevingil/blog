package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"backend/pkg/core/ml/llm/config"
	"backend/pkg/core/ml/llm/logging"
	"backend/pkg/core/ml/llm/message"
	"backend/pkg/core/ml/llm/models"
	"backend/pkg/core/ml/llm/tools"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

type openaiOptions struct {
	baseURL         string
	disableCache    bool
	reasoningEffort string
	extraHeaders    map[string]string
	isGroq          bool
}

type OpenAIOption func(*openaiOptions)

type openaiClient struct {
	providerOptions providerClientOptions
	options         openaiOptions
	client          openai.Client
}

type OpenAIClient ProviderClient

func newOpenAIClient(opts providerClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{
		reasoningEffort: "medium",
	}
	for _, o := range opts.openaiOptions {
		o(&openaiOpts)
	}

	openaiClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithAPIKey(opts.apiKey))
	}
	if openaiOpts.baseURL != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithBaseURL(openaiOpts.baseURL))
	}

	if openaiOpts.extraHeaders != nil {
		for key, value := range openaiOpts.extraHeaders {
			openaiClientOptions = append(openaiClientOptions, option.WithHeader(key, value))
		}
	}

	client := openai.NewClient(openaiClientOptions...)
	return &openaiClient{
		providerOptions: opts,
		options:         openaiOpts,
		client:          client,
	}
}

// convertMessages converts internal messages to Responses API input format
func (o *openaiClient) convertMessages(messages []message.Message) responses.ResponseInputParam {
	var inputItems responses.ResponseInputParam

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var contentList responses.ResponseInputMessageContentListParam

			// Add text content
			if msg.Content().String() != "" {
				contentList = append(contentList, responses.ResponseInputContentUnionParam{
					OfInputText: &responses.ResponseInputTextParam{
						Text: msg.Content().String(),
						Type: "input_text",
					},
				})
			}

			// Add image content
			for _, binaryContent := range msg.BinaryContent() {
				contentList = append(contentList, responses.ResponseInputContentUnionParam{
					OfInputImage: &responses.ResponseInputImageParam{
						ImageURL: openai.String(binaryContent.String(string(models.ProviderOpenAI))),
						Type:     "input_image",
						Detail:   responses.ResponseInputImageDetailAuto,
					},
				})
			}

			inputItems = append(inputItems, responses.ResponseInputItemParamOfMessage(contentList, "user"))

		case message.Assistant:
			// For assistant messages with content, add as message
			if msg.Content().String() != "" {
				contentList := responses.ResponseInputMessageContentListParam{
					responses.ResponseInputContentUnionParam{
						OfInputText: &responses.ResponseInputTextParam{
							Text: msg.Content().String(),
							Type: "input_text",
						},
					},
				}
				inputItems = append(inputItems, responses.ResponseInputItemParamOfMessage(contentList, "assistant"))
			}

			// Add tool calls as function_call items
			for _, call := range msg.ToolCalls() {
				if call.Name == "" {
					continue // Skip tool calls without names
				}
				inputItems = append(inputItems, responses.ResponseInputItemParamOfFunctionCall(
					call.Input,    // arguments
					call.ID,       // call_id
					call.Name,     // name
				))
			}

		case message.Tool:
			// Add tool results as function_call_output items
			for _, result := range msg.ToolResults() {
				inputItems = append(inputItems, responses.ResponseInputItemParamOfFunctionCallOutput(
					result.ToolCallID, // call_id
					result.Content,    // output
				))
			}
		}
	}

	return inputItems
}

// convertTools converts internal tools to Responses API tool format
func (o *openaiClient) convertTools(tools []tools.BaseTool) []responses.ToolUnionParam {
	var responsesTools []responses.ToolUnionParam

	for _, tool := range tools {
		info := tool.Info()
		if info.Name == "" {
			continue // Skip tools without names
		}
		responsesTools = append(responsesTools, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        info.Name,
				Description: openai.String(info.Description),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": info.Parameters,
					"required":   info.Required,
				},
			},
		})
	}

	return responsesTools
}

func (o *openaiClient) finishReason(status string) message.FinishReason {
	switch status {
	case "completed":
		return message.FinishReasonEndTurn
	case "incomplete":
		return message.FinishReasonMaxTokens
	default:
		return message.FinishReasonUnknown
	}
}

// prepareParams creates the Responses API request parameters
func (o *openaiClient) prepareParams(messages []message.Message, tools []tools.BaseTool) responses.ResponseNewParams {
	inputItems := o.convertMessages(messages)

	params := responses.ResponseNewParams{
		Model:        responses.ResponsesModel(o.providerOptions.model.APIModel),
		Instructions: openai.String(o.providerOptions.systemMessage),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
		Store: openai.Bool(false), // We manage state ourselves
	}

	// Add tools if any
	responsesTools := o.convertTools(tools)
	if len(responsesTools) > 0 {
		params.Tools = responsesTools
	}

	// Set max tokens
	if o.providerOptions.maxTokens > 0 {
		params.MaxOutputTokens = openai.Int(o.providerOptions.maxTokens)
	}

	// Set reasoning effort for reasoning models
	if o.providerOptions.model.CanReason {
		switch o.options.reasoningEffort {
		case "low":
			params.Reasoning = responses.ReasoningParam{
				Effort: responses.ReasoningEffortLow,
			}
		case "medium":
			params.Reasoning = responses.ReasoningParam{
				Effort: responses.ReasoningEffortMedium,
			}
		case "high":
			params.Reasoning = responses.ReasoningParam{
				Effort: responses.ReasoningEffortHigh,
			}
		default:
			params.Reasoning = responses.ReasoningParam{
				Effort: responses.ReasoningEffortMedium,
			}
		}
	}

	return params
}

func (o *openaiClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (response *ProviderResponse, err error) {
	params := o.prepareParams(messages, tools)

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared Responses API request", "params", string(jsonData))
	}

	// Log raw JSON request
	rawRequest, _ := json.Marshal(params)
	log.Printf("[RAW REQUEST]\n %s \n", string(rawRequest))

	attempts := 0
	for {
		attempts++
		resp, err := o.client.Responses.New(ctx, params)
		if err != nil {
			retry, after, retryErr := o.shouldRetry(attempts, err)
			if retryErr != nil {
				return nil, retryErr
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			return nil, err
		}

		// Extract content, reasoning, and tool calls from response
		content := ""
		reasoning := ""
		var toolCalls []message.ToolCall

		for _, output := range resp.Output {
			switch output.Type {
			case "message":
				msg := output.AsMessage()
				for _, contentPart := range msg.Content {
					if contentPart.Type == "output_text" {
						content += contentPart.Text
					}
				}
			case "reasoning":
				reasoningItem := output.AsReasoning()
				// Reasoning traces are in Summary field
				for _, summaryPart := range reasoningItem.Summary {
					if summaryPart.Type == "summary_text" {
						reasoning += summaryPart.Text
					}
				}
			case "function_call":
				funcCall := output.AsFunctionCall()
				toolCalls = append(toolCalls, message.ToolCall{
					ID:       funcCall.CallID,
					Name:     funcCall.Name,
					Input:    funcCall.Arguments,
					Type:     "function",
					Finished: true,
				})
			}
		}

		finishReason := o.finishReason(string(resp.Status))
		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &ProviderResponse{
			Content:      content,
			Reasoning:    reasoning,
			ToolCalls:    toolCalls,
			Usage:        o.usage(*resp),
			FinishReason: finishReason,
		}, nil
	}
}

func (o *openaiClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	params := o.prepareParams(messages, tools)

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared Responses API streaming request", "params", string(jsonData))
	}

	// Log raw JSON request
	rawRequest, _ := json.Marshal(params)
	log.Printf("[RAW REQUEST]\n %s \n", string(rawRequest))

	attempts := 0
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		for {
			attempts++
			stream := o.client.Responses.NewStreaming(ctx, params)

			var currentContent string
			var currentReasoning string
			var toolCalls []message.ToolCall
			// Map of ItemID -> ToolCall for tracking in-progress tool calls
			toolCallMap := make(map[string]*message.ToolCall)
			var finalResponse *responses.Response

			for stream.Next() {
				event := stream.Current()

				switch event.Type {
				case "response.output_text.delta":
					// Text content delta
					delta := event.AsResponseOutputTextDelta()
					currentContent += delta.Delta
					eventChan <- ProviderEvent{
						Type:    EventContentDelta,
						Content: delta.Delta,
					}

				case "response.reasoning_summary_text.delta":
					// Reasoning trace delta (for reasoning models)
					rawJSON := event.RawJSON()
					var deltaEvent struct {
						Delta string `json:"delta"`
					}
					if err := json.Unmarshal([]byte(rawJSON), &deltaEvent); err == nil && deltaEvent.Delta != "" {
						currentReasoning += deltaEvent.Delta
						eventChan <- ProviderEvent{
							Type:     EventThinkingDelta,
							Thinking: deltaEvent.Delta,
						}
					}

			case "response.output_item.added":
				// New output item added - check if it's a function call or reasoning
				added := event.AsResponseOutputItemAdded()
				if added.Item.Type == "function_call" {
					tc := &message.ToolCall{
						ID:       added.Item.CallID,
						Name:     added.Item.Name,
						Input:    "",
						Type:     "function",
						Finished: false,
					}
					toolCallMap[added.Item.ID] = tc
					eventChan <- ProviderEvent{
						Type:     EventToolUseStart,
						ToolCall: tc,
					}
				} else if added.Item.Type == "reasoning" {
					// Groq reasoning item added - extract content if available
					rawJSON := event.RawJSON()
					var reasoningItem struct {
						Item struct {
							Content []struct {
								Type string `json:"type"`
								Text string `json:"text"`
							} `json:"content"`
						} `json:"item"`
					}
					if err := json.Unmarshal([]byte(rawJSON), &reasoningItem); err == nil {
						for _, c := range reasoningItem.Item.Content {
							if c.Type == "reasoning_text" && c.Text != "" {
								currentReasoning += c.Text
								eventChan <- ProviderEvent{
									Type:     EventThinkingDelta,
									Thinking: c.Text,
								}
							}
						}
					}
				}

			case "response.reasoning_text.delta", "response.reasoning.delta":
				// Groq reasoning text delta (alternative event types)
				rawJSON := event.RawJSON()
				var deltaEvent struct {
					Delta string `json:"delta"`
					Text  string `json:"text"`
				}
				if err := json.Unmarshal([]byte(rawJSON), &deltaEvent); err == nil {
					delta := deltaEvent.Delta
					if delta == "" {
						delta = deltaEvent.Text
					}
					if delta != "" {
						currentReasoning += delta
						eventChan <- ProviderEvent{
							Type:     EventThinkingDelta,
							Thinking: delta,
						}
					}
				}

				case "response.function_call_arguments.delta":
					// Tool call arguments delta
					delta := event.AsResponseFunctionCallArgumentsDelta()
					if tc, ok := toolCallMap[delta.ItemID]; ok {
						tc.Input += delta.Delta
						eventChan <- ProviderEvent{
							Type:     EventToolUseDelta,
							ToolCall: tc,
						}
					}

				case "response.function_call_arguments.done":
					// Tool call complete
					done := event.AsResponseFunctionCallArgumentsDone()
					if tc, ok := toolCallMap[done.ItemID]; ok {
						tc.Finished = true
						tc.Input = done.Arguments
						toolCalls = append(toolCalls, *tc)
						eventChan <- ProviderEvent{
							Type:     EventToolUseStop,
							ToolCall: tc,
						}
						delete(toolCallMap, done.ItemID)
					}

				case "response.completed":
					// Response complete
					completed := event.AsResponseCompleted()
					finalResponse = &completed.Response

			case "error":
				// Handle error event
				log.Printf("Stream error event: %s", event.RawJSON())

			default:
				// Log unhandled event types to debug Groq reasoning and other providers
				log.Printf("[OpenAI Provider] Unhandled event type: %s, raw: %s", event.Type, event.RawJSON())
			}
			}

			err := stream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				// Stream completed successfully
				finishReason := message.FinishReasonEndTurn
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
					// Log tool calls
					for i, tc := range toolCalls {
						prettyArgs, _ := json.MarshalIndent(json.RawMessage(tc.Input), "    ", "  ")
						log.Printf("🔧 [ToolCall #%d] %s\n    %s", i+1, tc.Name, string(prettyArgs))
					}
				}

				var usage TokenUsage
				if finalResponse != nil {
					usage = o.usage(*finalResponse)
					if finalResponse.Status == "incomplete" {
						finishReason = message.FinishReasonMaxTokens
					}
				}

				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      currentContent,
						Reasoning:    currentReasoning,
						ToolCalls:    toolCalls,
						Usage:        usage,
						FinishReason: finishReason,
					},
				}
				return
			}

			// Handle retry logic
			retry, after, retryErr := o.shouldRetry(attempts, err)
			if retryErr != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
				return
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					if ctx.Err() != nil {
						eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
					}
					return
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			eventChan <- ProviderEvent{Type: EventError, Error: err}
			return
		}
	}()

	return eventChan
}

func (o *openaiClient) shouldRetry(attempts int, err error) (bool, int64, error) {
	var apierr *openai.Error
	if !errors.As(err, &apierr) {
		// Check for tool parsing errors in the error message
		if o.isToolParseError(err) && attempts <= maxToolRetriesOpenAI {
			return true, 1000, nil // 1 second delay for tool errors
		}
		return false, 0, err
	}

	// Handle tool parsing errors (400 status with tool parsing failure)
	if apierr.StatusCode == 400 && o.isToolParseError(err) {
		if attempts > maxToolRetriesOpenAI {
			return false, 0, fmt.Errorf("maximum retry attempts reached for tool parsing error: %d retries", maxToolRetriesOpenAI)
		}
		return true, 1000, nil // 1 second delay for tool errors
	}

	if apierr.StatusCode != 429 && apierr.StatusCode != 500 {
		return false, 0, err
	}

	if attempts > maxRetries {
		return false, 0, fmt.Errorf("maximum retry attempts reached for rate limit: %d retries", maxRetries)
	}

	retryMs := 0
	retryAfterValues := apierr.Response.Header.Values("Retry-After")

	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs = backoffMs + jitterMs
	if len(retryAfterValues) > 0 {
		if _, err := fmt.Sscanf(retryAfterValues[0], "%d", &retryMs); err == nil {
			retryMs = retryMs * 1000
		}
	}
	return true, int64(retryMs), nil
}

// maxToolRetriesOpenAI is the maximum number of retries for tool parsing errors
const maxToolRetriesOpenAI = 2

// isToolParseError checks if an error is a tool parsing error
func (o *openaiClient) isToolParseError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "tool_use_failed") ||
		strings.Contains(errStr, "Failed to parse tool call arguments") ||
		strings.Contains(errStr, "invalid_request_error") ||
		strings.Contains(errStr, "failed_generation")
}

func (o *openaiClient) usage(resp responses.Response) TokenUsage {
	if resp.Usage.InputTokens == 0 && resp.Usage.OutputTokens == 0 {
		return TokenUsage{}
	}

	cachedTokens := resp.Usage.InputTokensDetails.CachedTokens
	inputTokens := resp.Usage.InputTokens - cachedTokens

	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        resp.Usage.OutputTokens,
		CacheCreationTokens: 0, // Responses API doesn't provide this
		CacheReadTokens:     cachedTokens,
	}
}

func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(options *openaiOptions) {
		options.baseURL = baseURL
	}
}

func WithOpenAIExtraHeaders(headers map[string]string) OpenAIOption {
	return func(options *openaiOptions) {
		options.extraHeaders = headers
	}
}

func WithOpenAIDisableCache() OpenAIOption {
	return func(options *openaiOptions) {
		options.disableCache = true
	}
}

func WithReasoningEffort(effort string) OpenAIOption {
	return func(options *openaiOptions) {
		defaultReasoningEffort := "medium"
		switch effort {
		case "low", "medium", "high":
			defaultReasoningEffort = effort
		default:
			logging.Warn("Invalid reasoning effort, using default: medium")
		}
		options.reasoningEffort = defaultReasoningEffort
	}
}

func WithGroq() OpenAIOption {
	return func(options *openaiOptions) {
		options.isGroq = true
	}
}

// Unused imports guard - remove if these become unused
var (
	_ = shared.ReasoningEffortLow
)
