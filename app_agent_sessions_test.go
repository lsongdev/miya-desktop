package main

import (
	"path/filepath"
	"testing"

	agentsconfig "github.com/lsongdev/miya-agents/config"
	agentsession "github.com/lsongdev/miya-agents/session"
	miyaconfig "wails-app/internal/config"
)

func TestListAgentSessionsGroupsBuiltinSessionsByBoundAgent(t *testing.T) {
	previousPath := agentsconfig.ConfigPath
	previousFile := agentsconfig.ConfigFile
	agentsconfig.ConfigPath = t.TempDir()
	agentsconfig.ConfigFile = filepath.Join(agentsconfig.ConfigPath, "config.json")
	t.Cleanup(func() {
		agentsconfig.ConfigPath = previousPath
		agentsconfig.ConfigFile = previousFile
	})

	enabled := true
	service := miyaconfig.NewService()
	err := service.Save(&miyaconfig.Config{
		Agents: []miyaconfig.ACPAgentConfig{
			{ID: "miya-default", Name: "Miya Default", Enabled: &enabled, Type: "builtin", Profile: "default"},
			{ID: "miya-coding", Name: "Miya Coding", Enabled: &enabled, Type: "builtin", Profile: "coding"},
		},
		Profiles: map[string]*miyaconfig.ProfileConfig{
			"default": {Provider: "openai", ModelName: "gpt-5"},
			"coding":  {Provider: "openai", ModelName: "gpt-5-codex"},
		},
		Providers: map[string]*miyaconfig.ProviderConfig{},
	})
	if err != nil {
		t.Fatalf("save config: %v", err)
	}

	defaultSession := agentsession.New("default")
	if err := defaultSession.Save(); err != nil {
		t.Fatalf("save default session: %v", err)
	}
	codingSession := agentsession.New("coding")
	if err := codingSession.Save(); err != nil {
		t.Fatalf("save coding session: %v", err)
	}

	app := &App{config: service}
	sessions, err := app.ListAgentSessions()
	if err != nil {
		t.Fatalf("ListAgentSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(sessions))
	}
	byID := make(map[string]string, len(sessions))
	for _, session := range sessions {
		byID[session.ID] = session.AgentID
	}
	if got := byID[defaultSession.ID]; got != "miya-default" {
		t.Fatalf("default session agent = %q", got)
	}
	if got := byID[codingSession.ID]; got != "miya-coding" {
		t.Fatalf("coding session agent = %q", got)
	}
}
