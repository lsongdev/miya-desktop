package agentclient

import (
	"fmt"
	"path/filepath"
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
	if endpoint.Type != "" && endpoint.Type != "stdio" {
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
	if endpoint.ID == "miya" {
		return true
	}
	if endpoint.Type == "builtin" || endpoint.Type == "inprocess" {
		return true
	}
	command := filepath.Base(endpoint.Command)
	if command != "miya" && command != "miya-agent" && command != "miya-agents" {
		return isGoRunMiyaAgent(endpoint)
	}
	if len(endpoint.Args) == 0 {
		return true
	}
	return len(endpoint.Args) == 1 && endpoint.Args[0] == "acp"
}

func isGoRunMiyaAgent(endpoint miyaconfig.ACPAgentConfig) bool {
	if filepath.Base(endpoint.Command) != "go" {
		return false
	}
	hasMiyaAgentsDir := false
	hasACP := false
	for _, arg := range endpoint.Args {
		if strings.Contains(arg, "miya-agents") {
			hasMiyaAgentsDir = true
		}
		if arg == "acp" {
			hasACP = true
		}
	}
	return hasMiyaAgentsDir && hasACP
}
