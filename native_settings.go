package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lsongdev/miya-agents/mcp"
	channelconfig "github.com/lsongdev/miya-channels/config"
	channelservice "miya-desktop/internal/channels"
	miyaconfig "miya-desktop/internal/config"
)

func (s *nativeUI) settingsView(c *ui.Context) {
	ui.Scroll(c.Key("settings:"+s.page)).Grow(1).Padding(26, 30).Gap(18).Children(func() {
		if s.config == nil {
			ui.Text(c, "Configuration is unavailable.")
			ui.Button(c, "Retry").OnClick(s.load)
			return
		}
		if s.registry != "" {
			s.registryView(c)
			return
		}
		switch s.page {
		case "General":
			heading(c, "General", "Application preferences and appearance.")
			labeledSelect(c, "Theme", &s.prefs.Theme, []string{"light", "dark"})
			ui.Checkbox(c, &s.prefs.Notifications, "Desktop notifications")
			ui.Text(c, "Notify when an agent finishes responding while Miya is in the background.").FontSize(12).TextColor(c.Theme().TextMuted)
			ui.Button(c, "Save preferences").Disabled(s.busy).OnClick(s.savePreferences)
			level := s.config.Logging.Level
			if level == "" {
				level = "info"
			}
			ui.Text(c, "Log level").Bold().FontSize(13)
			if ui.Select(c, &level, []string{"debug", "info", "warn", "error"}).Label("Log level").Disabled(s.busy || s.sending).Changed() {
				s.changeConfig(func(cfg *miyaconfig.Config) error { cfg.Logging.Level = level; cfg.Logging.Enabled = true; return nil })
			}
			ui.Text(c, "Logs are written to ~/.miya/logs.").TextColor(c.Theme().TextMuted).FontSize(12)
		case "Agents":
			s.agentsView(c)
		case "Providers":
			s.providersView(c)
		case "Profiles":
			s.profilesView(c)
		case "MCP Servers":
			s.mcpView(c)
		case "Skills":
			s.skillsView(c)
		case "Channels":
			s.channelsView(c)
		case "About":
			s.aboutView(c)
		}
	})
}

func (s *nativeUI) sectionHeader(c *ui.Context, title, detail, kind string) {
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(5).Children(func() { heading(c, title, detail) })
		if kind != "" {
			ui.PrimaryButton(c, "Add "+kind).Disabled(s.busy || s.sending).OnClick(func() { s.edit(kind, "") })
		}
	})
}

func settingsRow(c *ui.Context, key, title, detail string, actions func()) {
	t := c.Theme()
	ui.Row(c.Key(key)).Padding(12, 0).Gap(12).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, title).Bold().FontSize(14)
			if detail != "" {
				ui.Text(c, detail).FontSize(12).TextColor(t.TextMuted)
			}
		})
		actions()
	})
}

func (s *nativeUI) editDelete(c *ui.Context, kind, id string, remove func()) {
	ui.Button(c, "Edit").Label("Edit " + kind + " " + id).Disabled(s.busy || s.sending).OnClick(func() { s.edit(kind, id) })
	ui.Button(c, "Delete").Label("Delete " + kind + " " + id).Disabled(s.busy || s.sending).OnClick(func() {
		s.ask("Delete "+kind+"?", id+" will be removed from this workspace.", remove)
	})
}

func (s *nativeUI) agentsView(c *ui.Context) {
	s.sectionHeader(c, "External Agents", "Connect stdio ACP agents such as OpenCode, Codex or Claude.", "Agent")
	for _, endpoint := range s.config.Agents {
		endpoint := endpoint
		title := endpoint.Name
		if title == "" {
			title = endpoint.ID
		}
		settingsRow(c, endpoint.ID, title, endpoint.Command+" "+strings.Join(endpoint.Args, " "), func() {
			s.editDelete(c, "Agent", endpoint.ID, func() {
				s.changeConfig(func(cfg *miyaconfig.Config) error {
					next := cfg.Agents[:0]
					for _, agent := range cfg.Agents {
						if agent.ID != endpoint.ID {
							next = append(next, agent)
						}
					}
					cfg.Agents = next
					return nil
				})
			})
			enabled := endpoint.IsEnabled()
			if ui.Switch(c, &enabled).Label("Enable " + title).Disabled(s.busy || s.sending).Changed() {
				s.changeConfig(func(cfg *miyaconfig.Config) error {
					for i := range cfg.Agents {
						if cfg.Agents[i].ID == endpoint.ID {
							cfg.Agents[i].Enabled = &enabled
						}
					}
					return nil
				})
			}
		})
	}
	if len(s.config.Agents) == 0 {
		ui.Text(c, "No external agents configured. Built-in profiles are listed in Chat.").TextColor(c.Theme().TextMuted)
	}
}

func (s *nativeUI) providersView(c *ui.Context) {
	s.sectionHeader(c, "Providers", "Manage model endpoints and credentials.", "Provider")
	for _, id := range sortedKeys(s.config.Providers) {
		id := id
		provider := s.config.Providers[id]
		if provider == nil {
			continue
		}
		keyStatus := "No API key"
		if provider.APIKey != "" {
			keyStatus = "Key configured"
		}
		settingsRow(c, id, id, provider.Type+" · "+provider.APIBase+" · "+keyStatus, func() {
			s.editDelete(c, "Provider", id, func() {
				s.changeConfig(func(cfg *miyaconfig.Config) error {
					for _, profile := range cfg.Profiles {
						if profile != nil && profile.Provider == id {
							return fmt.Errorf("provider is still used by an agent profile")
						}
					}
					delete(cfg.Providers, id)
					return nil
				})
			})
		})
	}
	if len(s.config.Providers) == 0 {
		ui.Text(c, "No providers configured.").TextColor(c.Theme().TextMuted)
	}
}

func (s *nativeUI) profilesView(c *ui.Context) {
	s.sectionHeader(c, "Profiles", "Built-in agents appear automatically in Chat.", "Profile")
	for _, id := range sortedKeys(s.config.Profiles) {
		id := id
		profile := s.config.Profiles[id]
		if profile == nil {
			continue
		}
		settingsRow(c, id, id, profile.Provider+" / "+profile.ModelName, func() {
			s.editDelete(c, "Profile", id, func() { s.changeConfig(func(cfg *miyaconfig.Config) error { delete(cfg.Profiles, id); return nil }) })
		})
	}
	if len(s.config.Profiles) == 0 {
		ui.Text(c, "No profiles configured. Add a provider, then create a profile.").TextColor(c.Theme().TextMuted)
	}
}

func (s *nativeUI) mcpView(c *ui.Context) {
	s.sectionHeader(c, "MCP Servers", "Tool connections for your agents.", "MCP Server")
	ui.Button(c, "Browse MCP Marketplace").Disabled(s.busy).OnClick(func() { s.registry = "MCP"; s.searchRegistry() })
	for _, id := range sortedKeys(s.config.McpServers) {
		id := id
		server := s.config.McpServers[id]
		if server == nil {
			continue
		}
		detail := server.URL
		if server.Type == "" || server.Type == "stdio" {
			detail = server.Command + " " + strings.Join(server.Args, " ")
		}
		settingsRow(c, id, id, server.Type+" · "+detail, func() {
			s.editDelete(c, "MCP Server", id, func() { s.changeConfig(func(cfg *miyaconfig.Config) error { delete(cfg.McpServers, id); return nil }) })
		})
	}
	if len(s.config.McpServers) == 0 {
		ui.Text(c, "No MCP servers configured.").TextColor(c.Theme().TextMuted)
	}
}

func (s *nativeUI) fetchEditorModels() {
	id := s.editor.Provider
	s.task("Fetching models…", func() (func(), error) {
		models, err := s.backend.FetchProviderModels(id)
		return func() {
			if s.editor.Open && s.editor.Provider == id {
				s.models = models
			}
		}, err
	})
}

func (s *nativeUI) chooseWorkspace() {
	window := s.win
	s.task("Choose workspace…", func() (func(), error) {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: window, Directory: true})
		return func() {
			if len(paths) > 0 {
				s.editor.Workspace = paths[0]
			}
		}, err
	})
}

func (s *nativeUI) loadSkills() {
	s.task("Loading skills…", func() (func(), error) {
		skills, err := s.backend.ListSkills()
		return func() { s.skills = skills }, err
	})
}

func (s *nativeUI) skillsView(c *ui.Context) {
	s.sectionHeader(c, "Skills", "Reusable instructions for the embedded agents.", "Skill")
	ui.Row(c).Gap(8).Children(func() {
		ui.Button(c, "Browse Skills Registry").Disabled(s.busy).OnClick(func() { s.registry = "Skills"; s.searchRegistry() })
		ui.Button(c, "Refresh skills").Disabled(s.busy).OnClick(s.loadSkills)
		ui.Button(c, "Open skills folder").OnClick(func() {
			path := s.backend.SkillsDirectory()
			s.task("Opening folder…", func() (func(), error) { return nil, mygo.Shell.OpenPath(path) })
		})
	})
	for _, skill := range s.skills {
		skill := skill
		settingsRow(c, skill.Name, skill.Name, skill.Description, func() {
			s.editDelete(c, "Skill", skill.Name, func() {
				s.task("Deleting skill…", func() (func(), error) {
					if err := s.backend.DeleteSkill(skill.Name); err != nil {
						return nil, err
					}
					return func() { s.loadSkills() }, nil
				})
			})
		})
	}
	if len(s.skills) == 0 {
		ui.Text(c, "No skills installed.").TextColor(c.Theme().TextMuted)
	}
}

func (s *nativeUI) registryView(c *ui.Context) {
	heading(c, s.registry+" Registry", "Search and install integrations.")
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c, &s.registryQuery).Label("Search registry").Placeholder("Search").Grow(1).OnSubmit(s.searchRegistry)
		ui.Button(c, "Search").Disabled(s.busy).OnClick(s.searchRegistry)
		ui.Button(c, "Back").OnClick(func() { s.registry = "" })
	})
	if s.registry == "MCP" {
		for _, server := range s.mcpResults {
			server := server
			settingsRow(c, server.ID, server.Name, server.Description, func() {
				ui.Button(c, "Install").Label("Install " + server.Name).Disabled(s.busy || s.sending).OnClick(func() { s.installMCP(server) })
			})
			if len(server.RequiredInputs) > 0 {
				ui.Text(c, "Requires: "+strings.Join(server.RequiredInputs, ", ")).FontSize(11).TextColor(c.Theme().Warning)
			}
		}
		if len(s.mcpResults) == 0 && !s.busy {
			ui.Text(c, "No installable servers found.")
		}
	} else {
		for _, skill := range s.skillResults {
			skill := skill
			settingsRow(c, skill.ID, skill.Name, skill.Description, func() {
				label := "Install"
				if skill.Installed {
					label = "Reinstall"
				}
				ui.Button(c, label).Label(label + " " + skill.Name).Disabled(s.busy).OnClick(func() {
					s.task("Installing skill…", func() (func(), error) {
						_, err := s.backend.InstallRegistrySkill(skill.ID)
						return func() { s.registry = ""; s.loadSkills() }, err
					})
				})
			})
		}
		if len(s.skillResults) == 0 && !s.busy {
			ui.Text(c, "No skills found.")
		}
	}
}

func (s *nativeUI) searchRegistry() {
	query, kind := s.registryQuery, s.registry
	s.task("Searching registry…", func() (func(), error) {
		if kind == "MCP" {
			results, err := s.backend.ListMCPRegistry(query)
			return func() { s.mcpResults = results }, err
		}
		results, err := s.backend.ListSkillRegistry(query)
		return func() { s.skillResults = results }, err
	})
}

func (s *nativeUI) installMCP(server MCPRegistryServerInfo) {
	if server.Config == nil {
		return
	}
	cfg, err := cloneConfig(s.config)
	if err != nil {
		s.err = err.Error()
		return
	}
	id := safeSkillName(server.ID)
	if id == "" {
		id = "server"
	}
	base := id
	for n := 2; cfg.McpServers[id] != nil; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	if cfg.McpServers == nil {
		cfg.McpServers = map[string]*mcp.McpServerConfig{}
	}
	cfg.McpServers[id] = server.Config
	s.saveConfig(cfg, func() {
		s.registry = ""
		if len(server.RequiredInputs) > 0 {
			s.edit("MCP Server", id)
			s.err = "Complete the required values: " + strings.Join(server.RequiredInputs, ", ")
		}
	})
}

func (s *nativeUI) channelsView(c *ui.Context) {
	s.sectionHeader(c, "Channels", "Bridge remote chat tools into your agents.", "Channel")
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		running := "Stopped"
		if s.channelStatus.Running {
			running = "Running"
		}
		ui.Text(c, "Gateway: "+running).Bold().Grow(1)
		label := "Start gateway"
		if s.channelStatus.Running {
			label = "Stop gateway"
		}
		ui.Button(c, label).Disabled(s.busy).OnClick(s.toggleChannels)
	})
	if s.channelStatus.Error != "" {
		ui.Text(c, s.channelStatus.Error).TextColor(c.Theme().Danger)
	}
	enabled := channelsEnabled(s.config)
	if ui.Checkbox(c, &enabled, "Start channels automatically").Disabled(s.busy || s.sending).Changed() {
		s.changeConfig(func(cfg *miyaconfig.Config) error { cfg.ChannelsEnabled = &enabled; return nil })
	}
	for _, raw := range s.config.Channels {
		var channel channelconfig.ChannelInstance
		if json.Unmarshal(raw, &channel) != nil {
			continue
		}
		detail := channel.Type
		if event, ok := s.channelEvents[channel.ID]; ok {
			detail += " · " + event.Status
		}
		settingsRow(c, channel.ID, channel.ID, detail, func() {
			s.editDelete(c, "Channel", channel.ID, func() {
				restart := s.channelStatus.Running
				cfg, err := cloneConfig(s.config)
				if err != nil {
					s.err = err.Error()
					return
				}
				var next []json.RawMessage
				for _, raw := range cfg.Channels {
					var item channelconfig.ChannelInstance
					if json.Unmarshal(raw, &item) == nil && item.ID != channel.ID {
						next = append(next, raw)
					}
				}
				cfg.Channels = next
				s.saveConfig(cfg, func() {
					if restart {
						s.restartChannels()
					}
				})
			})
		})
	}
	if len(s.config.Channels) == 0 {
		ui.Text(c, "No channels configured.").TextColor(c.Theme().TextMuted)
	}
}

func (s *nativeUI) toggleChannels() {
	running := s.channelStatus.Running
	s.task("Updating gateway…", func() (func(), error) {
		var status channelservice.Status
		var err error
		if running {
			status, err = s.backend.StopChannelsService()
		} else {
			status, err = s.backend.StartChannelsService()
		}
		return func() { s.channelStatus = status }, err
	})
}
func (s *nativeUI) restartChannels() {
	s.task("Restarting gateway…", func() (func(), error) {
		if _, err := s.backend.StopChannelsService(); err != nil {
			return nil, err
		}
		status, err := s.backend.StartChannelsService()
		return func() { s.channelStatus = status }, err
	})
}

func (s *nativeUI) wechatView(c *ui.Context) {
	event := s.channelEvents[s.editor.ID]
	if event.Channel == "" {
		event = s.channelEvents["wechat"]
	}
	if s.qr != nil {
		ui.Image(c, s.qr).Size(200, 200).Fit(ui.Contain)
	}
	if event.Status != "" {
		ui.Text(c, "Login: "+event.Status).FontSize(12)
	}
	if event.Error != "" {
		ui.Text(c, event.Error).TextColor(c.Theme().Danger)
	}
	ui.Button(c, "Sign in with QR code").Disabled(s.busy).OnClick(func() {
		values := map[string]any{}
		if s.editor.BaseURL != "" {
			values["base_url"] = s.editor.BaseURL
		}
		if s.editor.CDNBaseURL != "" {
			values["cdn_base_url"] = s.editor.CDNBaseURL
		}
		s.qr = nil
		s.task("Starting WeChat login…", func() (func(), error) { return nil, s.backend.StartWeChatLogin(values) })
	})
	if event.QRCodeURL != "" {
		ui.Button(c, "Open QR code").OnClick(func() { s.openAttachment(event.QRCodeURL) })
	}
}

func (s *nativeUI) receiveChannel(event channelservice.ChannelEvent) {
	s.channelEvents[event.Channel] = event
	if !s.editor.Open || s.editor.Kind != "Channel" || s.editor.Type != "wechat" {
		return
	}
	if event.Status == "authenticated" {
		s.editor.Token, s.editor.BaseURL, s.editor.CDNBaseURL = event.Token, event.BaseURL, event.CDNBaseURL
		s.qr = nil
	}
	if event.QRCodeImage != "" {
		data, err := decodeAttachment(event.QRCodeImage)
		if err == nil {
			bitmap, err := ui.DecodeBitmap(data)
			if err == nil {
				s.qr = bitmap
			}
		}
	}
}

func (s *nativeUI) aboutView(c *ui.Context) {
	heading(c, "About", "Miya Desktop")
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		if s.logo != nil {
			ui.Image(c, s.logo).Size(64, 64).Radius(12)
		}
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, "Miya Desktop").FontSize(24).Bold()
			ui.Text(c, "Version "+appVersion).TextColor(c.Theme().TextMuted)
			ui.Text(c, "Native Go interface powered by MyGo").FontSize(12).TextColor(c.Theme().TextMuted)
		})
	})
	ui.Text(c, "An ACP-native workspace for agent conversations, providers, MCP tools, skills and remote channels.").MaxWidth(650)
	ui.Button(c, "Check for Updates").Disabled(s.busy).OnClick(func() {
		s.task("Checking for updates…", func() (func(), error) {
			result, err := s.backend.CheckForUpdates(s.backend.ctx)
			return func() {
				if result != nil {
					s.notice = result.Message
				}
			}, err
		})
	})
	if s.notice != "" && !s.busy {
		ui.Text(c, s.notice).TextColor(c.Theme().TextMuted)
	}
	for _, repo := range []string{"miya-desktop", "miya-agents", "miya-channels"} {
		repo := repo
		ui.Link(c, repo, "https://github.com/lsongdev/"+repo)
	}
	ui.Text(c, "MIT License · Copyright © Lsong").FontSize(12).TextColor(c.Theme().TextMuted)
}

func validHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https")
}
