package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"wails-app/internal/acpadapter"
	"wails-app/internal/agentclient"
	miyaconfig "wails-app/internal/config"
	"wails-app/internal/conversation"

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
	client               *acp.Client
	ctx                  context.Context
	loadConfig           agentclient.ConfigLoader
	mu                   sync.Mutex
	agentInfo            *AgentInfo
	command              string
	endpoint             *miyaconfig.ACPAgentConfig
	initName             string
	initVer              string
	store                *conversation.Store
	emit                 EventEmitter
	conversationEmitMu   sync.Mutex
	lastConversationEmit map[string]time.Time
}

func New(ctx context.Context, loadConfig agentclient.ConfigLoader, emit EventEmitter) *Manager {
	if emit == nil {
		emit = func(string, ...any) {}
	}
	return &Manager{
		ctx:                  ctx,
		loadConfig:           loadConfig,
		store:                conversation.NewStore(),
		emit:                 emit,
		lastConversationEmit: make(map[string]time.Time),
	}
}

func (m *Manager) emitEvent(name string, data any) {
	if m.emit != nil {
		m.emit(name, data)
	}
}

func (m *Manager) shouldEmitConversationUpdate(sessionID string) bool {
	const minInterval = 50 * time.Millisecond

	m.conversationEmitMu.Lock()
	defer m.conversationEmitMu.Unlock()

	now := time.Now()
	if now.Sub(m.lastConversationEmit[sessionID]) < minInterval {
		return false
	}
	m.lastConversationEmit[sessionID] = now
	return true
}

func (m *Manager) currentAgentID() string {
	if m.endpoint == nil || strings.TrimSpace(m.endpoint.ID) == "" {
		return ""
	}
	return m.endpoint.ID
}

func (m *Manager) currentModel() string {
	if m.loadConfig == nil {
		return ""
	}
	cfg, err := m.loadConfig()
	if err != nil || cfg == nil {
		return ""
	}
	if profile := cfg.Profiles["default"]; profile != nil {
		return strings.TrimSpace(profile.ModelName)
	}
	if len(cfg.Profiles) == 1 {
		for _, profile := range cfg.Profiles {
			if profile != nil {
				return strings.TrimSpace(profile.ModelName)
			}
		}
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
	r.manager.store.EnsureSessionWithACP(conversationID, sessionID, "")
	snapshot := r.manager.store.ApplyACPEvent(conversationID, event)
	r.manager.emitEvent("session:update", map[string]any{
		"sessionId": conversationID,
		"event":     event,
	})
	if r.manager.shouldEmitConversationUpdate(conversationID) {
		r.manager.emitEvent("conversation:update", snapshot)
	}
}

func (r *managerNotificationReceiver) InvalidNotification(method string, params json.RawMessage, err error) {
	log.Printf("[agent] Failed to parse notification %s: %v", method, err)
}

// isConnectionError checks if the error indicates a dead connection.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "connection closed") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "use of closed") ||
		strings.Contains(s, "io: read/write on closed pipe") ||
		strings.Contains(s, "file already closed")
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

// runWithRetry executes fn under the lock. If it fails due to a closed
// connection, reconnects, re-initializes, and retries once.
func (m *Manager) runWithRetry(fn func(client *acp.Client) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		if m.command == "" {
			return fmt.Errorf("agent: not connected")
		}
		log.Printf("[agent] Client nil, reconnecting before operation...")
		if err := m.reconnectLocked(); err != nil {
			return fmt.Errorf("agent: connect: %w", err)
		}
	}

	// First attempt
	err := fn(m.client)
	if err == nil {
		return nil
	}

	if !isConnectionError(err) {
		return err
	}

	// Reconnect, re-initialize, and retry
	log.Printf("[agent] Connection lost (%v), reconnecting...", err)
	if reconnErr := m.reconnectLocked(); reconnErr != nil {
		return fmt.Errorf("agent: reconnect failed: %w (original: %v)", reconnErr, err)
	}

	log.Printf("[agent] Reconnected, retrying operation...")
	return fn(m.client)
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
	if m.command == "" {
		return fmt.Errorf("agent: not connected")
	}
	log.Printf("[agent] Client nil, reconnecting...")
	return m.connectLocked(m.command)
}

func (m *Manager) clientForLongCall() (*acp.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		if m.command == "" {
			return nil, fmt.Errorf("agent: not connected")
		}
		log.Printf("[agent] Client nil, reconnecting before long operation...")
		if err := m.reconnectLocked(); err != nil {
			return nil, fmt.Errorf("agent: connect: %w", err)
		}
	}
	return m.client, nil
}

func (m *Manager) LoadSession(sessionID, cwd string) error {
	return m.runWithRetry(func(client *acp.Client) error {
		conversationID, acpSessionID := m.splitSessionRef(sessionID)
		log.Printf("[agent] LoadSession: id=%q acp=%q cwd=%q", conversationID, acpSessionID, cwd)
		if m.store.HasMessages(conversationID) {
			if snapshot, ok := m.store.Snapshot(conversationID); ok {
				m.emitEvent("conversation:update", snapshot)
			}
			return nil
		}
		snapshot := m.store.ResetSessionWithACP(conversationID, acpSessionID, cwd)
		m.emitEvent("conversation:update", snapshot)
		m.annotateModel(conversationID)
		_, err := client.LoadSession(&acp.LoadSessionRequest{
			SessionID:  acp.SessionID(acpSessionID),
			Cwd:        cwd,
			McpServers: []acp.McpServer{},
		})
		if err != nil {
			return fmt.Errorf("agent: load session: %w", err)
		}
		if snapshot, ok := m.store.CompleteStreaming(conversationID); ok {
			m.emitEvent("conversation:update", snapshot)
		}
		return nil
	})
}

func (m *Manager) NewSession(cwd string) (*Session, error) {
	var session *Session
	err := m.runWithRetry(func(client *acp.Client) error {
		resp, err := client.NewSession(&acp.NewSessionRequest{
			Cwd:        cwd,
			McpServers: []acp.McpServer{},
		})
		if err != nil {
			return fmt.Errorf("agent: new session: %w", err)
		}
		sessionID := string(resp.SessionID)
		conversationID := m.scopedSessionID(sessionID)
		log.Printf("[agent] NewSession created: id=%q", sessionID)
		session = &Session{
			ID:        sessionID,
			Key:       conversationID,
			Cwd:       cwd,
			AgentID:   m.currentAgentID(),
			AgentName: m.currentAgentID(),
		}
		snapshot := m.store.RegisterSessionWithACP(conversationID, sessionID, cwd)
		m.emitEvent("conversation:update", snapshot)
		m.annotateModel(conversationID)
		return nil
	})
	return session, err
}

func (m *Manager) Prompt(sessionID, message string) error {
	conversationID, acpSessionID := m.splitSessionRef(sessionID)
	log.Printf("[agent] Prompt: session=%q acp=%q message=%q", conversationID, acpSessionID, message)
	m.store.RegisterSessionWithACP(conversationID, acpSessionID, "")
	m.annotateModel(conversationID)
	snapshot := m.store.AddLocalUserMessage(conversationID, message)
	m.emitEvent("conversation:update", snapshot)

	client, err := m.clientForLongCall()
	if err != nil {
		return err
	}

	resp, err := client.Prompt(&acp.PromptRequest{
		SessionID: acp.SessionID(acpSessionID),
		Prompt: []acp.ContentBlock{
			{Type: "text", Text: message},
		},
	})
	if err != nil && isConnectionError(err) {
		m.mu.Lock()
		reconnErr := m.reconnectLocked()
		if reconnErr == nil {
			client = m.client
		}
		m.mu.Unlock()
		if reconnErr != nil {
			return fmt.Errorf("agent: reconnect failed: %w (original: %v)", reconnErr, err)
		}
		resp, err = client.Prompt(&acp.PromptRequest{
			SessionID: acp.SessionID(acpSessionID),
			Prompt: []acp.ContentBlock{
				{Type: "text", Text: message},
			},
		})
	}
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
	var sessions []Session
	err := m.runWithRetry(func(client *acp.Client) error {
		resp, err := client.ListSessions(&acp.ListSessionsRequest{})
		if err != nil {
			return fmt.Errorf("agent: list sessions: %w", err)
		}
		sessions = make([]Session, 0, len(resp.Sessions))
		for _, s := range resp.Sessions {
			session := Session{
				ID:  string(s.SessionID),
				Cwd: s.Cwd,
			}
			session.Key = m.scopedSessionID(session.ID)
			session.AgentID = m.currentAgentID()
			session.AgentName = m.currentAgentID()
			if s.Title != nil {
				session.Title = *s.Title
			}
			if s.UpdatedAt != nil {
				session.UpdatedAt = *s.UpdatedAt
			}
			m.store.RegisterSessionWithACP(session.Key, session.ID, session.Cwd)
			sessions = append(sessions, session)
		}
		return nil
	})
	return sessions, err
}

func (m *Manager) GetConversation(sessionID string) (*conversation.Conversation, error) {
	snapshot, ok := m.store.Snapshot(sessionID)
	if !ok {
		return nil, nil
	}
	return &snapshot.Conversation, nil
}

func (m *Manager) CloseSession(sessionID string) error {
	return m.runWithRetry(func(client *acp.Client) error {
		_, acpSessionID := m.splitSessionRef(sessionID)
		_, err := client.CloseSession(&acp.CloseSessionRequest{
			SessionID: acp.SessionID(acpSessionID),
		})
		return err
	})
}

func (m *Manager) DeleteSession(sessionID string) error {
	log.Printf("[agent] DeleteSession: id=%q", sessionID)
	err := m.runWithRetry(func(client *acp.Client) error {
		_, acpSessionID := m.splitSessionRef(sessionID)
		_, err := client.DeleteSession(&acp.DeleteSessionRequest{
			SessionID: acp.SessionID(acpSessionID),
		})
		return err
	})
	if err != nil {
		log.Printf("[agent] DeleteSession failed: %v", err)
	}
	return err
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
