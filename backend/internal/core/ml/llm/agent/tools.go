package agent

import (
	"context"

	"blog-agent-go/backend/internal/core/ml/llm/message"
	"blog-agent-go/backend/internal/core/ml/llm/session"
	"blog-agent-go/backend/internal/core/ml/llm/tools"
)

func CoderAgentTools(
	sessions session.Service,
	messages message.Service,
) []tools.BaseTool {
	ctx := context.Background()
	otherTools := GetMcpTools(ctx)
	return append(
		[]tools.BaseTool{
			tools.NewFetchTool(),
			tools.NewEditTextTool(),
			NewAgentTool(sessions, messages),
		}, otherTools...,
	)
}

func TaskAgentTools() []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewFetchTool(),
		tools.NewEditTextTool(),
	}
}
