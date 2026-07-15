package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"wails-app/internal/agent"
	"wails-app/internal/agentclient"
	channelservice "wails-app/internal/channels"
	miyaconfig "wails-app/internal/config"

	"github.com/lsongdev/miya-agents/acp"
)

// App struct
type App struct {
	ctx      context.Context
	manager  *agent.Manager
	config   *miyaconfig.Service
	channels *channelservice.Service
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.config = miyaconfig.NewService()
	a.manager = agent.New(ctx, a.config.Load)
	a.channels = channelservice.NewService(a.config.Load)
}

func (a *App) shutdown(ctx context.Context) {
	if a.channels != nil {
		_, _ = a.channels.Stop()
	}
	if a.manager != nil {
		_ = a.manager.Disconnect()
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) ConnectAgent(command string) error {
	return a.manager.Connect(command)
}

func (a *App) InitializeAgent(name, version string) (*agent.AgentInfo, error) {
	return a.manager.Initialize(name, version)
}

func (a *App) CreateSession(cwd string) (*agent.Session, error) {
	return a.manager.NewSession(cwd)
}

func (a *App) DefaultCwd() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		return cwd
	}
	return "."
}

func (a *App) LoadSession(sessionID, cwd string) error {
	return a.manager.LoadSession(sessionID, cwd)
}

func (a *App) SendPrompt(sessionID, message string) error {
	return a.manager.Prompt(sessionID, message)
}

func (a *App) GetConversation(sessionID string) (*agent.Conversation, error) {
	return a.manager.GetConversation(sessionID)
}

func (a *App) ListSessions() ([]agent.Session, error) {
	return a.manager.ListSessions()
}

func (a *App) ListAgentSessions() ([]agent.Session, error) {
	cfg, err := a.config.Load()
	if err != nil {
		return nil, err
	}

	var sessions []agent.Session
	for _, endpoint := range cfg.Agents {
		if !endpoint.IsEnabled() {
			continue
		}
		if endpoint.Type != "" && endpoint.Type != "stdio" {
			continue
		}
		if strings.TrimSpace(endpoint.Command) == "" {
			continue
		}

		client, err := agentclient.NewForEndpoint(endpoint, a.config.Load)
		if err != nil {
			log.Printf("[agent] ListAgentSessions dial %s: %v", endpoint.ID, err)
			continue
		}
		list, err := listSessionsForClient(client)
		client.Close()
		if err != nil {
			log.Printf("[agent] ListAgentSessions list %s: %v", endpoint.ID, err)
			continue
		}
		for _, session := range list {
			session.AgentID = endpoint.ID
			session.AgentName = endpoint.Name
			if session.AgentName == "" {
				session.AgentName = endpoint.ID
			}
			session.AgentCommand = commandString(endpoint)
			session.Key = session.AgentID + ":" + session.ID
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (a *App) CloseSession(sessionID string) error {
	return a.manager.CloseSession(sessionID)
}

func (a *App) CancelSession(sessionID string) error {
	return a.manager.CancelSession(sessionID)
}

func (a *App) DeleteSession(sessionID string) error {
	return a.manager.DeleteSession(sessionID)
}

func (a *App) DisconnectAgent() error {
	return a.manager.Disconnect()
}

func (a *App) ReconnectAgent() error {
	return a.manager.Reconnect()
}

func (a *App) MiyaConfigPath() string {
	return a.config.Path()
}

func (a *App) LoadMiyaConfig() (*miyaconfig.Config, error) {
	return a.config.Load()
}

func (a *App) SaveMiyaConfig(cfg *miyaconfig.Config) error {
	return a.config.Save(cfg)
}

func (a *App) ChannelsServiceStatus() channelservice.Status {
	return a.channels.Status()
}

func (a *App) StartChannelsService() (channelservice.Status, error) {
	return a.channels.Start(a.ctx)
}

func (a *App) StopChannelsService() (channelservice.Status, error) {
	return a.channels.Stop()
}

func listSessionsForClient(client *acp.Client) ([]agent.Session, error) {
	_, err := client.Initialize(&acp.InitializeRequest{
		ProtocolVersion:    1,
		ClientCapabilities: acp.DefaultClientCapabilities(),
		ClientInfo:         &acp.Implementation{Name: "miya-desktop", Version: "0.1.0"},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}
	if err := client.SendNotification("notifications/initialized", struct{}{}); err != nil {
		return nil, fmt.Errorf("initialized notification: %w", err)
	}
	resp, err := client.ListSessions(&acp.ListSessionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	sessions := make([]agent.Session, 0, len(resp.Sessions))
	for _, s := range resp.Sessions {
		session := agent.Session{
			ID:  string(s.SessionID),
			Cwd: s.Cwd,
		}
		if s.Title != nil {
			session.Title = *s.Title
		}
		if s.UpdatedAt != nil {
			session.UpdatedAt = *s.UpdatedAt
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func commandString(endpoint miyaconfig.ACPAgentConfig) string {
	if endpoint.Command == "" {
		return ""
	}
	return strings.Join(append([]string{endpoint.Command}, endpoint.Args...), " ")
}
