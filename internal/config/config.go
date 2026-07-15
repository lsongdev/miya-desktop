package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	agentsconfig "github.com/lsongdev/miya-agents/config"
)

type Config = agentsconfig.Config
type ACPAgentConfig = agentsconfig.ACPAgentConfig
type ProfileConfig = agentsconfig.ProfileConfig
type ProviderConfig = agentsconfig.ProviderConfig

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
		Agents:     []ACPAgentConfig{},
		Profiles:   map[string]*ProfileConfig{},
		Providers:  map[string]*ProviderConfig{},
		McpServers: map[string]*agentsconfig.McpServerConfig{},
		Channels:   map[string]any{},
	}
	normalize(cfg)
	return cfg
}

func normalize(cfg *Config) {
	if cfg.Agents == nil {
		cfg.Agents = []ACPAgentConfig{}
	}
	for i := range cfg.Agents {
		if cfg.Agents[i].Type == "" {
			cfg.Agents[i].Type = "stdio"
		}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*ProfileConfig{}
	}
	if cfg.Providers == nil {
		cfg.Providers = map[string]*ProviderConfig{}
	}
	if cfg.McpServers == nil {
		cfg.McpServers = map[string]*agentsconfig.McpServerConfig{}
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
