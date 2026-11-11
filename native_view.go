package main

import (
	"strings"
	"time"

	"miya-desktop/internal/agent"
	miyaconfig "miya-desktop/internal/config"

	"github.com/egoist/mygo/ui"
)

var settingsPages = []string{"General", "Agents", "Profiles", "Providers", "MCP Servers", "Skills", "Channels", "About"}

func (s *nativeUI) view(c *ui.Context) {
	switch s.prefs.Theme {
	case "light":
		c.SetTheme(ui.LightTheme())
	case "dark":
		c.SetTheme(ui.DarkTheme())
	}
	t := c.Theme()
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		s.newSession()
	}
	if c.Shortcut(ui.Cmd, ui.KeyComma) {
		s.page = "General"
	}
	if c.Shortcut(ui.Cmd, ui.KeyB) {
		s.sidebarHidden = !s.sidebarHidden
	}
	// Where the window has a material, the sidebar shows it; elsewhere the
	// window gets a plain color of its own.
	root := ui.Row(c).Fill().AlignItems(ui.Stretch)
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	} else {
		root.Background(gutterColor(t))
	}
	root.Children(func() {
		if !s.sidebarHidden {
			ui.Column(c.Key("navigation")).Width(sidebarWidth).Shrink(0).Padding(12, 12, 12, 12).Gap(8).DragWindow().
				Children(func() { s.sidebar(c) })
		}
		island(ui.Column(c.Key("workspace")).Grow(1).MinWidth(0).Background(t.Background).
			Margin(islandGap, islandGap, islandGap, workspaceLeft(s.sidebarHidden)), t).Children(func() {
			if s.sidebarHidden {
				ui.Row(c.Key("workspace-bar")).Height(40).Shrink(0).AlignItems(ui.Center).Padding(0, 8, 0, max(c.TitleBar().Left-islandGap, 8)).DragWindow().Children(func() {
					if iconButton(c.Key("show-sidebar"), iconSidebar, 16, "Show sidebar").Clicked() {
						s.sidebarHidden = false
					}
				})
			}
			if s.err != "" {
				ui.Row(c.Key("error")).Padding(10, 18).Gap(8).Background(t.Danger.Alpha(0.08)).Children(func() {
					ui.Text(c, s.err).TextColor(t.Danger).Grow(1).Selectable()
					ui.Button(c, "Dismiss").OnClick(func() { s.err = "" })
				})
			}
			if s.busy {
				ui.Text(c, s.notice).FontSize(12).TextColor(t.TextMuted).Padding(8, 20)
			}
			switch {
			case s.loading:
				ui.Column(c.Key("loading")).Grow(1).Center().Children(func() {
					label := "Loading workspace…"
					if !s.busy {
						label = "Workspace unavailable"
					}
					ui.Text(c, label).TextColor(t.TextMuted)
					if !s.busy {
						ui.Button(c, "Retry").OnClick(s.load)
					}
				})
			case s.onboarding:
				s.welcomeView(c)
			case s.page == "Chat":
				s.chatView(c)
			default:
				s.settingsView(c)
			}
		})
	})
	s.editorView(c)
	ui.Modal(c.Key("confirm"), &s.confirmOpen, func() {
		ui.Text(c, s.confirmTitle).Bold().FontSize(18)
		ui.Text(c, s.confirmDetail).MaxWidth(460)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			ui.Button(c, "Cancel").OnClick(func() { s.confirmOpen = false })
			ui.PrimaryButton(c, "Confirm").Disabled(s.busy || s.sending).OnClick(func() {
				action := s.confirmAction
				s.confirmOpen, s.confirmAction = false, nil
				if action != nil {
					action()
				}
			})
		})
	})
	if !s.loading && s.backend != nil && c.Now().After(s.nextPoll) {
		s.nextPoll = c.Now().Add(8 * time.Second)
		s.refreshSessions()
	}
	c.After(8 * time.Second)
}

func (s *nativeUI) sidebar(c *ui.Context) {
	t := c.Theme()
	// Level with the window buttons, which sit over the top of the sidebar.
	ui.Row(c.Key("sidebar-bar")).Height(28).Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Spacer(c)
		if iconButton(c.Key("hide-sidebar"), iconSidebar, 16, "Hide sidebar").Clicked() {
			s.sidebarHidden = true
		}
	})
	if s.page != "Chat" {
		if navButton(c.Key("back"), iconBack, "Back to chats", false).Clicked() {
			s.page = "Chat"
		}
		ui.Text(c, "SETTINGS").FontSize(10).Bold().TextColor(t.TextMuted).Padding(10, 12, 2, 12)
		ui.Scroll(c.Key("settings-nav")).Grow(1).Gap(2).Children(func() {
			for _, page := range settingsPages {
				page := page
				if navButton(c.Key(page), nil, page, s.page == page).Clicked() {
					s.page, s.registry = page, ""
					if page == "Skills" {
						s.loadSkills()
					}
				}
			}
		})
		if s.backend != nil {
			ui.Text(c, s.backend.MiyaConfigPath()).FontSize(10).TextColor(t.TextMuted).Selectable()
		}
		return
	}
	options := make([]string, 0, len(s.endpoints))
	for _, endpoint := range s.endpoints {
		options = append(options, endpoint.ID)
	}
	if ui.Select(c.Key("agent"), &s.selectedAgent, options).Label("Agent").Disabled(s.sending || s.busy).Changed() {
		s.savePreferences()
	}
	ui.PrimaryButton(c, "New Session").Label("New Session").Height(36).Radius(10).Disabled(s.sending || s.busy || len(options) == 0).OnClick(s.newSession)
	ui.TextInput(c.Key("search"), &s.search).Placeholder("Search sessions").Label("Search sessions").Radius(10)
	ui.Scroll(c.Key("sessions")).Grow(1).Gap(2).Children(func() {
		visible := 0
		for _, session := range s.sessions {
			if !strings.Contains(strings.ToLower(session.Title+" "+session.ID+" "+session.AgentName), strings.ToLower(s.search)) {
				continue
			}
			visible++
			s.sessionRow(c, session)
		}
		if visible == 0 {
			ui.Text(c, "No sessions yet").FontSize(12).TextColor(t.TextMuted).Padding(12, 4)
		}
	})
	// Settings sits under the list of sessions.
	ui.Row(c.Key("sidebar-foot")).Shrink(0).AlignItems(ui.Center).Gap(4).Children(func() {
		if navButton(c.Key("settings"), iconSettings, "Settings", false).Grow(1).Clicked() {
			s.page = "General"
		}
		if iconButton(c.Key("refresh"), iconRefresh, 15, "Refresh").Disabled(s.polling).Clicked() {
			s.refreshSessions()
		}
	})
}

func (s *nativeUI) sessionRow(c *ui.Context, session agent.Session) {
	t := c.Theme()
	key := sessionKey(session)
	title := session.Title
	if title == "" {
		title = "Session " + shortID(session.ID)
	}
	selected := s.selected == key
	row := ui.Column(c.Key(key)).Padding(7, 10).Gap(2).Radius(10)
	bg := ui.Transparent
	switch {
	case selected:
		bg = t.Accent.Alpha(0.12)
	case row.Hovered():
		bg = t.Text.Alpha(0.06)
	}
	row.Background(row.AnimateColor("bg", bg, tintFade))
	// Close and Delete show for the chosen row and the one under the pointer.
	showActions := selected || row.Hovered()
	row.Children(func() {
		open := ui.ButtonBase(c.Key("open")).Label("Open " + title).Disabled(s.sending || s.busy).Justify(ui.Start).AlignItems(ui.Center)
		if open.Clicked() {
			s.openSession(session)
		}
		open.Children(func() {
			fg := t.Text
			if selected {
				fg = t.Accent
			}
			ui.Text(c, title).FontSize(13).TextColor(fg).SingleLine()
		})
		ui.Row(c.Key("meta")).AlignItems(ui.Center).Gap(2).Children(func() {
			ui.Text(c, session.AgentName).FontSize(10).TextColor(t.TextMuted).SingleLine().Grow(1)
			if !showActions {
				return
			}
			if iconButton(c.Key("close"), iconClose, 12, "Close "+title).Disabled(s.sending || s.busy).Clicked() {
				s.removeSession(session, false)
			}
			if iconButton(c.Key("delete"), iconTrash, 12, "Delete "+title).Disabled(s.sending || s.busy).Clicked() {
				s.ask("Delete conversation?", title+" will be permanently removed from the agent.", func() { s.removeSession(session, true) })
			}
		})
	})
}

func shortID(id string) string {
	r := []rune(id)
	if len(r) > 14 {
		return string(r[:14]) + "…"
	}
	return id
}

func heading(c *ui.Context, title, description string) {
	ui.Text(c, title).FontSize(23).Bold()
	if description != "" {
		ui.Text(c, description).TextColor(c.Theme().TextMuted).FontSize(13)
	}
}

func (s *nativeUI) welcomeView(c *ui.Context) {
	ui.Scroll(c.Key("welcome")).Grow(1).Padding(32).Gap(16).Children(func() {
		heading(c, "Welcome to Miya", "Connect your model provider and start your first conversation.")
		ui.Column(c).MaxWidth(560).Gap(12).Children(func() {
			field(c, "Provider name", &s.welcomeProvider, false)
			field(c, "API base URL", &s.welcomeBase, false)
			field(c, "API key", &s.welcomeKey, true)
			field(c, "Model", &s.welcomeModel, false)
			ui.Row(c).Gap(8).Children(func() {
				ui.Button(c, "Fetch Models").Disabled(s.busy).OnClick(s.welcomeModels)
				if len(s.models) > 0 {
					ui.Select(c, &s.welcomeModel, s.models).Grow(1).Label("Available models")
				}
			})
			ui.PrimaryButton(c, "Start using Miya").Disabled(s.busy || strings.TrimSpace(s.welcomeModel) == "").OnClick(s.initializeWorkspace)
			ui.Button(c, "Configure external agents instead").OnClick(func() { s.onboarding = false; s.page = "Agents" })
		})
	})
}

func (s *nativeUI) initializeWorkspace() {
	id, base, key, model := strings.TrimSpace(s.welcomeProvider), strings.TrimSpace(s.welcomeBase), strings.TrimSpace(s.welcomeKey), strings.TrimSpace(s.welcomeModel)
	if !validConfigID(id) || model == "" {
		s.err = "Enter a valid provider name and model."
		return
	}
	cfg, err := cloneConfig(s.config)
	if err != nil {
		s.err = err.Error()
		return
	}
	cfg.Providers[id] = &miyaconfig.ProviderConfig{Type: "openai", APIBase: base, APIKey: key}
	cfg.Profiles["default"] = &miyaconfig.ProfileConfig{Provider: id, ModelName: model, Workspace: "~/.miya/workspace", MaxTokens: 8192, ContextWindowTokens: 128000, CompactThreshold: 0.8}
	disabled := false
	cfg.ChannelsEnabled = &disabled
	s.task("Initializing workspace…", func() (func(), error) {
		if err := s.backend.InitializeDefaultWorkspace(); err != nil {
			return nil, err
		}
		if err := s.backend.SaveMiyaConfig(cfg); err != nil {
			return nil, err
		}
		return func() { s.config = cfg; s.onboarding = false; s.setEndpoints(); s.newSession() }, nil
	})
}

func (s *nativeUI) welcomeModels() {
	id := s.welcomeProvider
	provider := miyaconfig.ProviderConfig{Type: "openai", APIKey: s.welcomeKey, APIBase: s.welcomeBase}
	s.task("Fetching models…", func() (func(), error) {
		models, err := s.backend.FetchProviderModelsFromConfig(id, provider)
		return func() {
			s.models = models
			if s.welcomeModel == "" && len(models) > 0 {
				s.welcomeModel = models[0]
			}
		}, err
	})
}

func field(c *ui.Context, label string, value *string, secret bool) {
	ui.Column(c.Key(label)).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted)
		input := ui.TextInput(c, value).Label(label)
		if secret {
			input.Password()
		}
	})
}

func multiline(c *ui.Context, label string, value *string, height float32) {
	ui.Column(c.Key(label)).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.TextArea(c, value).Height(height).Label(label)
	})
}
