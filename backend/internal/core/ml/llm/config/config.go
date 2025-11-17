package config

import (
	"blog-agent-go/backend/internal/core/ml/llm/models"
	"os"
)

type AgentName string

const (
	AgentCoder      AgentName = "coder"
	AgentTitle      AgentName = "title"
	AgentSummarizer AgentName = "summarizer"
	AgentWriter     AgentName = "writer"
)

type MCPServerType string

const (
	MCPStdio MCPServerType = "stdio"
	MCPSse   MCPServerType = "sse"
)

type MCPServer struct {
	Type    MCPServerType     `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type AgentConfig struct {
	Model           models.ModelID `json:"model"`
	MaxTokens       int64          `json:"max_tokens"`
	ReasoningEffort int            `json:"reasoning_effort"`
}

type ProviderConfig struct {
	APIKey   string `json:"api_key"`
	Disabled bool   `json:"disabled"`
}

type Config struct {
	Debug        bool                                    `json:"debug"`
	WorkingDir   string                                  `json:"working_dir"`
	ContextPaths []string                                `json:"context_paths"`
	Agents       map[AgentName]AgentConfig               `json:"agents"`
	Providers    map[models.ModelProvider]ProviderConfig `json:"providers"`
	MCPServers   map[string]MCPServer                    `json:"mcp_servers"`
}

var globalConfig *Config

func init() {
	workingDir, _ := os.Getwd()
	globalConfig = &Config{
		Debug:        os.Getenv("DEBUG") == "true",
		WorkingDir:   workingDir,
		ContextPaths: []string{}, // Empty by default for blog agent
		Agents: map[AgentName]AgentConfig{
			AgentWriter: {
				Model:           models.GPT5,
				MaxTokens:       4000,
				ReasoningEffort: 1,
			},
			AgentTitle: {
				Model:           models.GPT4oMini,
				MaxTokens:       100,
				ReasoningEffort: 1,
			},
			AgentSummarizer: {
				Model:           models.GPT4oMini,
				MaxTokens:       2000,
				ReasoningEffort: 1,
			},
		},
		Providers: map[models.ModelProvider]ProviderConfig{
			models.ProviderOpenAI: {
				APIKey:   os.Getenv("OPENAI_API_KEY"),
				Disabled: false,
			},
			models.ProviderAnthropic: {
				APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
				Disabled: true, // Default disabled unless API key is provided
			},
		},
		MCPServers: map[string]MCPServer{}, // Empty by default for blog agent
	}

	// Enable Anthropic if API key is provided
	if globalConfig.Providers[models.ProviderAnthropic].APIKey != "" {
		providerConfig := globalConfig.Providers[models.ProviderAnthropic]
		providerConfig.Disabled = false
		globalConfig.Providers[models.ProviderAnthropic] = providerConfig
	}
}

func Get() *Config {
	return globalConfig
}

func UpdateAgentModel(agentName AgentName, modelID models.ModelID) error {
	if globalConfig.Agents == nil {
		globalConfig.Agents = make(map[AgentName]AgentConfig)
	}

	agent := globalConfig.Agents[agentName]
	agent.Model = modelID
	globalConfig.Agents[agentName] = agent

	return nil
}
