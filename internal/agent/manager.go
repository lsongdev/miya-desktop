package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/lsongdev/miya-agents/acp"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Session struct {
	ID        string `json:"id"`
	Cwd       string `json:"cwd"`
	Title     string `json:"title,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type AgentInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	LoadSession bool   `json:"loadSession"`
}

type ContentChunk struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Thought   string `json:"thought,omitempty"`
	Data      string `json:"data,omitempty"`
	Mime      string `json:"mime,omitempty"`
	MessageID string `json:"messageId,omitempty"`
}

type UpdateEvent struct {
	Type    string                 `json:"type"`
	Content *ContentChunk          `json:"content,omitempty"`
	Tool    *acp.ToolCall          `json:"tool,omitempty"`
	Plan    *acp.Plan              `json:"plan,omitempty"`
	Usage   *acp.UsageUpdate       `json:"usage,omitempty"`
	Mode    *acp.CurrentModeUpdate `json:"mode,omitempty"`
	Info    *acp.SessionInfoUpdate `json:"info,omitempty"`
}

type Manager struct {
	client    *acp.Client
	ctx       context.Context
	mu        sync.Mutex
	agentInfo *AgentInfo
	command   string
	initName  string
	initVer   string
}

func New(ctx context.Context) *Manager {
	return &Manager{ctx: ctx}
}

func (m *Manager) Connect(command string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.connectLocked(command)
}

func (m *Manager) connectLocked(command string) error {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("agent: empty command")
	}

	client, err := acp.DialStdio(parts[0], parts[1:]...)
	if err != nil {
		return fmt.Errorf("agent: dial: %w", err)
	}

	client.OnNotification(func(method string, params json.RawMessage) {
		m.handleNotification(method, params)
	})

	m.client = client
	m.command = command
	log.Printf("[agent] Connected to %q", command)
	return nil
}

func (m *Manager) Reconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.reconnectLocked()
}

func (m *Manager) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "session/update":
		var n struct {
			SessionID string            `json:"sessionId"`
			Update    acp.SessionUpdate `json:"update"`
		}
		if err := json.Unmarshal(params, &n); err != nil {
			return
		}
		event := m.convertUpdate(&n.Update)
		runtime.EventsEmit(m.ctx, "session:update", map[string]any{
			"sessionId": n.SessionID,
			"event":     event,
		})
	}
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
	if err := m.connectLocked(m.command); err != nil {
		return err
	}
	if m.initName != "" {
		if err := m.initializeLocked(); err != nil {
			return fmt.Errorf("agent: re-initialize after reconnect: %w", err)
		}
	}
	return nil
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
	if err := m.client.SendNotification("notifications/initialized", struct{}{}); err != nil {
		return fmt.Errorf("agent: initialized notification: %w", err)
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

func (m *Manager) convertUpdate(u *acp.SessionUpdate) *UpdateEvent {
	e := &UpdateEvent{Type: u.SessionUpdate}

	switch u.SessionUpdate {
	case "user_message_chunk", "agent_message_chunk":
		e.Content = &ContentChunk{
			Type: u.Content.Type,
		}
		if u.MessageID != nil {
			e.Content.MessageID = string(*u.MessageID)
		}
		if u.Content.Type == "text" {
			e.Content.Content = u.Content.Text
		} else if u.Content.Type == "image" || u.Content.Type == "audio" {
			e.Content.Data = u.Content.Data
			e.Content.Mime = u.Content.MimeType
		}
	case "agent_thought_chunk":
		e.Content = &ContentChunk{Thought: u.Thought}
	case "tool_call":
		if u.ToolCall != nil {
			e.Tool = u.ToolCall
		}
	case "tool_call_update":
		if u.ToolCallUpdate != nil {
			e.Tool = &acp.ToolCall{
				ToolCallID: u.ToolCallUpdate.ToolCallID,
				Status:     *u.ToolCallUpdate.Status,
			}
		}
	case "plan":
		e.Plan = u.Plan
	case "usage_update":
		e.Usage = u.Usage
	case "current_mode_update":
		e.Mode = u.CurrentMode
	case "session_info_update":
		e.Info = u.SessionInfo
	}
	return e
}

func (m *Manager) LoadSession(sessionID, cwd string) error {
	return m.runWithRetry(func(client *acp.Client) error {
		log.Printf("[agent] LoadSession: id=%q cwd=%q", sessionID, cwd)
		_, err := client.LoadSession(&acp.LoadSessionRequest{
			SessionID:  acp.SessionID(sessionID),
			Cwd:        cwd,
			McpServers: []acp.McpServer{},
		})
		if err != nil {
			return fmt.Errorf("agent: load session: %w", err)
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
		log.Printf("[agent] NewSession created: id=%q", sessionID)
		session = &Session{ID: sessionID, Cwd: cwd}
		return nil
	})
	return session, err
}

func (m *Manager) Prompt(sessionID, message string) error {
	return m.runWithRetry(func(client *acp.Client) error {
		log.Printf("[agent] Prompt: session=%q message=%q", sessionID, message)
		_, err := client.Prompt(&acp.PromptRequest{
			SessionID: acp.SessionID(sessionID),
			Prompt: []acp.ContentBlock{
				{Type: "text", Text: message},
			},
		})
		if err != nil {
			return fmt.Errorf("agent: prompt: %w", err)
		}
		return nil
	})
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
			if s.Title != nil {
				session.Title = *s.Title
			}
			if s.UpdatedAt != nil {
				session.UpdatedAt = *s.UpdatedAt
			}
			sessions = append(sessions, session)
		}
		return nil
	})
	return sessions, err
}

func (m *Manager) CloseSession(sessionID string) error {
	return m.runWithRetry(func(client *acp.Client) error {
		_, err := client.CloseSession(&acp.CloseSessionRequest{
			SessionID: acp.SessionID(sessionID),
		})
		return err
	})
}

func (m *Manager) DeleteSession(sessionID string) error {
	return m.runWithRetry(func(client *acp.Client) error {
		_, err := client.DeleteSession(&acp.DeleteSessionRequest{
			SessionID: acp.SessionID(sessionID),
		})
		return err
	})
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
