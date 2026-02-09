package agent

import (
	"context"

	"backend/pkg/core/ml/llm/message"
	"backend/pkg/core/ml/llm/session"
	"backend/pkg/core/ml/llm/tools"
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
			tools.NewEditTextTool(nil), // no draft persistence for coder agent
			NewAgentTool(sessions, messages),
		}, otherTools...,
	)
}

func TaskAgentTools() []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewFetchTool(),
		tools.NewEditTextTool(nil), // no draft persistence for task agent
	}
}
