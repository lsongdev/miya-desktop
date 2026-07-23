package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"miya-desktop/internal/agent"
	"miya-desktop/internal/agentclient"
	channelservice "miya-desktop/internal/channels"
	miyaconfig "miya-desktop/internal/config"

	"github.com/lsongdev/miya-agents/acp"
	"github.com/lsongdev/miya-agents/anthropic"
	"github.com/lsongdev/miya-agents/openai"
	"github.com/lsongdev/miya-agents/skills"
	channelpkg "github.com/lsongdev/miya-channels/channels"
	"github.com/wailsapp/wails/v3/pkg/application"
	wailsupdater "github.com/wailsapp/wails/v3/pkg/updater"
)

// App struct
type App struct {
	ctx     context.Context
	emit    agent.EventEmitter
	manager *agent.Manager

	config   *miyaconfig.Service
	channels *channelservice.Service
	updater  *wailsupdater.Updater

	wechatLoginMu     sync.Mutex
	wechatLoginCancel context.CancelFunc
	wechatLoginID     int64
}

type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
	Prompt      string `json:"prompt,omitempty"`
}

type AttachmentData struct {
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
	Data     string `json:"data"`
}

type UpdateCheckResult struct {
	Status         string `json:"status"`
	Message        string `json:"message"`
	CurrentVersion string `json:"currentVersion"`
	State          string `json:"state"`
}

// NewApp creates a new App application struct
func NewApp(emit agent.EventEmitter) *App {
	return &App{emit: emit}
}

func (a *App) setUpdater(updater *wailsupdater.Updater) {
	a.updater = updater
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.startup(ctx)
	return nil
}

func (a *App) ServiceShutdown() error {
	a.shutdown()
	return nil
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.config = miyaconfig.NewService()
	a.manager = agent.New(ctx, a.config.Load, a.emit)
	a.channels = channelservice.NewService(a.config.Load, a.emit)
	if err := a.ensureBundledDesktopSkill(); err != nil {
		log.Printf("[skills] install bundled desktop skill: %v", err)
	}
	hasConfig := a.config.Exists()
	cfg, err := a.config.Load()
	if err != nil {
		log.Printf("[channels] config load failed during auto-start: %v", err)
		return
	}
	if hasConfig && channelsEnabled(cfg) {
		go func() {
			if _, err := a.channels.Start(ctx); err != nil {
				log.Printf("[channels] auto-start failed: %v", err)
			}
		}()
	}
}

func (a *App) shutdown() {
	a.wechatLoginMu.Lock()
	if a.wechatLoginCancel != nil {
		a.wechatLoginCancel()
		a.wechatLoginCancel = nil
	}
	a.wechatLoginMu.Unlock()
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

func (a *App) CheckForUpdates() (*UpdateCheckResult, error) {
	if a.updater == nil {
		return nil, fmt.Errorf("updater is not available")
	}
	if !isReleaseVersion(appVersion) {
		return nil, fmt.Errorf("updates are only available in release builds")
	}
	if err := a.updater.CheckAndInstall(context.Background()); err != nil {
		return nil, err
	}
	state := string(a.updater.State())
	message := "Update check complete."
	switch a.updater.State() {
	case wailsupdater.StateUpToDate:
		message = "You're running the latest version."
	case wailsupdater.StateReady:
		message = "Update downloaded. Restart from the update window to finish installing."
	case wailsupdater.StateAvailable:
		message = "Update available."
	}
	return &UpdateCheckResult{
		Status:         state,
		Message:        message,
		CurrentVersion: appVersion,
		State:          state,
	}, nil
}

func isReleaseVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
		if len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}

func (a *App) OpenAttachment(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("attachment target is required")
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("invalid attachment target: %w", err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "file":
		path, err := fileURLPath(parsed)
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("attachment file is not accessible: %w", err)
		}
		return openWithDefaultApp(path)
	case "http", "https":
		if parsed.Host == "" {
			return fmt.Errorf("attachment URL is missing host")
		}
		return openWithDefaultApp(parsed.String())
	default:
		return fmt.Errorf("attachment scheme %q is not supported", parsed.Scheme)
	}
}

func (a *App) ReadAttachment(target string, maxBytes int64) (*AttachmentData, error) {
	if maxBytes <= 0 {
		maxBytes = 20 * 1024 * 1024
	}
	path, err := attachmentFilePath(target)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("attachment file is not accessible: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("attachment is a directory")
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("attachment is too large to preview")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read attachment: %w", err)
	}
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	return &AttachmentData{
		Name:     filepath.Base(path),
		MimeType: mimeType,
		Size:     info.Size(),
		Data:     base64.StdEncoding.EncodeToString(data),
	}, nil
}

func attachmentFilePath(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("attachment target is required")
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("invalid attachment target: %w", err)
	}
	if strings.ToLower(parsed.Scheme) != "file" {
		return "", fmt.Errorf("attachment scheme %q is not readable", parsed.Scheme)
	}
	return fileURLPath(parsed)
}

func fileURLPath(u *url.URL) (string, error) {
	path, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", fmt.Errorf("invalid file URL path: %w", err)
	}
	if runtime.GOOS == "windows" {
		if u.Host != "" {
			path = `\\` + u.Host + filepath.FromSlash(path)
		} else {
			if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
				path = path[1:]
			}
			path = filepath.FromSlash(path)
		}
	} else if u.Host != "" && u.Host != "localhost" {
		return "", fmt.Errorf("file URL host %q is not supported", u.Host)
	}
	if path == "" {
		return "", fmt.Errorf("file URL path is empty")
	}
	return path, nil
}

func openWithDefaultApp(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	configureCommand(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open attachment: %w", err)
	}
	return nil
}

func (a *App) ConnectAgent(command string) error {
	return a.manager.Connect(command)
}

func (a *App) ConnectConfiguredAgent(agentID string) error {
	cfg, err := a.config.Load()
	if err != nil {
		return err
	}
	endpoints, err := miyaconfig.AgentEndpoints(cfg)
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
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
	if strings.TrimSpace(version) == "" {
		version = appVersion
	}
	return a.manager.Initialize(name, version)
}

func (a *App) CreateSession(cwd string) (*agent.Session, error) {
	return a.manager.NewSession(a.normalizeCwd(cwd))
}

func (a *App) DefaultCwd() string {
	return a.ensureDefaultWorkspace()
}

func (a *App) normalizeCwd(cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd != "" {
		return cwd
	}
	return a.ensureDefaultWorkspace()
}

func (a *App) ensureDefaultWorkspace() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		workspace := filepath.Join(home, ".miya", "workspace")
		if mkErr := os.MkdirAll(workspace, 0755); mkErr == nil {
			return workspace
		}
	}
	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		return cwd
	}
	return "."
}

func (a *App) LoadSession(sessionID, cwd string) error {
	return a.manager.LoadSession(sessionID, a.normalizeCwd(cwd))
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
	endpoints, err := miyaconfig.AgentEndpoints(cfg)
	if err != nil {
		return nil, err
	}
	for _, endpoint := range endpoints {
		if !endpoint.IsEnabled() {
			continue
		}
		if endpoint.Type != "builtin" && strings.TrimSpace(endpoint.Command) == "" {
			continue
		}

		client, err := agentclient.NewForEndpoint(endpoint, a.config.Load)
		if err != nil {
			log.Printf("[agent] ListAgentSessions dial %s: %v", endpoint.ID, err)
			continue
		}
		list, err := listSessionsForClientWithTimeout(client, 3*time.Second)
		if err != nil {
			log.Printf("[agent] ListAgentSessions list %s: %v", endpoint.ID, err)
			continue
		}
		for _, session := range list {
			if endpoint.Type == "builtin" && session.ACPProfile != strings.TrimSpace(endpoint.Profile) {
				continue
			}
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

func listSessionsForClientWithTimeout(client *acp.Client, timeout time.Duration) ([]agent.Session, error) {
	type result struct {
		sessions []agent.Session
		err      error
	}
	done := make(chan result, 1)
	go func() {
		sessions, err := listSessionsForClient(client)
		done <- result{sessions: sessions, err: err}
	}()

	select {
	case result := <-done:
		_ = client.Close()
		return result.sessions, result.err
	case <-time.After(timeout):
		_ = client.Close()
		return nil, fmt.Errorf("list sessions timeout after %s", timeout)
	}
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

func (a *App) MiyaConfigExists() bool {
	return a.config.Exists()
}

func (a *App) LoadMiyaConfig() (*miyaconfig.Config, error) {
	return a.config.Load()
}

func (a *App) SaveMiyaConfig(cfg *miyaconfig.Config) error {
	return a.config.Save(cfg)
}

func (a *App) SkillsDirectory() string {
	return filepath.Join(filepath.Dir(a.config.Path()), "skills")
}

func (a *App) ListSkills() ([]SkillInfo, error) {
	dir := a.SkillsDirectory()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create skills directory: %w", err)
	}
	loaded, err := skills.LoadSkillsFromDirectory(dir)
	if err != nil {
		return nil, err
	}
	out := make([]SkillInfo, 0, len(loaded))
	for _, skill := range loaded {
		if skill == nil || strings.TrimSpace(skill.Name) == "" {
			continue
		}
		name := strings.TrimSpace(skill.Name)
		out = append(out, SkillInfo{
			Name:        name,
			Description: strings.TrimSpace(skill.Description),
			Path:        filepath.Join(dir, safeSkillName(name), "SKILL.md"),
			Prompt:      skill.Prompt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (a *App) InstallSkill(name, description, prompt string) (SkillInfo, error) {
	name = safeSkillName(name)
	description = strings.TrimSpace(description)
	prompt = strings.TrimSpace(prompt)
	if name == "" {
		return SkillInfo{}, fmt.Errorf("skill name is required")
	}
	if prompt == "" {
		return SkillInfo{}, fmt.Errorf("skill prompt is required")
	}
	dir := filepath.Join(a.SkillsDirectory(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return SkillInfo{}, fmt.Errorf("create skill directory: %w", err)
	}
	path := filepath.Join(dir, "SKILL.md")
	content := fmt.Sprintf("---\nname: %q\ndescription: %q\n---\n\n%s\n", name, description, prompt)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return SkillInfo{}, fmt.Errorf("write skill: %w", err)
	}
	return SkillInfo{Name: name, Description: description, Path: path, Prompt: prompt}, nil
}

func (a *App) DeleteSkill(name string) error {
	name = safeSkillName(name)
	if name == "" {
		return fmt.Errorf("skill name is required")
	}
	dir := filepath.Join(a.SkillsDirectory(), name)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

func safeSkillName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if (r == '-' || r == '_' || r == ' ' || r == '.') && !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
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
	return fetchProviderModels(providerID, provider)
}

func (a *App) FetchProviderModelsFromConfig(providerID string, provider miyaconfig.ProviderConfig) ([]string, error) {
	return fetchProviderModels(providerID, &provider)
}

func fetchProviderModels(providerID string, provider *miyaconfig.ProviderConfig) ([]string, error) {
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

func (a *App) StartWeChatLogin(rawConfig map[string]any) error {
	data, err := json.Marshal(rawConfig)
	if err != nil {
		return fmt.Errorf("marshal wechat config: %w", err)
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.wechatLoginMu.Lock()
	if a.wechatLoginCancel != nil {
		a.wechatLoginCancel()
	}
	a.wechatLoginID++
	loginID := a.wechatLoginID
	a.wechatLoginCancel = cancel
	a.wechatLoginMu.Unlock()

	go func() {
		defer func() {
			a.wechatLoginMu.Lock()
			if a.wechatLoginID == loginID {
				a.wechatLoginCancel = nil
			}
			a.wechatLoginMu.Unlock()
		}()

		cfg, err := channelpkg.LoginWeChat(ctx, data, channelpkg.ChannelOptions{
			Emit: func(event channelpkg.ChannelEvent) {
				a.emit("channel:event", channelservice.ChannelEvent{
					Channel:     event.Channel,
					Type:        event.Type,
					Status:      event.Status,
					QRCode:      event.QRCode,
					QRCodeURL:   event.QRCodeURL,
					QRCodeImage: event.QRCodeImage,
					Error:       event.Error,
				})
			},
		})
		if err != nil {
			if ctx.Err() == nil {
				a.emit("channel:event", channelservice.ChannelEvent{
					Channel: "wechat",
					Type:    "login",
					Status:  "error",
					Error:   err.Error(),
				})
			}
			return
		}
		a.emit("channel:event", channelservice.ChannelEvent{
			Channel:    "wechat",
			Type:       "login",
			Status:     "authenticated",
			Token:      cfg.Token,
			BaseURL:    cfg.BaseURL,
			CDNBaseURL: cfg.CDNBaseURL,
		})
	}()

	return nil
}

func listSessionsForClient(client *acp.Client) ([]agent.Session, error) {
	_, err := client.Initialize(&acp.InitializeRequest{
		ProtocolVersion:    1,
		ClientCapabilities: acp.DefaultClientCapabilities(),
		ClientInfo:         &acp.Implementation{Name: "miya-desktop", Version: appVersion},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
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
		if profileID, ok := s.Meta[acp.MiyaProfileMetaKey].(string); ok {
			session.ACPProfile = strings.TrimSpace(profileID)
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
