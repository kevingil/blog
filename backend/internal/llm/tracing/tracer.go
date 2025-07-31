package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"blog-agent-go/backend/internal/llm/config"
	"blog-agent-go/backend/internal/llm/logging"
)

var (
	globalTracer *Tracer
	tracerOnce   sync.Once
)

type Tracer struct {
	exporter *Exporter
	config   *config.TracingConfig
}

type CallContext struct {
	TraceID  string
	CallID   string
	ParentID string
}

type ActiveCall struct {
	callStart CallStart
	startTime time.Time
	tracer    *Tracer
	inputs    map[string]interface{}
	outputs   map[string]interface{}
}

// Initialize sets up the global tracer
func Initialize(tracingConfig *config.TracingConfig) error {
	var err error
	tracerOnce.Do(func() {
		globalTracer, err = NewTracer(tracingConfig)
	})
	return err
}

// GetTracer returns the global tracer instance
func GetTracer() *Tracer {
	if globalTracer == nil {
		// Fallback to disabled tracer if not initialized
		tracingConfig := &config.TracingConfig{Enabled: false}
		globalTracer, _ = NewTracer(tracingConfig)
	}
	return globalTracer
}

// Shutdown gracefully shuts down the global tracer
func Shutdown(ctx context.Context) error {
	if globalTracer != nil {
		return globalTracer.Shutdown(ctx)
	}
	return nil
}

func NewTracer(tracingConfig *config.TracingConfig) (*Tracer, error) {
	exporter, err := NewExporter(tracingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	return &Tracer{
		exporter: exporter,
		config:   tracingConfig,
	}, nil
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.exporter.Stop(ctx)
}

// StartCall creates a new W&B Weave call for tracing an operation
func (t *Tracer) StartCall(name string, parentCtx context.Context) (*ActiveCall, context.Context) {
	if !t.config.Enabled {
		return &ActiveCall{tracer: t}, parentCtx
	}

	traceID := generateTraceID()
	callID := generateCallID()
	parentID := ""

	// Extract parent call ID from context if available
	if callCtx := CallFromContext(parentCtx); callCtx != nil {
		traceID = callCtx.TraceID
		parentID = callCtx.CallID
	}

	startTime := time.Now()
	callStart := CallStart{
		ProjectID:   t.config.ProjectID,
		ID:          callID,
		OpName:      name,
		DisplayName: name,
		TraceID:     traceID,
		ParentID:    parentID,
		StartedAt:   TimeToRFC3339(startTime),
		Attributes:  make(map[string]interface{}),
		Inputs:      make(map[string]interface{}),
	}

	activeCall := &ActiveCall{
		callStart: callStart,
		startTime: startTime,
		tracer:    t,
		inputs:    make(map[string]interface{}),
		outputs:   make(map[string]interface{}),
	}

	// Send the start call to W&B Weave
	if err := t.exporter.StartCall(callStart); err != nil {
		logging.ErrorPersist(fmt.Sprintf("Failed to start call: %v", err))
	} else if t.config.Debug {
		logging.Debug("[TRACING] Call started", "name", name, "id", callID, "trace_id", traceID)
	}

	// Create new context with call
	newCtx := ContextWithCall(parentCtx, &CallContext{
		TraceID:  traceID,
		CallID:   callID,
		ParentID: parentID,
	})

	return activeCall, newCtx
}

// Context key for call context
type callContextKey struct{}

// ContextWithCall returns a new context with the call context
func ContextWithCall(ctx context.Context, callCtx *CallContext) context.Context {
	return context.WithValue(ctx, callContextKey{}, callCtx)
}

// CallFromContext extracts call context from context
func CallFromContext(ctx context.Context) *CallContext {
	if callCtx, ok := ctx.Value(callContextKey{}).(*CallContext); ok {
		return callCtx
	}
	return nil
}

// ActiveCall methods

func (c *ActiveCall) SetAttribute(key, value string) {
	if c.tracer.config.Enabled {
		c.callStart.Attributes[key] = value
	}
}

func (c *ActiveCall) SetIntAttribute(key string, value int64) {
	if c.tracer.config.Enabled {
		c.callStart.Attributes[key] = value
	}
}

func (c *ActiveCall) SetBoolAttribute(key string, value bool) {
	if c.tracer.config.Enabled {
		c.callStart.Attributes[key] = value
	}
}

func (c *ActiveCall) SetFloatAttribute(key string, value float64) {
	if c.tracer.config.Enabled {
		c.callStart.Attributes[key] = value
	}
}

func (c *ActiveCall) SetInput(key string, value interface{}) {
	if c.tracer.config.Enabled {
		c.inputs[key] = value
	}
}

func (c *ActiveCall) SetOutput(key string, value interface{}) {
	if c.tracer.config.Enabled {
		c.outputs[key] = value
	}
}

func (c *ActiveCall) SetError(err error) {
	if c.tracer.config.Enabled {
		c.SetAttribute("error", err.Error())
		c.SetOutput("exception", err.Error())
	}
}

func (c *ActiveCall) End() {
	if !c.tracer.config.Enabled {
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(c.startTime)

	// Add duration as attribute
	c.SetFloatAttribute("duration_ms", float64(duration.Nanoseconds())/1000000)

	// Create the call end event
	callEnd := CallEnd{
		ProjectID: c.callStart.ProjectID,
		ID:        c.callStart.ID,
		EndedAt:   TimeToRFC3339(endTime),
		Outputs:   c.outputs,
	}

	// Send the end call to W&B Weave
	if err := c.tracer.exporter.EndCall(callEnd); err != nil {
		logging.ErrorPersist(fmt.Sprintf("Failed to end call: %v", err))
	} else if c.tracer.config.Debug {
		logging.Debug("[TRACING] Call ended", "name", c.callStart.OpName, "id", c.callStart.ID, "duration", duration.String())
	}
}

// Convenience functions

// TraceAgentRequest traces an entire agent request
func (t *Tracer) TraceAgentRequest(ctx context.Context, sessionID, prompt string) (*ActiveCall, context.Context) {
	call, ctx := t.StartCall("agent.request", ctx)
	call.SetAttribute("agent.session_id", sessionID)
	call.SetInput("prompt", prompt)
	call.SetAttribute("component", "agent")
	return call, ctx
}

// TraceToolCall traces a tool execution
func (t *Tracer) TraceToolCall(ctx context.Context, toolName string, input string) (*ActiveCall, context.Context) {
	call, ctx := t.StartCall("tool.call", ctx)
	call.SetAttribute("tool.name", toolName)
	call.SetInput("input", input)
	call.SetAttribute("component", "tool")
	return call, ctx
}

// TraceLLMCall traces an LLM provider call
func (t *Tracer) TraceLLMCall(ctx context.Context, provider, model string) (*ActiveCall, context.Context) {
	call, ctx := t.StartCall("llm.call", ctx)
	call.SetAttribute("llm.provider", provider)
	call.SetAttribute("llm.model", model)
	call.SetAttribute("component", "llm")
	return call, ctx
}

// Helper functions

func generateTraceID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateCallID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
