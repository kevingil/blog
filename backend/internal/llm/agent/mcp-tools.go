package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"blog-agent-go/backend/internal/llm/config"
	"blog-agent-go/backend/internal/llm/logging"
	"blog-agent-go/backend/internal/llm/tools"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpTool struct {
	mcpName   string
	tool      mcp.Tool
	mcpConfig config.MCPServer
}

type MCPClient interface {
	Initialize(
		ctx context.Context,
		request mcp.InitializeRequest,
	) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Close() error
}

func (b *mcpTool) Info() tools.ToolInfo {
	required := b.tool.InputSchema.Required
	if required == nil {
		required = make([]string, 0)
	}
	return tools.ToolInfo{
		Name:        fmt.Sprintf("%s_%s", b.mcpName, b.tool.Name),
		Description: b.tool.Description,
		Parameters:  b.tool.InputSchema.Properties,
		Required:    required,
	}
}

func runTool(ctx context.Context, c MCPClient, toolName string, input string) (tools.ToolResponse, error) {
	defer c.Close()
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "BlogAgent",
		Version: "1.0.0",
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		return tools.NewTextErrorResponse(err.Error()), nil
	}

	toolRequest := mcp.CallToolRequest{}
	toolRequest.Params.Name = toolName
	var args map[string]any
	if err = json.Unmarshal([]byte(input), &args); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	toolRequest.Params.Arguments = args
	result, err := c.CallTool(ctx, toolRequest)
	if err != nil {
		return tools.NewTextErrorResponse(err.Error()), nil
	}

	output := ""
	for _, v := range result.Content {
		if v, ok := v.(mcp.TextContent); ok {
			output = v.Text
		} else {
			output = fmt.Sprintf("%v", v)
		}
	}

	return tools.NewTextResponse(output), nil
}

func (b *mcpTool) Run(ctx context.Context, params tools.ToolCall) (tools.ToolResponse, error) {
	switch b.mcpConfig.Type {
	case config.MCPStdio:
		env := make([]string, 0, len(b.mcpConfig.Env))
		for k, v := range b.mcpConfig.Env {
			env = append(env, k+"="+v)
		}
		c, err := client.NewStdioMCPClient(
			b.mcpConfig.Command,
			env,
			b.mcpConfig.Args...,
		)
		if err != nil {
			return tools.NewTextErrorResponse(err.Error()), nil
		}
		return runTool(ctx, c, b.tool.Name, params.Input)
	case config.MCPSse:
		c, err := client.NewSSEMCPClient(
			b.mcpConfig.URL,
			client.WithHeaders(b.mcpConfig.Headers),
		)
		if err != nil {
			return tools.NewTextErrorResponse(err.Error()), nil
		}
		return runTool(ctx, c, b.tool.Name, params.Input)
	}

	return tools.NewTextErrorResponse("invalid mcp type"), nil
}

func NewMcpTool(name string, tool mcp.Tool, mcpConfig config.MCPServer) tools.BaseTool {
	return &mcpTool{
		mcpName:   name,
		tool:      tool,
		mcpConfig: mcpConfig,
	}
}

var mcpTools []tools.BaseTool

func getTools(ctx context.Context, name string, m config.MCPServer, c MCPClient) []tools.BaseTool {
	var stdioTools []tools.BaseTool
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "BlogAgent",
		Version: "1.0.0",
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		logging.Error("error initializing mcp client", "error", err)
		return stdioTools
	}
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		logging.Error("error listing tools", "error", err)
		return stdioTools
	}
	for _, t := range tools.Tools {
		stdioTools = append(stdioTools, NewMcpTool(name, t, m))
	}
	defer c.Close()
	return stdioTools
}

func GetMcpTools(ctx context.Context) []tools.BaseTool {
	if len(mcpTools) > 0 {
		return mcpTools
	}
	for name, m := range config.Get().MCPServers {
		switch m.Type {
		case config.MCPStdio:
			env := make([]string, 0, len(m.Env))
			for k, v := range m.Env {
				env = append(env, k+"="+v)
			}
			c, err := client.NewStdioMCPClient(
				m.Command,
				env,
				m.Args...,
			)
			if err != nil {
				logging.Error("error creating mcp client", "error", err)
				continue
			}

			mcpTools = append(mcpTools, getTools(ctx, name, m, c)...)
		case config.MCPSse:
			c, err := client.NewSSEMCPClient(
				m.URL,
				client.WithHeaders(m.Headers),
			)
			if err != nil {
				logging.Error("error creating mcp client", "error", err)
				continue
			}
			mcpTools = append(mcpTools, getTools(ctx, name, m, c)...)
		}
	}

	return mcpTools
}
