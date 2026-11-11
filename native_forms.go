package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/lsongdev/miya-agents/mcp"
	channelconfig "github.com/lsongdev/miya-channels/config"
	miyaconfig "miya-desktop/internal/config"
)

type configEditor struct {
	Open                                                                                bool
	Kind, Original                                                                      string
	ID, Name, Type, Command, Args, Env, URL, Headers                                    string
	Provider, Model, Workspace, MaxTokens, Temperature, ContextWindow, CompactThreshold string
	APIKey, APIBase, Token, BaseURL, CDNBaseURL, JSON, Description, Prompt              string
	Enabled                                                                             bool
}

func validConfigID(id string) bool {
	if id == "" {
		return false
	}
	for i, r := range id {
		if r > 127 || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && (i == 0 || (r != '-' && r != '_' && r != '.'))) {
			return false
		}
	}
	return true
}

func sortedKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func encodeArgs(args []string) string {
	if len(args) == 0 {
		return "[]"
	}
	return prettyJSONValue(args)
}
func parseArgs(text string) ([]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	var args []string
	if strings.HasPrefix(text, "[") {
		if err := json.Unmarshal([]byte(text), &args); err != nil {
			return nil, fmt.Errorf("arguments must be a JSON array of strings: %w", err)
		}
		return args, nil
	}
	for _, line := range strings.Split(text, "\n") {
		if line != "" {
			args = append(args, line)
		}
	}
	return args, nil
}

func parsePairs(text string) (map[string]string, error) {
	values := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("expected one KEY=value entry per line")
		}
		values[key] = value
	}
	return values, nil
}
func pairText(values map[string]string) string {
	var lines []string
	for _, key := range sortedKeys(values) {
		lines = append(lines, key+"="+values[key])
	}
	return strings.Join(lines, "\n")
}

func (s *nativeUI) edit(kind, id string) {
	s.err, s.models, s.qr = "", nil, nil
	e := configEditor{Open: true, Kind: kind, Original: id, ID: id, Enabled: true, Type: "stdio", JSON: "{}"}
	switch kind {
	case "Agent":
		for _, endpoint := range s.config.Agents {
			if endpoint.ID == id {
				e.Name, e.Type, e.Command, e.Args, e.Enabled = endpoint.Name, endpoint.Type, endpoint.Command, encodeArgs(endpoint.Args), endpoint.IsEnabled()
			}
		}
	case "Provider":
		e.Type = "openai"
		if provider := s.config.Providers[id]; provider != nil {
			e.Type, e.APIKey, e.APIBase = provider.Type, provider.APIKey, provider.APIBase
		}
	case "Profile":
		if ids := sortedKeys(s.config.Providers); len(ids) > 0 {
			e.Provider = ids[0]
		}
		if profile := s.config.Profiles[id]; profile != nil {
			e.Provider, e.Model, e.Workspace = profile.Provider, profile.ModelName, profile.Workspace
			e.MaxTokens, e.Temperature = strconv.Itoa(profile.MaxTokens), strconv.FormatFloat(profile.Temperature, 'f', -1, 64)
			e.ContextWindow, e.CompactThreshold = strconv.Itoa(profile.ContextWindowTokens), strconv.FormatFloat(profile.CompactThreshold, 'f', -1, 64)
		}
	case "MCP Server":
		if server := s.config.McpServers[id]; server != nil {
			e.Type, e.Command, e.Args, e.URL = server.Type, server.Command, encodeArgs(server.Args), server.URL
			e.Env, e.Headers = pairText(server.Env), pairText(server.Headers)
		}
	case "Channel":
		e.Type = "telegram"
		for _, raw := range s.config.Channels {
			var channel channelconfig.ChannelInstance
			if json.Unmarshal(raw, &channel) == nil && channel.ID == id {
				e.Type, e.Enabled = channel.Type, channel.IsEnabled()
				var values map[string]any
				if json.Unmarshal(channel.Config, &values) == nil {
					e.Token, _ = values["token"].(string)
					e.BaseURL, _ = values["base_url"].(string)
					e.CDNBaseURL, _ = values["cdn_base_url"].(string)
					e.JSON = prettyJSON(channel.Config)
				}
			}
		}
	case "Skill":
		for _, skill := range s.skills {
			if skill.Name == id {
				e.Description, e.Prompt = skill.Description, skill.Prompt
			}
		}
	}
	if e.Type == "" {
		e.Type = "stdio"
	}
	s.editor = e
}

func (s *nativeUI) editorView(c *ui.Context) {
	e := &s.editor
	ui.Modal(c.Key("editor"), &e.Open, func() {
		title := "Add " + e.Kind
		if e.Original != "" {
			title = "Edit " + e.Kind
		}
		ui.Text(c, title).Bold().FontSize(20)
		ui.Scroll(c.Key(e.Kind+e.Original)).Width(530).Height(410).Gap(12).Padding(0, 4).Children(func() {
			if e.Kind == "Skill" && e.Original != "" {
				ui.Text(c, "Skill ID").FontSize(12).TextColor(c.Theme().TextMuted)
				ui.TextInput(c, &e.ID).ReadOnly(true).Label("Skill ID")
			} else {
				field(c, e.Kind+" ID", &e.ID, false)
			}
			switch e.Kind {
			case "Agent":
				field(c, "Display name", &e.Name, false)
				field(c, "Command", &e.Command, false)
				multiline(c, "Arguments (JSON array or one per line)", &e.Args, 70)
				ui.Checkbox(c, &e.Enabled, "Enabled")
			case "Provider":
				labeledSelect(c, "Provider type", &e.Type, []string{"openai", "anthropic"})
				field(c, "API key", &e.APIKey, true)
				field(c, "API base URL", &e.APIBase, false)
			case "Profile":
				labeledSelect(c, "Provider", &e.Provider, sortedKeys(s.config.Providers))
				field(c, "Model", &e.Model, false)
				ui.Button(c, "Fetch Models").Disabled(s.busy || e.Provider == "").OnClick(s.fetchEditorModels)
				if len(s.models) > 0 {
					labeledSelect(c, "Available models", &e.Model, s.models)
				}
				field(c, "Workspace", &e.Workspace, false)
				ui.Button(c, "Choose workspace…").Disabled(s.busy).OnClick(s.chooseWorkspace)
				field(c, "Max output tokens", &e.MaxTokens, false)
				field(c, "Temperature", &e.Temperature, false)
				field(c, "Context window tokens", &e.ContextWindow, false)
				field(c, "Compact at (fraction of window)", &e.CompactThreshold, false)
			case "MCP Server":
				labeledSelect(c, "Transport", &e.Type, []string{"stdio", "streamablehttp", "sse"})
				if e.Type == "stdio" {
					field(c, "Command", &e.Command, false)
					multiline(c, "Arguments (JSON array or one per line)", &e.Args, 64)
					multiline(c, "Environment (KEY=value)", &e.Env, 80)
				} else {
					field(c, "Server URL", &e.URL, false)
					multiline(c, "Headers (Name=value)", &e.Headers, 100)
				}
			case "Channel":
				labeledSelect(c, "Channel type", &e.Type, []string{"telegram", "wechat", "feishu", "wecom"})
				ui.Checkbox(c, &e.Enabled, "Enabled")
				if e.Type == "telegram" || e.Type == "wechat" {
					field(c, "Token", &e.Token, true)
					field(c, "Base URL", &e.BaseURL, false)
					if e.Type == "wechat" {
						field(c, "CDN base URL", &e.CDNBaseURL, false)
						s.wechatView(c)
					}
				} else {
					multiline(c, "Channel configuration (JSON)", &e.JSON, 210)
				}
			case "Skill":
				field(c, "Description", &e.Description, false)
				multiline(c, "Instructions", &e.Prompt, 240)
			}
		})
		if s.err != "" {
			ui.Text(c, s.err).MaxWidth(530).TextColor(c.Theme().Danger).FontSize(12)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			ui.Button(c, "Cancel").Disabled(s.busy).OnClick(func() { s.editor = configEditor{}; s.err = "" })
			ui.PrimaryButton(c, "Save").Disabled(s.busy || s.sending || strings.TrimSpace(e.ID) == "").OnClick(s.saveEditor)
		})
	})
}

func labeledSelect(c *ui.Context, label string, value *string, options []string) {
	ui.Column(c.Key(label)).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Select(c, value, options).Label(label)
	})
}

func (s *nativeUI) saveEditor() {
	e := s.editor
	if e.Kind == "Skill" {
		if safeSkillName(e.ID) == "" || strings.TrimSpace(e.Prompt) == "" {
			s.err = "Enter a name and instructions."
			return
		}
		s.task("Saving skill…", func() (func(), error) {
			_, err := s.backend.InstallSkill(e.ID, e.Description, e.Prompt)
			if err != nil {
				return nil, err
			}
			return func() { s.editor = configEditor{}; s.loadSkills() }, nil
		})
		return
	}
	cfg, err := cloneConfig(s.config)
	if err == nil {
		err = applyEditor(cfg, e)
	}
	if err != nil {
		s.err = err.Error()
		return
	}
	restart := e.Kind == "Channel" && s.channelStatus.Running
	s.saveConfig(cfg, func() {
		s.editor = configEditor{}
		if restart {
			s.restartChannels()
		}
	})
}

// applyEditor validates a complete draft before changing the copy to save.
// Existing provider/profile options are retained when editing a subset.
func applyEditor(cfg *miyaconfig.Config, e configEditor) error {
	id := strings.TrimSpace(e.ID)
	if id == "" {
		return fmt.Errorf("a name is required")
	}
	if cfg.Providers == nil {
		cfg.Providers = map[string]*miyaconfig.ProviderConfig{}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*miyaconfig.ProfileConfig{}
	}
	if e.Kind != "Provider" && !validConfigID(id) {
		return fmt.Errorf("use letters, digits, dashes, underscores or dots in IDs")
	}
	switch e.Kind {
	case "Provider":
		if cfg.Providers[id] != nil && e.Original != id {
			return fmt.Errorf("provider %q already exists", id)
		}
		provider := miyaconfig.ProviderConfig{}
		if previous := cfg.Providers[e.Original]; previous != nil {
			provider = *previous
		}
		provider.Type, provider.APIKey, provider.APIBase = e.Type, strings.TrimSpace(e.APIKey), strings.TrimSpace(e.APIBase)
		if e.Original != "" && e.Original != id {
			delete(cfg.Providers, e.Original)
			for _, profile := range cfg.Profiles {
				if profile != nil && profile.Provider == e.Original {
					profile.Provider = id
				}
			}
		}
		cfg.Providers[id] = &provider
	case "Profile":
		if cfg.Profiles[id] != nil && e.Original != id {
			return fmt.Errorf("profile %q already exists", id)
		}
		for _, endpoint := range cfg.Agents {
			if endpoint.ID == id {
				return fmt.Errorf("an external agent already uses %q", id)
			}
		}
		if cfg.Providers[e.Provider] == nil {
			return fmt.Errorf("choose an existing provider")
		}
		profile := miyaconfig.ProfileConfig{}
		if previous := cfg.Profiles[e.Original]; previous != nil {
			profile = *previous
		}
		profile.Provider, profile.ModelName, profile.Workspace = e.Provider, strings.TrimSpace(e.Model), strings.TrimSpace(e.Workspace)
		var err error
		if profile.MaxTokens, err = parseInteger(e.MaxTokens, "max tokens"); err != nil {
			return err
		}
		if profile.ContextWindowTokens, err = parseInteger(e.ContextWindow, "context window"); err != nil {
			return err
		}
		if profile.Temperature, err = parseDecimal(e.Temperature, "temperature"); err != nil {
			return err
		}
		if profile.CompactThreshold, err = parseDecimal(e.CompactThreshold, "compaction threshold"); err != nil {
			return err
		}
		if profile.CompactThreshold > 1 {
			return fmt.Errorf("compaction threshold must be between 0 and 1")
		}
		if e.Original != "" && e.Original != id {
			delete(cfg.Profiles, e.Original)
		}
		cfg.Profiles[id] = &profile
	case "Agent":
		if cfg.Profiles[id] != nil {
			return fmt.Errorf("a profile already uses %q", id)
		}
		if strings.TrimSpace(e.Command) == "" {
			return fmt.Errorf("enter an executable command")
		}
		args, err := parseArgs(e.Args)
		if err != nil {
			return err
		}
		endpoint := miyaconfig.ACPAgentConfig{ID: id, Name: strings.TrimSpace(e.Name), Type: "stdio", Command: strings.TrimSpace(e.Command), Args: args}
		enabled := e.Enabled
		endpoint.Enabled = &enabled
		next := make([]miyaconfig.ACPAgentConfig, 0, len(cfg.Agents)+1)
		for _, old := range cfg.Agents {
			if old.ID == id && old.ID != e.Original {
				return fmt.Errorf("agent %q already exists", id)
			}
			if old.ID != e.Original {
				next = append(next, old)
			}
		}
		cfg.Agents = append(next, endpoint)
	case "MCP Server":
		if cfg.McpServers[id] != nil && e.Original != id {
			return fmt.Errorf("MCP server %q already exists", id)
		}
		server := &mcp.McpServerConfig{Type: e.Type}
		var err error
		if e.Type == "stdio" {
			if strings.TrimSpace(e.Command) == "" {
				return fmt.Errorf("enter an executable command")
			}
			server.Command = strings.TrimSpace(e.Command)
			if server.Args, err = parseArgs(e.Args); err != nil {
				return err
			}
			if server.Env, err = parsePairs(e.Env); err != nil {
				return err
			}
		} else {
			if !validHTTPURL(e.URL) {
				return fmt.Errorf("enter a valid HTTP(S) server URL")
			}
			server.URL = strings.TrimSpace(e.URL)
			if server.Headers, err = parsePairs(e.Headers); err != nil {
				return err
			}
		}
		if e.Original != "" && e.Original != id {
			delete(cfg.McpServers, e.Original)
		}
		if cfg.McpServers == nil {
			cfg.McpServers = map[string]*mcp.McpServerConfig{}
		}
		cfg.McpServers[id] = server
	case "Channel":
		var existing channelconfig.ChannelInstance
		var next []json.RawMessage
		for _, raw := range cfg.Channels {
			var channel channelconfig.ChannelInstance
			if err := json.Unmarshal(raw, &channel); err != nil {
				return fmt.Errorf("parse existing channel: %w", err)
			}
			if channel.ID == id && channel.ID != e.Original {
				return fmt.Errorf("channel %q already exists", id)
			}
			if channel.ID == e.Original {
				existing = channel
			} else {
				next = append(next, raw)
			}
		}
		values := map[string]any{}
		if len(existing.Config) > 0 {
			if err := json.Unmarshal(existing.Config, &values); err != nil {
				return err
			}
		}
		if e.Type == "telegram" || e.Type == "wechat" {
			values["token"] = strings.TrimSpace(e.Token)
			if e.BaseURL != "" {
				values["base_url"] = strings.TrimSpace(e.BaseURL)
			} else {
				delete(values, "base_url")
			}
			if e.Type == "wechat" {
				if e.CDNBaseURL != "" {
					values["cdn_base_url"] = strings.TrimSpace(e.CDNBaseURL)
				} else {
					delete(values, "cdn_base_url")
				}
			}
		} else {
			if err := json.Unmarshal([]byte(e.JSON), &values); err != nil {
				return fmt.Errorf("channel configuration: %w", err)
			}
		}
		enabled := e.Enabled
		existing.ID, existing.Type, existing.Enabled = id, e.Type, &enabled
		data, err := json.Marshal(values)
		if err != nil {
			return err
		}
		existing.Config = data
		raw, err := json.Marshal(existing)
		if err != nil {
			return err
		}
		cfg.Channels = append(next, raw)
	default:
		return fmt.Errorf("unknown configuration section")
	}
	return nil
}

func parseInteger(value, label string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 0 {
		return 0, fmt.Errorf("%s must be a nonnegative integer", label)
	}
	return number, nil
}
func parseDecimal(value, label string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("%s must be a nonnegative number", label)
	}
	return number, nil
}
