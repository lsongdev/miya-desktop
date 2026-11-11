package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"miya-desktop/internal/agent"
	"miya-desktop/internal/conversation"

	"github.com/egoist/mygo/ui"
	"github.com/lsongdev/miya-agents/acp"
)

// With MIYA_SCREENSHOT_DIR set, saves the chat and settings pages in both
// appearances as PNGs into that directory, to look at the design.
func TestNativeDesignScreenshots(t *testing.T) {
	dir := os.Getenv("MIYA_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MIYA_SCREENSHOT_DIR to a directory to save screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		for _, page := range []string{"Chat", "Providers"} {
			s := nativeFixture()
			s.prefs.Theme = theme
			s.page = page
			s.selected, s.connectedAgent = "default:s1", "default"
			s.sessions = []agent.Session{
				{ID: "s1", Title: "Redesign the desktop UI", AgentName: "default"},
				{ID: "s2", Title: "Fix ACP wire format", AgentName: "default"},
				{ID: "s3", AgentName: "opencode"},
			}
			s.conversations[s.selected] = conversation.Snapshot{Conversation: conversation.Conversation{
				ID: s.selected, ACPSessionID: "s1", Title: "Redesign the desktop UI", Model: "test-model",
				Usage: &acp.UsageUpdate{Used: 2500, Size: 128000},
				Messages: []conversation.Message{
					{ID: "user", Role: "user", Status: "complete", Blocks: []conversation.Block{{ID: "b1", Type: "text", Content: "Make the desktop UI use an island style."}}},
					{ID: "answer", Role: "assistant", Status: "complete", Blocks: []conversation.Block{
						{ID: "b3", Type: "thought", Content: "Inspecting the project"},
						{ID: "b4", Type: "tool_call", Tool: &acp.ToolCall{ToolCallID: "tool1", Title: "Read native_view.go", Status: "completed", RawOutput: json.RawMessage(`{"ok":true}`)}},
						{ID: "b2", Type: "markdown", Content: "I'll split the window into **two islands**: a sidebar and the workspace.\n\n- rounded panels on a darker gutter\n- a composer card at the bottom"},
						{ID: "b5", Type: "plan", Plan: &acp.Plan{Entries: []acp.PlanEntry{{Content: "Sidebar island", Status: "completed", Priority: "high"}, {Content: "Composer card", Status: "in_progress", Priority: "high"}, {Content: "Settings pages", Status: "pending", Priority: "low"}}}},
					}},
				},
			}}
			tt := ui.NewTester(s.view, 1200, 800)
			tt.Frame()
			out, err := os.Create(filepath.Join(dir, theme+"-"+page+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(out, tt.Image()); err != nil {
				t.Fatal(err)
			}
			out.Close()
		}
	}
}
