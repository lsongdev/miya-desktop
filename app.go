package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"wails-app/internal/agent"
	"wails-app/internal/agentclient"
	channelservice "wails-app/internal/channels"
	miyaconfig "wails-app/internal/config"

	"github.com/lsongdev/miya-agents/acp"
	"github.com/lsongdev/miya-agents/anthropic"
	"github.com/lsongdev/miya-agents/openai"
)

// App struct
type App struct {
	ctx      context.Context
	manager  *agent.Manager
	config   *miyaconfig.Service
	channels *channelservice.Service
}

func builtinAgentEndpoint() miyaconfig.ACPAgentConfig {
	return miyaconfig.ACPAgentConfig{
		ID:      "miya",
		Name:    "Miya Agents",
		Enabled: boolPtr(true),
		Type:    "builtin",
		Command: "miya-agent",
		Args:    []string{"acp"},
	}
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
	cfg, err := a.config.Load()
	if err != nil {
		log.Printf("[channels] config load failed during auto-start: %v", err)
		return
	}
	if channelsEnabled(cfg) {
		go func() {
			if _, err := a.channels.Start(ctx); err != nil {
				log.Printf("[channels] auto-start failed: %v", err)
			}
		}()
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.channels != nil {
		_, _ = a.channels.Stop()
	}
	if a.manager != nil {
		_ = a.manager.Disconnect()
	}
}

func channelsEnabled(cfg *miyaconfig.Config) bool {
	return cfg.ChannelsEnabled == nil || *cfg.ChannelsEnabled
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) ConnectAgent(command string) error {
	return a.manager.Connect(command)
}

func (a *App) ConnectConfiguredAgent(agentID string) error {
	if agentID == "miya" {
		return a.manager.ConnectEndpoint(builtinAgentEndpoint())
	}
	cfg, err := a.config.Load()
	if err != nil {
		return err
	}
	for _, endpoint := range cfg.Agents {
		if endpoint.ID == agentID {
			if !endpoint.IsEnabled() {
				return fmt.Errorf("agent %q is disabled", agentID)
			}
			return a.manager.ConnectEndpoint(endpoint)
		}
	}
	return fmt.Errorf("agent %q not found", agentID)
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
	configuredAgents := make([]miyaconfig.ACPAgentConfig, 0, len(cfg.Agents))
	for _, endpoint := range cfg.Agents {
		if endpoint.ID != "miya" {
			configuredAgents = append(configuredAgents, endpoint)
		}
	}
	for _, endpoint := range append([]miyaconfig.ACPAgentConfig{builtinAgentEndpoint()}, configuredAgents...) {
		if !endpoint.IsEnabled() {
			continue
		}
		if endpoint.Type != "builtin" && endpoint.Type != "inprocess" && strings.TrimSpace(endpoint.Command) == "" {
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

func (a *App) FetchProviderModels(providerID string) ([]string, error) {
	cfg, err := a.config.Load()
	if err != nil {
		return nil, err
	}
	provider, ok := cfg.Providers[providerID]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerID)
	}
	if strings.TrimSpace(provider.APIKey) == "" {
		return nil, fmt.Errorf("provider %q has no API key", providerID)
	}

	providerType := strings.ToLower(strings.TrimSpace(provider.Type))
	if providerType == "" {
		providerType = "openai"
	}
	switch providerType {
	case "anthropic":
		return fetchAnthropicModels(provider.APIBase, provider.APIKey)
	default:
		return fetchOpenAIModels(provider.APIBase, provider.APIKey)
	}
}

func fetchOpenAIModels(apiBase, apiKey string) ([]string, error) {
	client, err := openai.NewClient(&openai.Configuration{
		API:    providerAPIBase(apiBase, "https://api.openai.com/v1"),
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	client.SetHTTPClient(&http.Client{Timeout: 15 * time.Second})
	modelsResp, err := client.Models()
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(modelsResp))
	for _, model := range modelsResp {
		if model.ID != "" {
			models = append(models, model.ID)
		}
	}
	sort.Strings(models)
	return models, nil
}

func fetchAnthropicModels(apiBase, apiKey string) ([]string, error) {
	client := anthropic.NewClient(&anthropic.Configuration{
		API:    providerAPIBase(apiBase, "https://api.anthropic.com"),
		APIKey: apiKey,
	})
	client.SetHTTPClient(&http.Client{Timeout: 15 * time.Second})
	modelsResp, err := client.Models(context.Background())
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(modelsResp))
	for _, model := range modelsResp {
		if model.ID != "" {
			models = append(models, model.ID)
		}
	}
	sort.Strings(models)
	return models, nil
}

func providerAPIBase(apiBase, fallback string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		return fallback
	}
	return base
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

func boolPtr(v bool) *bool {
	return &v
}
