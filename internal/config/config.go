package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Agents     map[string]AgentConfig     `json:"agents,omitempty"`
	Providers  map[string]ProviderConfig  `json:"providers,omitempty"`
	McpServers map[string]McpServerConfig `json:"mcpServers,omitempty"`
	ACP        *ACPConfig                 `json:"acp,omitempty"`
	Channels   map[string]any             `json:"channels,omitempty"`
	Tools      map[string]any             `json:"tools,omitempty"`
	Logging    map[string]any             `json:"logging,omitempty"`
}

type ProviderConfig struct {
	APIKey  string `json:"apiKey"`
	APIBase string `json:"apiBase,omitempty"`
	Type    string `json:"type,omitempty"`
}

type AgentConfig struct {
	Provider            string  `json:"provider"`
	Model               string  `json:"model,omitempty"`
	Workspace           string  `json:"workspace,omitempty"`
	MaxTokens           int     `json:"maxTokens,omitempty"`
	Temperature         float64 `json:"temperature,omitempty"`
	ContextWindowTokens int     `json:"contextWindowTokens,omitempty"`
	ContextWarnRatio    float64 `json:"contextWarnRatio,omitempty"`
}

type McpServerConfig struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type ACPConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type Service struct {
	path string
}

func NewService() *Service {
	return &Service{path: DefaultConfigFile()}
}

func DefaultConfigFile() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".miya", "config.json")
	}
	return filepath.Join(home, ".miya", "config.json")
}

func (s *Service) Path() string {
	return s.path
}

func (s *Service) Load() (*Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := defaultConfig()
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if len(data) == 0 {
		return defaultConfig(), nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	normalize(&cfg)
	return &cfg, nil
}

func (s *Service) Save(cfg *Config) error {
	if cfg == nil {
		cfg = defaultConfig()
	}
	normalize(cfg)
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func defaultConfig() *Config {
	cfg := &Config{
		Agents:     map[string]AgentConfig{},
		Providers:  map[string]ProviderConfig{},
		McpServers: map[string]McpServerConfig{},
		Channels:   map[string]any{},
	}
	normalize(cfg)
	return cfg
}

func normalize(cfg *Config) {
	if cfg.Agents == nil {
		cfg.Agents = map[string]AgentConfig{}
	}
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderConfig{}
	}
	if cfg.McpServers == nil {
		cfg.McpServers = map[string]McpServerConfig{}
	}
	if cfg.Channels == nil {
		cfg.Channels = map[string]any{}
	}
	for id, server := range cfg.McpServers {
		if server.Type == "" {
			if server.URL != "" && server.Command == "" {
				server.Type = "sse"
			} else {
				server.Type = "stdio"
			}
			cfg.McpServers[id] = server
		}
	}
}
