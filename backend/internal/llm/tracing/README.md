# Tracing Module for Blog Agent

This module implements W&B Weave Call API tracing for the blog agent system.

## Overview

The tracing module provides:
- W&B Weave Call API integration for trace export
- Automatic agent request tracing
- Tool execution tracing
- LLM provider call tracing
- Real-time call start/end events

## Configuration

Set the following environment variables to enable tracing:

```bash
# Required for tracing
WANDB_TRACING_ENABLED=true
WANDB_API_KEY=your_wandb_api_key
WANDB_PROJECT=your_project_name

# Optional configuration
WANDB_TRACE_ENDPOINT=https://trace.wandb.ai
WANDB_SERVICE_NAME=blog-agent
WANDB_SERVICE_VERSION=1.0.0
WANDB_TRACING_DEBUG=false
```

## Usage

### Basic Tracing

```go
// Get the global tracer
tracer := tracing.GetTracer()

// Start a call
call, ctx := tracer.StartCall("operation_name", ctx)
defer call.End()

// Add attributes
call.SetAttribute("key", "value")
call.SetIntAttribute("count", 42)

// Set inputs and outputs
call.SetInput("input_data", inputValue)
call.SetOutput("result", outputValue)

// Handle errors
if err != nil {
    call.SetError(err)
    return err
}
```

### Agent Tracing

Agent requests are automatically traced when the tracing module is enabled. The following calls are created:

- `agent.request` - The entire agent request
- `tool.call` - Individual tool executions
- `llm.call` - LLM provider calls

### Manual Tool Tracing

```go
tracer := tracing.GetTracer()
call, ctx := tracer.TraceToolCall(ctx, "analyze_document", input)
defer call.End()

// Tool execution logic here...
call.SetOutput("result", result)

if err != nil {
    call.SetError(err)
}
```

## W&B Weave Integration

Traces are automatically exported to W&B Weave using the Call API:

- `POST /call/start` - Start a new call
- `POST /call/end` - End an existing call

The exporter:
- Sends real-time start/end events
- Includes proper project and user information
- Handles authentication via API key
- Provides detailed error logging

## Debugging

Enable debug mode to see detailed tracing logs:

```bash
WANDB_TRACING_DEBUG=true
```

This will log:
- Call creation and completion
- HTTP requests to W&B Weave API
- Response status and body

## API Format

### Call Start
```json
{
  "start": {
    "project_id": "your_project",
    "id": "call_id",
    "op_name": "agent.request",
    "trace_id": "trace_id",
    "parent_id": "parent_call_id",
    "started_at": "2025-01-31T00:00:00Z",
    "attributes": {"key": "value"},
    "inputs": {"prompt": "user input"}
  }
}
```

### Call End
```json
{
  "end": {
    "project_id": "your_project", 
    "id": "call_id",
    "ended_at": "2025-01-31T00:01:00Z",
    "outputs": {"result": "agent response"}
  }
}
```

## Performance

The tracing module is designed to have minimal performance impact:

- Real-time API calls (no batching)
- Configurable enable/disable
- Graceful degradation when W&B API is unavailable
- Asynchronous error handling

## Error Handling

- Failed API calls are logged but don't affect application operation
- Calls are sent individually and retried on temporary failures
- Graceful shutdown ensures all pending calls are completed
