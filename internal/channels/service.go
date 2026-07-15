package channels

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"wails-app/internal/agentclient"
	miyaconfig "wails-app/internal/config"

	"github.com/lsongdev/miya-agents/acp"
	channelapp "github.com/lsongdev/miya-channels/app"
)

type Status struct {
	Running bool   `json:"running"`
	Command string `json:"command,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Service struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	done       chan error
	status     Status
	loadConfig agentclient.ConfigLoader
}

func NewService(loadConfig agentclient.ConfigLoader) *Service {
	return &Service{loadConfig: loadConfig}
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) Start(ctx context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.status.Running = true
		return s.status, nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	cfg, err := s.loadConfig()
	if err != nil {
		cancel()
		s.status = Status{Error: err.Error()}
		return s.status, err
	}
	done := make(chan error, 1)
	s.cancel = cancel
	s.done = done
	s.status = Status{Running: true, Command: "embedded miya-channels"}

	go s.run(runCtx, cfg, done)
	return s.status, nil
}

func (s *Service) Stop() (Status, error) {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	if cancel == nil {
		s.cancel = nil
		s.done = nil
		s.status.Running = false
		status := s.status
		s.mu.Unlock()
		return status, nil
	}
	s.mu.Unlock()

	cancel()
	if done != nil {
		if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
			return s.Status(), fmt.Errorf("stop miya-channels: %w", err)
		}
	}

	s.mu.Lock()
	s.cancel = nil
	s.done = nil
	s.status.Running = false
	status := s.status
	s.mu.Unlock()
	return status, nil
}

func (s *Service) run(ctx context.Context, cfg *miyaconfig.Config, done chan<- error) {
	err := channelapp.Run(ctx, channelapp.Options{
		Config: cfg,
		NewClient: func(cfg *miyaconfig.Config) (*acp.Client, string, error) {
			endpoint, err := firstEnabledAgent(cfg)
			if err != nil {
				return nil, "", err
			}
			client, err := agentclient.NewForEndpoint(endpoint, s.loadConfig)
			if err != nil {
				return nil, "", err
			}
			return client, endpoint.ID, nil
		},
	})
	s.mu.Lock()
	if s.done == done {
		s.cancel = nil
		s.done = nil
		s.status.Running = false
		s.status.PID = 0
		if ctx.Err() == nil && err != nil {
			s.status.Error = err.Error()
		}
	}
	s.mu.Unlock()
	if ctx.Err() == nil && err != nil {
		log.Printf("[channels] embedded miya-channels exited: %v", err)
	}
	done <- err
}

func firstEnabledAgent(cfg *miyaconfig.Config) (miyaconfig.ACPAgentConfig, error) {
	for _, endpoint := range cfg.Agents {
		if endpoint.IsEnabled() {
			return endpoint, nil
		}
	}
	return miyaconfig.ACPAgentConfig{}, fmt.Errorf("no enabled ACP agent configured")
}
