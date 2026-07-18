package config

import (
	"errors"
	"fmt"
	"os"

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
	cfg, err := agentsconfig.LoadConfigFromFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return agentsconfig.NewConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	return cfg, nil
}

func (s *Service) Save(cfg *Config) error {
	if err := agentsconfig.SaveConfigToFile(s.path, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func AgentEndpoints(cfg *Config) ([]ACPAgentConfig, error) {
	return agentsconfig.AgentEndpoints(cfg)
}
