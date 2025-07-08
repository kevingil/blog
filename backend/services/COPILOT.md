# Writing Copilot WebSocket API

This document explains how to use the new enhanced Writing Copilot with planning, tool execution, and real-time streaming via WebSockets.

## Architecture Overview

The new copilot follows this flow:
1. **Planning Phase**: Agent analyzes the request and decides what tools to use
2. **Response Phase**: Sends initial response and communicates the work to be done
3. **Tool Execution**: Executes tools with real-time artifact updates
4. **Final Response**: Provides summary of completed work

## API Flow

### 1. Submit Chat Request

First, submit a chat request to get a request ID:

```javascript
// POST /agent/writing_copilot
const response = await fetch('/agent/writing_copilot', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    messages: [
      {
        role: 'user',
        content: 'Please rewrite this document to be more engaging'
      }
    ],
    model: 'gpt-4o', // optional
    documentContent: 'Your current document content here...' // optional
  })
});

const { requestId } = await response.json();
```

### 2. Connect to WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/websocket');

ws.onopen = () => {
  // Subscribe to the request stream
  ws.send(JSON.stringify({
    action: 'subscribe',
    requestId: requestId
  }));
};
```

### 3. Handle Streaming Responses

```javascript
ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  
  switch (response.type) {
    case 'plan':
      handlePlan(response.data);
      break;
    case 'chat':
      handleChatMessage(response);
      break;
    case 'artifact':
      handleArtifactUpdate(response.data);
      break;
    case 'error':
      handleError(response);
      break;
    case 'done':
      handleCompletion();
      break;
  }
};

function handlePlan(plan) {
  console.log('Agent Plan:', plan);
  // plan.strategy: "respond_only" | "use_tools"
  // plan.reasoning: string
  // plan.tools: array of tools to execute
  // plan.response_msg: initial response
}

function handleChatMessage(response) {
  // Stream chat content to UI
  appendToChat(response.role, response.content);
}

function handleArtifactUpdate(artifact) {
  console.log('Tool Update:', artifact);
  // artifact.tool_name: name of tool being executed
  // artifact.status: "starting" | "in_progress" | "completed" | "error"
  // artifact.message: user-friendly message
  // artifact.result: tool result (when completed)
  
  updateToolStatus(artifact.tool_name, artifact.status, artifact.message);
  
  if (artifact.status === 'completed' && artifact.result) {
    handleToolResult(artifact.tool_name, artifact.result);
  }
}

function handleError(response) {
  console.error('Error:', response.error);
  showErrorMessage(response.error);
}

function handleCompletion() {
  console.log('Request completed');
  hideLoadingIndicator();
}
```

## Message Types

### Plan Message
```json
{
  "requestId": "uuid",
  "type": "plan",
  "data": {
    "strategy": "use_tools",
    "reasoning": "User wants document rewritten, will use rewrite tool",
    "tools": [
      {
        "name": "rewrite_document",
        "parameters": {
          "new_content": "...",
          "reason": "Making content more engaging"
        },
        "message": "Rewriting document for better engagement..."
      }
    ],
    "response_msg": "I'll rewrite your document to make it more engaging"
  }
}
```

### Chat Message
```json
{
  "requestId": "uuid",
  "type": "chat",
  "role": "assistant",
  "content": "I've successfully rewritten your document with more engaging language."
}
```

### Artifact Update
```json
{
  "requestId": "uuid",
  "type": "artifact",
  "data": {
    "tool_name": "analyze_document",
    "status": "completed",
    "message": "Document analysis completed",
    "result": {
      "focus_area": "engagement",
      "user_request": "Help me make this more engaging",
      "suggestions": [
        "Consider adding more compelling examples or stories",
        "Use active voice instead of passive voice",
        "Break up long paragraphs for better readability",
        "Add rhetorical questions to engage readers",
        "Include specific details and concrete examples"
      ],
      "analysis_done": true
    }
  }
}
```

### Error Message
```json
{
  "requestId": "uuid",
  "type": "error",
  "error": "Failed to process request: Invalid document content",
  "done": true
}
```

### Completion Message
```json
{
  "requestId": "uuid",
  "type": "done",
  "done": true
}
```

## Available Tools

### 1. edit_text
Edits specific text in the document while preserving the rest. Ideal for targeted improvements and live editing.

**Parameters:**
- `original_text` (string): The exact text to find and replace in the document
- `new_text` (string): The new text to replace the original text with
- `reason` (string): Brief explanation of why this edit is being made

**Result:**
- `original_text` (string): The text that was replaced
- `new_text` (string): The replacement text
- `reason` (string): Explanation of the edit
- `edit_type` (string): Type of edit performed

**Frontend Behavior:**
- Shows live progress in chat with status updates
- Applies text changes directly to the TipTap editor in real-time
- Provides visual feedback about successful edits

### 2. rewrite_document
Completely rewrites or significantly edits document content.

**Parameters:**
- `new_content` (string): The new document content in markdown
- `reason` (string): Brief explanation of changes made

### 3. generate_image_prompt
Generates an image prompt based on document content.

**Parameters:**
- `content` (string): Document content to generate prompt for

**Result:**
- `prompt` (string): Generated image prompt

### 4. analyze_document
Analyzes document for improvement areas and provides contextual suggestions.

**Parameters:**
- `user_request` (string, required): The user's original request to understand what they want to improve
- `focus_area` (string, optional): What to focus on (engagement, clarity, structure, grammar, flow, technical_accuracy). If not provided, the tool will infer from the user request or provide overall analysis

**Result:**
- `focus_area` (string): The area that was analyzed
- `user_request` (string): The original user request
- `suggestions` (array): List of contextual improvement suggestions
- `analysis_done` (boolean): Completion status

## UI Implementation Example

```javascript
class CopilotUI {
  constructor() {
    this.ws = null;
    this.currentRequestId = null;
  }

  async submitRequest(messages, documentContent) {
    // Show loading state
    this.showLoadingState();
    
    // Submit request
    const response = await fetch('/agent/writing_copilot', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ messages, documentContent })
    });
    
    const { requestId } = await response.json();
    this.currentRequestId = requestId;
    
    // Connect WebSocket
    this.connectWebSocket(requestId);
  }

  connectWebSocket(requestId) {
    this.ws = new WebSocket('ws://localhost:8080/websocket');
    
    this.ws.onopen = () => {
      this.ws.send(JSON.stringify({
        action: 'subscribe',
        requestId: requestId
      }));
    };

    this.ws.onmessage = (event) => {
      const response = JSON.parse(event.data);
      this.handleMessage(response);
    };
  }

  handleMessage(response) {
    switch (response.type) {
      case 'plan':
        this.showPlan(response.data);
        break;
      case 'chat':
        this.streamChatMessage(response.content);
        break;
      case 'artifact':
        this.updateToolProgress(response.data);
        break;
      case 'done':
        this.hideLoadingState();
        break;
    }
  }

  showPlan(plan) {
    const planElement = document.getElementById('agent-plan');
    planElement.innerHTML = `
      <div class="plan">
        <h4>Agent Plan</h4>
        <p><strong>Strategy:</strong> ${plan.strategy}</p>
        <p><strong>Reasoning:</strong> ${plan.reasoning}</p>
        ${plan.tools.length > 0 ? `
          <div class="tools">
            <h5>Tools to execute:</h5>
            <ul>
              ${plan.tools.map(tool => 
                `<li id="tool-${tool.name}">${tool.name}: ${tool.message}</li>`
              ).join('')}
            </ul>
          </div>
        ` : ''}
      </div>
    `;
  }

  updateToolProgress(artifact) {
    const toolElement = document.getElementById(`tool-${artifact.tool_name}`);
    if (toolElement) {
      toolElement.className = `tool-status-${artifact.status}`;
      toolElement.textContent = `${artifact.tool_name}: ${artifact.message}`;
    }
  }
}
```

## Error Handling

Always implement proper error handling:

```javascript
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
  showErrorMessage('Connection error occurred');
};

ws.onclose = (event) => {
  if (event.code !== 1000) {
    console.error('WebSocket closed unexpectedly:', event.code, event.reason);
    showErrorMessage('Connection lost. Please try again.');
  }
};
```

## Live Text Editing

The `edit_text` tool provides a Cursor-like agentic editing experience with real-time updates:

### Frontend Integration

```javascript
// Handle edit_text artifacts specifically
if (artifact.tool_name === 'edit_text') {
  if (artifact.status === 'starting') {
    // Show starting feedback in chat
    showChatMessage(`🔧 Starting text edit: ${artifact.message}`);
  } else if (artifact.status === 'completed' && artifact.result) {
    const editResult = artifact.result;
    // Apply the edit to the TipTap editor
    applyTextEdit(editResult.original_text, editResult.new_text, editResult.reason);
    
    // Show completion feedback in chat
    showChatMessage(`✅ Text edited: ${editResult.reason}`);
  }
}

// Apply text edit to TipTap editor
function applyTextEdit(originalText, newText, reason) {
  const currentText = editor.getText();
  const index = currentText.indexOf(originalText);
  
  if (index !== -1) {
    const from = index;
    const to = index + originalText.length;
    
    // Replace the text in the editor
    editor.chain()
      .focus()
      .setTextSelection({ from, to })
      .insertContent(newText)
      .run();
    
    // Show success feedback
    toast({ title: "Text Updated", description: reason });
  }
}
```

### User Experience

1. **Live Progress**: Users see real-time updates in the chat as edits are applied
2. **Visual Feedback**: Success toasts and progress indicators show edit status
3. **Immediate Application**: Text changes are applied instantly to the editor
4. **Fallback Handling**: If text cannot be found, user gets clear error feedback

### Usage Examples

- **Typo fixes**: "Fix the typo in the second paragraph"
- **Tone adjustments**: "Make the introduction more formal"
- **Grammar improvements**: "Fix grammar issues in this sentence"
- **Style changes**: "Simplify the technical jargon in the conclusion"

## Agent Memory

The system maintains agent memory for each session, storing:
- Session context
- Tool execution results
- Document history

This memory persists throughout the session and helps the agent provide more contextual responses. 
