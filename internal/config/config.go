package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	agentsconfig "github.com/lsongdev/miya-agents/config"
)

type Config struct {
	agentsconfig.Config
	Channels        []json.RawMessage `json:"channels,omitempty" yaml:"channels,omitempty"`
	ChannelsEnabled *bool             `json:"channelsEnabled,omitempty" yaml:"channelsEnabled,omitempty"`
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var shared agentsconfig.Config
	if err := json.Unmarshal(data, &shared); err != nil {
		return err
	}
	var desktop struct {
		Channels        []json.RawMessage `json:"channels"`
		ChannelsEnabled *bool             `json:"channelsEnabled"`
	}
	if err := json.Unmarshal(data, &desktop); err != nil {
		return err
	}
	c.Config = shared
	c.Channels = desktop.Channels
	c.ChannelsEnabled = desktop.ChannelsEnabled
	return nil
}

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
	return agentsconfig.ConfigFile
}

func (s *Service) Path() string {
	return s.path
}

func (s *Service) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

func (s *Service) Load() (*Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := newConfig()
	if len(data) > 0 {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}
	agentsconfig.Normalize(&cfg.Config)
	return cfg, nil
}

func (s *Service) Save(cfg *Config) error {
	if cfg == nil {
		cfg = newConfig()
	}
	agentsconfig.Normalize(&cfg.Config)
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
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

func AgentEndpoints(cfg *Config) ([]ACPAgentConfig, error) {
	if cfg == nil {
		return nil, nil
	}
	return agentsconfig.AgentEndpoints(&cfg.Config)
}

func newConfig() *Config {
	return &Config{
		Config:   *agentsconfig.NewConfig(),
		Channels: []json.RawMessage{},
	}
}
