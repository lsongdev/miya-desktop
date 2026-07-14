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
		var envelope struct {
			SessionID string          `json:"sessionId"`
			Update    json.RawMessage `json:"update"`
		}
		if err := json.Unmarshal(params, &envelope); err != nil {
			log.Printf("[agent] Failed to unmarshal session/update: %v", err)
			return
		}
		event := m.parseUpdate(envelope.Update)
		if event == nil {
			return
		}
		runtime.EventsEmit(m.ctx, "session:update", map[string]any{
			"sessionId": envelope.SessionID,
			"event":     event,
		})
	}
}

// parseUpdate decodes a session/update payload based on its `sessionUpdate`
// discriminator. Zed's ACP protocol uses a discriminated-union wire format
// where fields for tool_call / tool_call_update are inlined at the top level
// (not nested under `toolCall`), and agent_thought_chunk carries text inside
// a `content` ContentBlock rather than a bare `thought` string. The acp
// package's SessionUpdate struct doesn't match this layout, so we decode
// each variant directly here.
func (m *Manager) parseUpdate(raw json.RawMessage) *UpdateEvent {
	var disc struct {
		SessionUpdate string `json:"sessionUpdate"`
	}
	if err := json.Unmarshal(raw, &disc); err != nil {
		return nil
	}
	e := &UpdateEvent{Type: disc.SessionUpdate}

	switch disc.SessionUpdate {
	case "user_message_chunk", "agent_message_chunk":
		var u struct {
			Content   acp.ContentBlock `json:"content"`
			MessageID *string          `json:"messageId,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		e.Content = &ContentChunk{Type: u.Content.Type}
		if u.MessageID != nil {
			e.Content.MessageID = *u.MessageID
		}
		switch u.Content.Type {
		case "text":
			e.Content.Content = u.Content.Text
		case "image", "audio":
			e.Content.Data = u.Content.Data
			e.Content.Mime = u.Content.MimeType
		}
	case "agent_thought_chunk":
		// Two possible wire formats: a bare `thought` string (older) or a
		// `content` ContentBlock (Zed's spec). Handle both.
		var u struct {
			Thought string           `json:"thought,omitempty"`
			Content acp.ContentBlock `json:"content,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		text := u.Thought
		if text == "" {
			text = u.Content.Text
		}
		e.Content = &ContentChunk{Type: "text", Thought: text}
	case "tool_call":
		var u acp.ToolCall
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		if u.ToolCallID == "" {
			// Fall back to nested form in case some agent sends it that way.
			var wrap struct {
				ToolCall *acp.ToolCall `json:"toolCall"`
			}
			if json.Unmarshal(raw, &wrap) == nil && wrap.ToolCall != nil {
				u = *wrap.ToolCall
			}
		}
		tc := u
		e.Tool = &tc
	case "tool_call_update":
		var u acp.ToolCallUpdate
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		if u.ToolCallID == "" {
			var wrap struct {
				Update *acp.ToolCallUpdate `json:"toolCallUpdate"`
			}
			if json.Unmarshal(raw, &wrap) == nil && wrap.Update != nil {
				u = *wrap.Update
			}
		}
		tc := &acp.ToolCall{
			ToolCallID: u.ToolCallID,
			Content:    u.Content,
			Locations:  u.Locations,
		}
		if u.Title != nil {
			tc.Title = *u.Title
		}
		if u.Kind != nil {
			tc.Kind = *u.Kind
		}
		if u.Status != nil {
			tc.Status = *u.Status
		}
		e.Tool = tc
	case "plan":
		var u struct {
			Entries []acp.PlanEntry `json:"entries,omitempty"`
			Plan    *acp.Plan       `json:"plan,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		if u.Plan != nil {
			e.Plan = u.Plan
		} else if len(u.Entries) > 0 {
			e.Plan = &acp.Plan{Entries: u.Entries}
		}
	case "usage_update":
		var u struct {
			Usage *acp.UsageUpdate `json:"usage,omitempty"`
			Size  *uint64          `json:"size,omitempty"`
			Used  *uint64          `json:"used,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		if u.Usage != nil {
			e.Usage = u.Usage
		} else if u.Size != nil || u.Used != nil {
			e.Usage = &acp.UsageUpdate{}
			if u.Size != nil {
				e.Usage.Size = *u.Size
			}
			if u.Used != nil {
				e.Usage.Used = *u.Used
			}
		}
	case "current_mode_update":
		var u struct {
			CurrentMode   *acp.CurrentModeUpdate `json:"currentMode,omitempty"`
			CurrentModeID *acp.SessionModeID     `json:"currentModeId,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		if u.CurrentMode != nil {
			e.Mode = u.CurrentMode
		} else if u.CurrentModeID != nil {
			e.Mode = &acp.CurrentModeUpdate{CurrentModeID: *u.CurrentModeID}
		}
	case "session_info_update":
		var u struct {
			SessionInfo *acp.SessionInfoUpdate `json:"sessionInfo,omitempty"`
			Title       *string                `json:"title,omitempty"`
			UpdatedAt   *string                `json:"updatedAt,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil
		}
		if u.SessionInfo != nil {
			e.Info = u.SessionInfo
		} else if u.Title != nil || u.UpdatedAt != nil {
			e.Info = &acp.SessionInfoUpdate{Title: u.Title, UpdatedAt: u.UpdatedAt}
		}
	}
	return e
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
	log.Printf("[agent] DeleteSession: id=%q", sessionID)
	err := m.runWithRetry(func(client *acp.Client) error {
		_, err := client.DeleteSession(&acp.DeleteSessionRequest{
			SessionID: acp.SessionID(sessionID),
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
