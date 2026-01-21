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

	"blog-agent-go/backend/internal/core/ml/llm/config"
	"blog-agent-go/backend/internal/core/ml/llm/logging"
	"blog-agent-go/backend/internal/core/ml/llm/message"
	"blog-agent-go/backend/internal/core/ml/llm/models"
	"blog-agent-go/backend/internal/core/ml/llm/tools"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
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

func (o *openaiClient) convertMessages(messages []message.Message) (openaiMessages []openai.ChatCompletionMessageParamUnion) {
	// Add system message first
	openaiMessages = append(openaiMessages, openai.SystemMessage(o.providerOptions.systemMessage))

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
			content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})
			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: binaryContent.String(string(models.ProviderOpenAI))}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}

				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}

			openaiMessages = append(openaiMessages, openai.UserMessage(content))

		case message.Assistant:
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}

			if msg.Content().String() != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content().String()),
				}
			}

			if len(msg.ToolCalls()) > 0 {
				var validToolCalls []openai.ChatCompletionMessageToolCallParam
				for _, call := range msg.ToolCalls() {
					if call.Name == "" {
						continue // Skip tool calls without names to avoid Groq API errors
					}
					validToolCalls = append(validToolCalls, openai.ChatCompletionMessageToolCallParam{
						ID: call.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      call.Name,
							Arguments: call.Input,
						},
					})
				}
				if len(validToolCalls) > 0 {
					assistantMsg.ToolCalls = validToolCalls
				}
			}

			openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})

		case message.Tool:
			for _, result := range msg.ToolResults() {
				openaiMessages = append(openaiMessages,
					openai.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

func (o *openaiClient) convertTools(tools []tools.BaseTool) []openai.ChatCompletionToolParam {
	var openaiTools []openai.ChatCompletionToolParam

	for _, tool := range tools {
		info := tool.Info()
		if info.Name == "" {
			continue // Skip tools without names to avoid Groq API errors
		}
		openaiTools = append(openaiTools, openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
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

	return openaiTools
}

func (o *openaiClient) finishReason(reason string) message.FinishReason {
	switch reason {
	case "stop":
		return message.FinishReasonEndTurn
	case "length":
		return message.FinishReasonMaxTokens
	case "tool_calls":
		return message.FinishReasonToolUse
	default:
		return message.FinishReasonUnknown
	}
}

func (o *openaiClient) preparedParams(messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(o.providerOptions.model.APIModel),
		Messages: messages,
	}

	// Only include Tools if there are actual tools to avoid Groq API errors with empty arrays
	if len(tools) > 0 {
		params.Tools = tools
	}

	if o.providerOptions.model.CanReason == true {
		params.MaxCompletionTokens = openai.Int(o.providerOptions.maxTokens)

		// Only set reasoning effort for non-Groq providers (Groq uses WithJSONSet in request options)
		if !o.options.isGroq {
			switch o.options.reasoningEffort {
			case "low":
				params.ReasoningEffort = shared.ReasoningEffortLow
			case "medium":
				params.ReasoningEffort = shared.ReasoningEffortMedium
			case "high":
				params.ReasoningEffort = shared.ReasoningEffortHigh
			default:
				params.ReasoningEffort = shared.ReasoningEffortMedium
			}
		}
	} else {
		params.MaxTokens = openai.Int(o.providerOptions.maxTokens)
	}

	return params
}

func (o *openaiClient) getRequestOptions() []option.RequestOption {
	var opts []option.RequestOption

	// Add Groq-specific reasoning parameters
	if o.options.isGroq && o.providerOptions.model.CanReason {
		opts = append(opts,
			option.WithJSONSet("reasoning_format", "parsed"),
		)
	}

	return opts
}

func (o *openaiClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (response *ProviderResponse, err error) {
	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))
	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}
	requestOpts := o.getRequestOptions()

	// Log raw JSON request
	rawRequest, _ := json.Marshal(params)
	log.Printf("[RAW REQUEST]\n %s \n", string(rawRequest))

	attempts := 0
	for {
		attempts++
		openaiResponse, err := o.client.Chat.Completions.New(
			ctx,
			params,
			requestOpts...,
		)
		// If there is an error we are going to see if we can retry the call
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
			return nil, retryErr
		}

		content := ""
		if openaiResponse.Choices[0].Message.Content != "" {
			content = openaiResponse.Choices[0].Message.Content
		}

		toolCalls := o.toolCalls(*openaiResponse)
		finishReason := o.finishReason(string(openaiResponse.Choices[0].FinishReason))

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        o.usage(*openaiResponse),
			FinishReason: finishReason,
		}, nil
	}
}

func (o *openaiClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	// Log raw JSON request
	rawRequest, _ := json.Marshal(params)
	log.Printf("[RAW REQUEST]\n %s \n", string(rawRequest))

	requestOpts := o.getRequestOptions()
	attempts := 0
	eventChan := make(chan ProviderEvent)

	go func() {
		for {
			attempts++
			openaiStream := o.client.Chat.Completions.NewStreaming(
				ctx,
				params,
				requestOpts...,
			)

			acc := openai.ChatCompletionAccumulator{}
			currentContent := ""
			toolCalls := make([]message.ToolCall, 0)
			chunkNum := 0

			for openaiStream.Next() {
				chunk := openaiStream.Current()
				acc.AddChunk(chunk)
				chunkNum++

				for _, choice := range chunk.Choices {
					// Stream content deltas
					if choice.Delta.Content != "" {
						eventChan <- ProviderEvent{
							Type:    EventContentDelta,
							Content: choice.Delta.Content,
						}
						currentContent += choice.Delta.Content
					}
				}
			}

			err := openaiStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				// Stream completed successfully
				finishReason := o.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
				if len(acc.ChatCompletion.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, o.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
					// Log prettified tool calls
					for i, tc := range toolCalls {
						prettyArgs, _ := json.MarshalIndent(json.RawMessage(tc.Input), "    ", "  ")
						log.Printf("🔧 [ToolCall #%d] %s\n    %s", i+1, tc.Name, string(prettyArgs))
					}
				}

				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        o.usage(acc.ChatCompletion),
						FinishReason: finishReason,
					},
				}
				close(eventChan)
				return
			}

			// If there is an error we are going to see if we can retry the call
			retry, after, retryErr := o.shouldRetry(attempts, err)
			if retryErr != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
				close(eventChan)
				return
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					// context cancelled
					if ctx.Err() == nil {
						eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
					}
					close(eventChan)
					return
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
			close(eventChan)
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

func (o *openaiClient) toolCalls(completion openai.ChatCompletion) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(completion.Choices) > 0 && len(completion.Choices[0].Message.ToolCalls) > 0 {
		for _, call := range completion.Choices[0].Message.ToolCalls {
			toolCall := message.ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Input:    call.Function.Arguments,
				Type:     "function",
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (o *openaiClient) usage(completion openai.ChatCompletion) TokenUsage {
	cachedTokens := completion.Usage.PromptTokensDetails.CachedTokens
	inputTokens := completion.Usage.PromptTokens - cachedTokens

	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        completion.Usage.CompletionTokens,
		CacheCreationTokens: 0, // OpenAI doesn't provide this directly
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
