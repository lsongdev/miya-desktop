package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"miya-desktop/internal/acpadapter"
	"miya-desktop/internal/agentclient"
	miyaconfig "miya-desktop/internal/config"
	"miya-desktop/internal/conversation"

	"github.com/lsongdev/miya-agents/acp"
)

type EventEmitter func(name string, data ...any)

type Session struct {
	ID           string `json:"id"`
	Key          string `json:"key,omitempty"`
	Cwd          string `json:"cwd"`
	Title        string `json:"title,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	AgentID      string `json:"agentId,omitempty"`
	AgentName    string `json:"agentName,omitempty"`
	AgentCommand string `json:"agentCommand,omitempty"`
	ACPProfile   string `json:"-"`
}

type AgentInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	LoadSession bool   `json:"loadSession"`
}

type ContentChunk = acpadapter.ContentChunk
type UpdateEvent = acpadapter.Event
type Conversation = conversation.Conversation

type Manager struct {
	client         *acp.Client
	ctx            context.Context
	loadConfig     agentclient.ConfigLoader
	mu             sync.Mutex
	agentInfo      *AgentInfo
	command        string
	endpoint       *miyaconfig.ACPAgentConfig
	initName       string
	initVer        string
	store          *conversation.Store
	emit           EventEmitter
	replayMu       sync.Mutex
	replays        map[string]*sessionReplay
	loadedSessions map[string]bool
}

type sessionReplay struct {
	store        *conversation.Store
	showProgress bool
	lastEmit     time.Time
}

func New(ctx context.Context, loadConfig agentclient.ConfigLoader, emit EventEmitter) *Manager {
	if emit == nil {
		emit = func(string, ...any) {}
	}
	return &Manager{
		ctx:            ctx,
		loadConfig:     loadConfig,
		store:          conversation.NewStore(),
		emit:           emit,
		replays:        make(map[string]*sessionReplay),
		loadedSessions: make(map[string]bool),
	}
}

func (m *Manager) emitEvent(name string, data any) {
	if m.emit != nil {
		m.emit(name, data)
	}
}

func (m *Manager) currentAgentID() string {
	if m.endpoint == nil || strings.TrimSpace(m.endpoint.ID) == "" {
		return ""
	}
	return m.endpoint.ID
}

func (m *Manager) currentAgentName() string {
	if m.endpoint == nil {
		return ""
	}
	if name := strings.TrimSpace(m.endpoint.Name); name != "" {
		return name
	}
	return strings.TrimSpace(m.endpoint.ID)
}

func (m *Manager) currentModel() string {
	if m.loadConfig == nil || m.endpoint == nil || m.endpoint.Type != "builtin" {
		return ""
	}
	cfg, err := m.loadConfig()
	if err != nil || cfg == nil {
		return ""
	}
	profileID := strings.TrimSpace(m.endpoint.Profile)
	if profile := cfg.Profiles[profileID]; profile != nil {
		return strings.TrimSpace(profile.ModelName)
	}
	return ""
}

func (m *Manager) annotateModel(conversationID string) {
	model := m.currentModel()
	if model == "" {
		return
	}
	if snapshot, ok := m.store.SetModel(conversationID, model); ok {
		m.emitEvent("conversation:update", snapshot)
	}
}

func (m *Manager) scopedSessionID(sessionID string) string {
	agentID := m.currentAgentID()
	if agentID == "" || strings.Contains(sessionID, ":") {
		return sessionID
	}
	return agentID + ":" + sessionID
}

func (m *Manager) splitSessionRef(sessionRef string) (conversationID string, acpSessionID string) {
	if agentID, raw, ok := strings.Cut(sessionRef, ":"); ok && agentID != "" && raw != "" {
		return sessionRef, raw
	}
	return m.scopedSessionID(sessionRef), sessionRef
}

func (m *Manager) Connect(command string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.connectLocked(command)
}

func (m *Manager) ConnectEndpoint(endpoint miyaconfig.ACPAgentConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.connectEndpointLocked(endpoint)
}

func (m *Manager) connectLocked(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("agent: empty command")
	}
	endpoint := miyaconfig.ACPAgentConfig{
		Type:    "stdio",
		Command: parts[0],
		Args:    parts[1:],
	}
	return m.connectEndpointLocked(endpoint)
}

func (m *Manager) connectEndpointLocked(endpoint miyaconfig.ACPAgentConfig) error {
	if strings.TrimSpace(endpoint.ID) == "" || strings.Contains(endpoint.ID, ":") {
		return fmt.Errorf("agent: invalid endpoint id %q", endpoint.ID)
	}
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	client, err := agentclient.NewForEndpoint(endpoint, m.loadConfig)
	if err != nil {
		return fmt.Errorf("agent: dial: %w", err)
	}

	client.OnNotification(acp.NewNotificationHandler(&managerNotificationReceiver{manager: m}))

	m.client = client
	m.command = endpointCommand(endpoint)
	m.endpoint = &endpoint
	clear(m.loadedSessions)
	log.Printf("[agent] Connected to %q", m.command)
	return nil
}

func (m *Manager) Reconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.reconnectLocked()
}

type managerNotificationReceiver struct {
	acp.DefaultNotificationReceiver
	manager *Manager
}

func (r *managerNotificationReceiver) SessionUpdate(notification *acp.SessionNotification) {
	event, err := acpadapter.ParseSessionUpdate(notification.Update)
	if err != nil {
		log.Printf("[agent] Failed to parse session/update: %v", err)
		return
	}
	sessionID := string(notification.SessionID)
	conversationID := r.manager.scopedSessionID(sessionID)
	store, emitConversation := r.manager.storeForUpdate(conversationID)
	store.EnsureSessionWithACP(conversationID, sessionID, "")
	var snapshot conversation.Snapshot
	if emitConversation {
		snapshot = store.ApplyACPEvent(conversationID, event)
	} else {
		store.ApplyACPEventQuiet(conversationID, event)
	}
	r.manager.emitEvent("session:update", map[string]any{
		"sessionId": conversationID,
		"event":     event,
	})
	if emitConversation {
		r.manager.emitEvent("conversation:update", snapshot)
	}
}

func (m *Manager) storeForUpdate(conversationID string) (*conversation.Store, bool) {
	m.replayMu.Lock()
	defer m.replayMu.Unlock()
	if replay := m.replays[conversationID]; replay != nil {
		if !replay.showProgress {
			return replay.store, false
		}
		now := time.Now()
		if replay.lastEmit.IsZero() || now.Sub(replay.lastEmit) >= 200*time.Millisecond {
			replay.lastEmit = now
			return replay.store, true
		}
		return replay.store, false
	}
	return m.store, true
}

func (r *managerNotificationReceiver) InvalidNotification(method string, params json.RawMessage, err error) {
	log.Printf("[agent] Failed to parse notification %s: %v", method, err)
}

// reconnectLocked reconnects and re-initializes the ACP session.
// Must be called with m.mu held.
func (m *Manager) reconnectLocked() error {
	if m.command == "" {
		return fmt.Errorf("agent: no command configured")
	}
	if m.endpoint != nil {
		if err := m.connectEndpointLocked(*m.endpoint); err != nil {
			return err
		}
	} else if err := m.connectLocked(m.command); err != nil {
		return err
	}
	if m.initName != "" {
		if err := m.initializeLocked(); err != nil {
			return fmt.Errorf("agent: re-initialize after reconnect: %w", err)
		}
	}
	return nil
}

func endpointCommand(endpoint miyaconfig.ACPAgentConfig) string {
	if endpoint.Command == "" {
		return endpoint.Type
	}
	return strings.Join(append([]string{endpoint.Command}, endpoint.Args...), " ")
}

// initializeLocked sends the ACP initialize handshake.
// Must be called with m.mu held and m.client non-nil.
func (m *Manager) initializeLocked() error {
	resp, err := m.client.Initialize(&acp.InitializeRequest{
		ProtocolVersion:    1,
		ClientCapabilities: acp.DefaultClientCapabilities(),
		ClientInfo: &acp.Implementation{
			Name:    m.initName,
			Version: m.initVer,
		},
	})
	if err != nil {
		return fmt.Errorf("agent: initialize: %w", err)
	}
	info := &AgentInfo{
		LoadSession: resp.AgentCapabilities.LoadSession,
	}
	if resp.AgentInfo != nil {
		info.Name = resp.AgentInfo.Name
		info.Version = resp.AgentInfo.Version
	}
	m.agentInfo = info
	return nil
}

func (m *Manager) clientForCall() (*acp.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client == nil {
		return nil, fmt.Errorf("agent: not connected")
	}
	return m.client, nil
}

func (m *Manager) Initialize(name, version string) (*AgentInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureConnected(); err != nil {
		return nil, err
	}

	m.initName = name
	m.initVer = version

	if err := m.initializeLocked(); err != nil {
		return nil, err
	}

	return m.agentInfo, nil
}

func (m *Manager) ensureConnected() error {
	if m.client != nil {
		return nil
	}
	return fmt.Errorf("agent: not connected")
}

func (m *Manager) LoadSession(sessionID, cwd string) error {
	client, err := m.clientForCall()
	if err != nil {
		return err
	}
	conversationID, acpSessionID := m.splitSessionRef(sessionID)
	log.Printf("[agent] LoadSession: id=%q acp=%q cwd=%q", conversationID, acpSessionID, cwd)
	m.mu.Lock()
	alreadyLoaded := m.loadedSessions[conversationID]
	m.mu.Unlock()
	if alreadyLoaded {
		if snapshot, ok := m.store.Snapshot(conversationID); ok {
			m.emitEvent("conversation:update", snapshot)
		}
		return nil
	}

	hasExistingMessages := m.store.HasMessages(conversationID)
	replayStore := conversation.NewStore()
	snapshot := replayStore.ResetSessionWithACP(conversationID, acpSessionID, cwd)
	if !hasExistingMessages {
		m.emitEvent("conversation:update", snapshot)
	}
	m.replayMu.Lock()
	m.replays[conversationID] = &sessionReplay{store: replayStore, showProgress: !hasExistingMessages}
	m.replayMu.Unlock()
	defer func() {
		m.replayMu.Lock()
		delete(m.replays, conversationID)
		m.replayMu.Unlock()
	}()

	if _, err := client.LoadSession(&acp.LoadSessionRequest{
		SessionID:  acp.SessionID(acpSessionID),
		Cwd:        cwd,
		McpServers: []acp.McpServer{},
	}); err != nil {
		return fmt.Errorf("agent: load session: %w", err)
	}
	if model := m.currentModel(); model != "" {
		replayStore.SetModel(conversationID, model)
	}
	if replayed, ok := replayStore.CompleteStreaming(conversationID); ok {
		final := m.store.Replace(replayed.Conversation, "replay_completed")
		m.mu.Lock()
		m.loadedSessions[conversationID] = true
		m.mu.Unlock()
		m.emitEvent("conversation:update", final)
	}
	return nil
}

func (m *Manager) NewSession(cwd string) (*Session, error) {
	client, err := m.clientForCall()
	if err != nil {
		return nil, err
	}
	meta := acp.Meta{}
	if m.endpoint != nil && strings.TrimSpace(m.endpoint.Profile) != "" {
		meta[acp.MiyaProfileMetaKey] = strings.TrimSpace(m.endpoint.Profile)
	}
	resp, err := client.NewSession(&acp.NewSessionRequest{
		Cwd:        cwd,
		McpServers: []acp.McpServer{},
		Meta:       meta,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: new session: %w", err)
	}
	sessionID := string(resp.SessionID)
	conversationID := m.scopedSessionID(sessionID)
	log.Printf("[agent] NewSession created: id=%q", sessionID)
	session := &Session{
		ID:        sessionID,
		Key:       conversationID,
		Cwd:       cwd,
		AgentID:   m.currentAgentID(),
		AgentName: m.currentAgentName(),
	}
	if resolved, ok := resp.Meta[acp.MiyaProfileMetaKey].(string); ok && strings.TrimSpace(resolved) != "" {
		session.ACPProfile = strings.TrimSpace(resolved)
	}
	snapshot := m.store.RegisterSessionWithACP(conversationID, sessionID, cwd)
	m.mu.Lock()
	m.loadedSessions[conversationID] = true
	m.mu.Unlock()
	m.emitEvent("conversation:update", snapshot)
	m.annotateModel(conversationID)
	return session, nil
}

func (m *Manager) Prompt(sessionID, message string) error {
	conversationID, acpSessionID := m.splitSessionRef(sessionID)
	m.mu.Lock()
	m.loadedSessions[conversationID] = true
	m.mu.Unlock()
	log.Printf("[agent] Prompt: session=%q acp=%q message=%q", conversationID, acpSessionID, message)
	m.store.RegisterSessionWithACP(conversationID, acpSessionID, "")
	m.annotateModel(conversationID)
	snapshot := m.store.AddLocalUserMessage(conversationID, message)
	m.emitEvent("conversation:update", snapshot)

	client, err := m.clientForCall()
	if err != nil {
		return err
	}

	resp, err := client.Prompt(&acp.PromptRequest{
		SessionID: acp.SessionID(acpSessionID),
		Prompt: []acp.ContentBlock{
			{Type: "text", Text: message},
		},
	})
	if err != nil {
		return fmt.Errorf("agent: prompt: %w", err)
	}

	if snapshot, ok := m.store.CompleteStreaming(conversationID); ok {
		m.emitEvent("conversation:update", map[string]any{
			"conversation": snapshot.Conversation,
			"eventType":    snapshot.EventType,
			"stopReason":   resp.StopReason,
		})
	}
	return nil
}

func (m *Manager) CancelSession(sessionID string) error {
	m.mu.Lock()
	client := m.client
	m.mu.Unlock()

	if client == nil {
		return fmt.Errorf("agent: not connected")
	}
	_, acpSessionID := m.splitSessionRef(sessionID)
	if err := client.CancelSession(acp.SessionID(acpSessionID)); err != nil {
		return fmt.Errorf("agent: cancel session: %w", err)
	}
	return nil
}

func (m *Manager) ListSessions() ([]Session, error) {
	client, err := m.clientForCall()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListSessions(&acp.ListSessionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("agent: list sessions: %w", err)
	}
	sessions := make([]Session, 0, len(resp.Sessions))
	for _, s := range resp.Sessions {
		session := Session{ID: string(s.SessionID), Cwd: s.Cwd}
		session.Key = m.scopedSessionID(session.ID)
		session.AgentID = m.currentAgentID()
		session.AgentName = m.currentAgentID()
		if s.Title != nil {
			session.Title = *s.Title
		}
		if s.UpdatedAt != nil {
			session.UpdatedAt = *s.UpdatedAt
		}
		if profileID, ok := s.Meta[acp.MiyaProfileMetaKey].(string); ok {
			session.ACPProfile = strings.TrimSpace(profileID)
		}
		m.store.RegisterSessionWithACP(session.Key, session.ID, session.Cwd)
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (m *Manager) GetConversation(sessionID string) (*conversation.Conversation, error) {
	snapshot, ok := m.store.Snapshot(sessionID)
	if !ok {
		return nil, nil
	}
	return &snapshot.Conversation, nil
}

func (m *Manager) CloseSession(sessionID string) error {
	client, err := m.clientForCall()
	if err != nil {
		return err
	}
	_, acpSessionID := m.splitSessionRef(sessionID)
	_, err = client.CloseSession(&acp.CloseSessionRequest{SessionID: acp.SessionID(acpSessionID)})
	return err
}

func (m *Manager) DeleteSession(sessionID string) error {
	log.Printf("[agent] DeleteSession: id=%q", sessionID)
	client, err := m.clientForCall()
	if err != nil {
		return err
	}
	_, acpSessionID := m.splitSessionRef(sessionID)
	_, err = client.DeleteSession(&acp.DeleteSessionRequest{SessionID: acp.SessionID(acpSessionID)})
	if err != nil {
		log.Printf("[agent] DeleteSession failed: %v", err)
		return err
	}
	conversationID, _ := m.splitSessionRef(sessionID)
	m.store.Delete(conversationID)
	m.mu.Lock()
	delete(m.loadedSessions, conversationID)
	m.mu.Unlock()
	return nil
}

func (m *Manager) Disconnect() error {
	m.mu.Lock()
	client := m.client
	m.client = nil
	m.mu.Unlock()

	if client != nil {
		return client.Close()
	}
	return nil
}
