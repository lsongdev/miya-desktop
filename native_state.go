package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"miya-desktop/internal/agent"
	channelservice "miya-desktop/internal/channels"
	miyaconfig "miya-desktop/internal/config"
	"miya-desktop/internal/conversation"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// nativeUI owns presentation state on the window thread. Protocol reduction
// remains in conversation.Store. Jobs return a commit for Window.Update.
type nativeUI struct {
	backend                                                *App
	win                                                    *mygo.Window
	dispatch                                               func(func())
	launch                                                 func(func())
	notify                                                 func(DesktopNotification)
	config                                                 *miyaconfig.Config
	endpoints                                              []miyaconfig.ACPAgentConfig
	selectedAgent, connectedAgent                          string
	agentInfo                                              *agent.AgentInfo
	sessions                                               []agent.Session
	selected                                               string
	conversations                                          map[string]conversation.Snapshot
	drafts                                                 map[string]string
	draft, search                                          string
	page                                                   string
	loading, busy, sending, cancelling                     bool
	active                                                 bool
	err, notice                                            string
	prefs                                                  desktopPreferences
	prefsPath                                              string
	timeline                                               ui.ListState
	md                                                     markdown.Options
	logo                                                   *ui.Bitmap
	images                                                 map[string]*ui.Bitmap
	imageErrors                                            map[string]string
	expanded                                               map[string]bool
	editor                                                 configEditor
	confirmOpen                                            bool
	confirmTitle, confirmDetail                            string
	confirmAction                                          func()
	skills                                                 []SkillInfo
	skillResults                                           []RegistrySkillInfo
	mcpResults                                             []MCPRegistryServerInfo
	registry, registryQuery                                string
	models                                                 []string
	channelStatus                                          channelservice.Status
	channelEvents                                          map[string]channelservice.ChannelEvent
	qr                                                     *ui.Bitmap
	audio                                                  *audio.Player
	audioKey                                               string
	nextPoll                                               time.Time
	polling                                                bool
	onboarding                                             bool
	welcomeProvider, welcomeBase, welcomeKey, welcomeModel string
}

type desktopPreferences struct {
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
	SelectedAgent string `json:"selectedAgent,omitempty"`
}

func newNativeUI() *nativeUI {
	s := &nativeUI{
		page: "Chat", active: true, loading: true,
		conversations: map[string]conversation.Snapshot{}, drafts: map[string]string{},
		images: map[string]*ui.Bitmap{}, imageErrors: map[string]string{},
		expanded:        map[string]bool{},
		channelEvents:   map[string]channelservice.ChannelEvent{},
		prefs:           desktopPreferences{Theme: "light", Notifications: true},
		welcomeProvider: "openai", welcomeBase: "https://api.openai.com/v1",
	}
	s.launch = func(job func()) { go job() }
	return s
}

func (s *nativeUI) post(fn func()) {
	if s.dispatch != nil {
		s.dispatch(fn)
	}
}

func (s *nativeUI) task(label string, job func() (func(), error)) {
	if s.busy {
		return
	}
	s.busy, s.err, s.notice = true, "", label
	s.launch(func() {
		commit, err := job()
		s.post(func() {
			s.busy, s.notice = false, ""
			if err != nil {
				s.err = err.Error()
				return
			}
			if commit != nil {
				commit()
			}
		})
	})
}

func (s *nativeUI) load() {
	b := s.backend
	s.task("Loading workspace…", func() (func(), error) {
		cfg, err := b.LoadMiyaConfig()
		if err != nil {
			return nil, err
		}
		prefsPath := filepath.Join(filepath.Dir(b.MiyaConfigPath()), "desktop.json")
		prefs := desktopPreferences{Theme: "light", Notifications: true}
		if data, err := os.ReadFile(prefsPath); err == nil {
			if err := json.Unmarshal(data, &prefs); err != nil {
				return nil, fmt.Errorf("read desktop preferences: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read desktop preferences: %w", err)
		}
		exists := b.MiyaConfigExists()
		status := b.ChannelsServiceStatus()
		return func() {
			s.loading, s.onboarding = false, !exists
			s.config, s.prefs, s.prefsPath = cfg, prefs, prefsPath
			s.selectedAgent = prefs.SelectedAgent
			s.channelStatus = status
			s.setEndpoints()
			s.refreshSessions()
		}, nil
	})
}

func (s *nativeUI) setEndpoints() {
	if s.config == nil {
		return
	}
	all, err := miyaconfig.AgentEndpoints(s.config)
	if err != nil {
		s.err = err.Error()
		return
	}
	s.endpoints = nil
	for _, endpoint := range all {
		if endpoint.IsEnabled() {
			s.endpoints = append(s.endpoints, endpoint)
		}
	}
	for _, endpoint := range s.endpoints {
		if endpoint.ID == s.selectedAgent {
			return
		}
	}
	s.selectedAgent = ""
	if len(s.endpoints) > 0 {
		s.selectedAgent = s.endpoints[0].ID
	}
}

func (s *nativeUI) refreshSessions() {
	if s.polling || s.backend == nil {
		return
	}
	s.polling = true
	b := s.backend
	s.launch(func() {
		sessions, err := b.ListAgentSessions()
		status := b.ChannelsServiceStatus()
		s.post(func() {
			s.polling = false
			if err != nil {
				s.err = err.Error()
				return
			}
			sort.SliceStable(sessions, func(i, j int) bool { return sessions[i].UpdatedAt > sessions[j].UpdatedAt })
			s.sessions, s.channelStatus = sessions, status
			for id, event := range status.ChannelEvents {
				s.channelEvents[id] = event
			}
		})
	})
}

func sessionKey(session agent.Session) string {
	if session.Key != "" {
		return session.Key
	}
	if session.AgentID != "" {
		return session.AgentID + ":" + session.ID
	}
	return session.ID
}

func (s *nativeUI) connect(agentID string) (*agent.AgentInfo, error) {
	if err := s.backend.DisconnectAgent(); err != nil {
		return nil, err
	}
	if err := s.backend.ConnectConfiguredAgent(agentID); err != nil {
		return nil, err
	}
	return s.backend.InitializeAgent("miya-desktop", appVersion)
}

func (s *nativeUI) newSession() {
	if s.sending || s.busy || s.selectedAgent == "" {
		return
	}
	agentID := s.selectedAgent
	s.connectedAgent = ""
	s.task("Creating session…", func() (func(), error) {
		info, err := s.connect(agentID)
		if err != nil {
			return nil, err
		}
		session, err := s.backend.CreateSession("")
		if err != nil {
			return nil, err
		}
		return func() {
			s.agentInfo, s.connectedAgent = info, agentID
			s.selectConversation(*session)
			s.sessions = append([]agent.Session{*session}, s.sessions...)
			s.page = "Chat"
		}, nil
	})
}

func (s *nativeUI) selectConversation(session agent.Session) {
	if s.audio != nil {
		s.audio.Close()
		s.audio, s.audioKey = nil, ""
	}
	s.drafts[s.selected] = s.draft
	s.selected = sessionKey(session)
	s.draft = s.drafts[s.selected]
	s.selectedAgent = session.AgentID
	s.timeline = ui.ListState{}
}

func (s *nativeUI) openSession(session agent.Session) {
	if s.sending || s.busy {
		return
	}
	key := sessionKey(session)
	if s.selected == key {
		return
	}
	s.connectedAgent = ""
	s.task("Loading conversation…", func() (func(), error) {
		info, err := s.connect(session.AgentID)
		if err != nil {
			return nil, err
		}
		if info.LoadSession {
			if err := s.backend.LoadSession(key, session.Cwd); err != nil {
				return nil, err
			}
		}
		conv, err := s.backend.GetConversation(key)
		if err != nil {
			return nil, err
		}
		if conv == nil && !info.LoadSession {
			return nil, fmt.Errorf("this agent cannot resume saved sessions")
		}
		return func() {
			s.agentInfo, s.connectedAgent = info, session.AgentID
			s.selectConversation(session)
			if conv != nil {
				s.conversations[key] = conversation.Snapshot{Conversation: *conv}
			}
		}, nil
	})
}

func (s *nativeUI) send(text string) {
	text = strings.TrimSpace(text)
	if text == "" || s.sending || s.busy || s.selected == "" {
		return
	}
	key := s.selected
	agentID, _, _ := strings.Cut(key, ":")
	needsConnect := s.connectedAgent != agentID
	cwd := s.conversations[key].Conversation.Cwd
	snapshot := s.conversations[key]
	snapshot.StopReason = ""
	s.conversations[key] = snapshot
	s.sending, s.err, s.draft = true, "", ""
	s.drafts[key] = ""
	b := s.backend
	s.launch(func() {
		var info *agent.AgentInfo
		var err error
		if needsConnect {
			info, err = s.connect(agentID)
			if err == nil && !info.LoadSession {
				err = fmt.Errorf("this agent cannot resume a disconnected session")
			}
			if err == nil {
				err = b.LoadSession(key, cwd)
			}
		}
		if err == nil {
			err = b.SendPrompt(key, text)
		}
		s.post(func() {
			s.sending, s.cancelling = false, false
			if info != nil && err == nil {
				s.agentInfo, s.connectedAgent = info, agentID
			}
			if err != nil {
				s.err = err.Error()
				s.drafts[key] = text
				if s.selected == key {
					s.draft = text
				}
			}
			s.refreshSessions()
		})
	})
}

func (s *nativeUI) cancel() {
	if !s.sending || s.cancelling {
		return
	}
	s.cancelling = true
	key, b := s.selected, s.backend
	s.launch(func() {
		err := b.CancelSession(key)
		s.post(func() {
			s.cancelling = false
			if err != nil {
				s.err = err.Error()
			}
		})
	})
}

func (s *nativeUI) emit(name string, data any) {
	s.post(func() {
		switch name {
		case "conversation:update":
			if snapshot, ok := data.(conversation.Snapshot); ok {
				s.receive(snapshot)
			}
		case "session:update":
			if update, ok := data.(agent.SessionUpdate); ok && update.Event != nil && update.Event.Info != nil {
				for i := range s.sessions {
					if sessionKey(s.sessions[i]) == update.SessionID && update.Event.Info.Title != nil {
						s.sessions[i].Title = *update.Event.Info.Title
					}
				}
			}
		case "channel:event":
			if event, ok := data.(channelservice.ChannelEvent); ok {
				s.receiveChannel(event)
			}
		}
	})
}

func (s *nativeUI) receive(snapshot conversation.Snapshot) {
	key := snapshot.Conversation.ID
	previous := s.conversations[key]
	// Preserve completion metadata when a later metadata-only snapshot arrives.
	if snapshot.StopReason == "" {
		snapshot.StopReason = previous.StopReason
	}
	s.conversations[key] = snapshot
	if s.active || !s.prefs.Notifications || s.notify == nil {
		return
	}
	status := map[string]string{}
	for _, message := range previous.Conversation.Messages {
		status[message.ID] = message.Status
	}
	for _, message := range snapshot.Conversation.Messages {
		if message.Role != conversation.RoleAssistant || status[message.ID] != conversation.MessageStreaming ||
			(message.Status != conversation.MessageComplete && message.Status != conversation.MessageFailed) {
			continue
		}
		var text strings.Builder
		for _, block := range message.Blocks {
			if block.Type == conversation.BlockText || block.Type == conversation.BlockMarkdown {
				text.WriteString(block.Content)
			}
		}
		body := []rune(strings.TrimSpace(text.String()))
		if len(body) > 180 {
			body = append(body[:177], []rune("…")...)
		}
		if len(body) == 0 {
			body = []rune("Agent response " + message.Status + ".")
		}
		title := snapshot.Conversation.Title
		if title == "" {
			title = "Miya Desktop"
		}
		s.notify(DesktopNotification{ID: key + ":" + message.ID, Title: title, Body: string(body), ConversationID: key})
	}
}

func (s *nativeUI) removeSession(session agent.Session, deleteSaved bool) {
	if s.sending {
		return
	}
	key := sessionKey(session)
	s.connectedAgent = ""
	s.task("Closing session…", func() (func(), error) {
		info, err := s.connect(session.AgentID)
		if err != nil {
			return nil, err
		}
		if deleteSaved {
			err = s.backend.DeleteSession(key)
		} else {
			err = s.backend.CloseSession(key)
		}
		if err != nil {
			return nil, err
		}
		return func() {
			s.agentInfo, s.connectedAgent = info, session.AgentID
			if s.selected == key {
				s.selected, s.draft = "", ""
			}
			delete(s.conversations, key)
			delete(s.drafts, key)
			s.refreshSessions()
		}, nil
	})
}

func cloneConfig(cfg *miyaconfig.Config) (*miyaconfig.Config, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var copy miyaconfig.Config
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	return &copy, nil
}

func (s *nativeUI) saveConfig(cfg *miyaconfig.Config, after func()) {
	s.task("Saving configuration…", func() (func(), error) {
		if err := s.backend.SaveMiyaConfig(cfg); err != nil {
			return nil, err
		}
		return func() {
			s.config = cfg
			s.connectedAgent = ""
			s.setEndpoints()
			s.models = nil
			if after != nil {
				after()
			}
		}, nil
	})
}

func (s *nativeUI) changeConfig(edit func(*miyaconfig.Config) error) {
	if s.sending || s.busy || s.config == nil {
		return
	}
	cfg, err := cloneConfig(s.config)
	if err == nil {
		err = edit(cfg)
	}
	if err != nil {
		s.err = err.Error()
		return
	}
	s.saveConfig(cfg, nil)
}

func (s *nativeUI) savePreferences() {
	prefs := s.prefs
	prefs.SelectedAgent = s.selectedAgent
	path := s.prefsPath
	if path == "" {
		return
	}
	s.task("Saving preferences…", func() (func(), error) {
		data, err := json.MarshalIndent(prefs, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
		return nil, os.WriteFile(path, append(data, '\n'), 0600)
	})
}

func (s *nativeUI) ask(title, detail string, action func()) {
	s.confirmTitle, s.confirmDetail, s.confirmAction, s.confirmOpen = title, detail, action, true
}
