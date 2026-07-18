package agentclient

import (
	"fmt"
	"strings"

	miyaconfig "wails-app/internal/config"

	"github.com/lsongdev/miya-agents/acp"
	miyaagent "github.com/lsongdev/miya-agents/agent"
	agentsconfig "github.com/lsongdev/miya-agents/config"
)

type ConfigLoader func() (*miyaconfig.Config, error)

func NewForCommand(command string, loadConfig ConfigLoader) (*acp.Client, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("agent: empty command")
	}
	endpoint := miyaconfig.ACPAgentConfig{
		Type:    "stdio",
		Command: parts[0],
		Args:    parts[1:],
	}
	return NewForEndpoint(endpoint, loadConfig)
}

func NewForEndpoint(endpoint miyaconfig.ACPAgentConfig, loadConfig ConfigLoader) (*acp.Client, error) {
	if isBuiltin(endpoint) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, fmt.Errorf("agent: load builtin config: %w", err)
		}
		return NewBuiltin(cfg), nil
	}
	if endpoint.Type != "stdio" {
		return nil, fmt.Errorf("agent: unsupported ACP agent type: %s", endpoint.Type)
	}
	if strings.TrimSpace(endpoint.Command) == "" {
		return nil, fmt.Errorf("agent: empty command")
	}
	return acp.DialStdio(endpoint.Command, endpoint.Args...)
}

func NewBuiltin(cfg *agentsconfig.Config) *acp.Client {
	return acp.DialInProcess(miyaagent.NewAgentManager(cfg))
}

func IsBuiltinEndpoint(endpoint miyaconfig.ACPAgentConfig) bool {
	return isBuiltin(endpoint)
}

func isBuiltin(endpoint miyaconfig.ACPAgentConfig) bool {
	return endpoint.Type == "builtin"
}
