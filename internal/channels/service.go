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
	channelpkg "github.com/lsongdev/miya-channels/channels"
)

type ChannelEvent struct {
	Channel     string `json:"channel"`
	Type        string `json:"type"`
	Status      string `json:"status,omitempty"`
	QRCode      string `json:"qrcode,omitempty"`
	QRCodeURL   string `json:"qrcodeUrl,omitempty"`
	QRCodeImage string `json:"qrcodeImage,omitempty"`
	Token       string `json:"token,omitempty"`
	BaseURL     string `json:"baseUrl,omitempty"`
	CDNBaseURL  string `json:"cdnBaseUrl,omitempty"`
	Error       string `json:"error,omitempty"`
}

type Status struct {
	Running       bool                    `json:"running"`
	Command       string                  `json:"command,omitempty"`
	PID           int                     `json:"pid,omitempty"`
	Error         string                  `json:"error,omitempty"`
	ChannelEvents map[string]ChannelEvent `json:"channelEvents,omitempty"`
}

type Service struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	done       chan error
	status     Status
	loadConfig agentclient.ConfigLoader
	emit       func(string, ...any)
}

func NewService(loadConfig agentclient.ConfigLoader, emit func(string, ...any)) *Service {
	if emit == nil {
		emit = func(string, ...any) {}
	}
	return &Service{loadConfig: loadConfig, emit: emit}
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
	s.status = Status{
		Running:       true,
		Command:       "embedded miya-channels",
		ChannelEvents: map[string]ChannelEvent{},
	}

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
			return firstAvailableAgentClient(cfg, s.loadConfig)
		},
		OnEvent: s.handleChannelEvent,
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

func (s *Service) handleChannelEvent(event channelpkg.ChannelEvent) {
	next := ChannelEvent{
		Channel:     event.Channel,
		Type:        event.Type,
		Status:      event.Status,
		QRCode:      event.QRCode,
		QRCodeURL:   event.QRCodeURL,
		QRCodeImage: event.QRCodeImage,
		Error:       event.Error,
	}
	s.mu.Lock()
	if s.status.ChannelEvents == nil {
		s.status.ChannelEvents = map[string]ChannelEvent{}
	}
	s.status.ChannelEvents[next.Channel] = next
	s.mu.Unlock()
	s.emit("channel:event", next)
}

func firstAvailableAgentClient(cfg *miyaconfig.Config, loadConfig agentclient.ConfigLoader) (*acp.Client, string, error) {
	var lastErr error
	endpoints := append([]miyaconfig.ACPAgentConfig{{
		ID:      "miya",
		Name:    "Miya Agents",
		Enabled: boolPtr(true),
		Type:    "builtin",
		Command: "miya-agent",
		Args:    []string{"acp"},
	}}, cfg.Agents...)
	for _, endpoint := range endpoints {
		if !endpoint.IsEnabled() {
			continue
		}
		client, err := agentclient.NewForEndpoint(endpoint, loadConfig)
		if err == nil {
			return client, endpoint.ID, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("no enabled ACP agent could be connected: %w", lastErr)
	}
	return nil, "", fmt.Errorf("no enabled ACP agent configured")
}

func boolPtr(v bool) *bool {
	return &v
}
