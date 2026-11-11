package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"miya-desktop/internal/agent"
	channelservice "miya-desktop/internal/channels"
	miyaconfig "miya-desktop/internal/config"
	"miya-desktop/internal/conversation"

	"github.com/egoist/mygo/ui"
	"github.com/lsongdev/miya-agents/acp"
	agentsconfig "github.com/lsongdev/miya-agents/config"
)

func nativeFixture() *nativeUI {
	s := newNativeUI()
	s.loading = false
	s.config = &miyaconfig.Config{SharedConfig: miyaconfig.SharedConfig{
		Profiles:  map[string]*miyaconfig.ProfileConfig{"default": {Provider: "local", ModelName: "test-model"}},
		Providers: map[string]*miyaconfig.ProviderConfig{"local": {Type: "openai", APIKey: "fixture-secret", Protocol: "chat", Models: map[string]string{"public": "upstream"}}},
	}}
	s.setEndpoints()
	s.nextPoll = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	s.launch = func(job func()) { job() }
	s.dispatch = func(commit func()) { commit() }
	return s
}

func nativeClick(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatalf("click %q: %v; texts: %q", label, err, tt.Texts())
	}
}

func nativeType(t *testing.T, tt *ui.Tester, label, text string) {
	t.Helper()
	nativeClick(t, tt, label)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(text)
}

func TestNativePagesAndEditors(t *testing.T) {
	for _, page := range settingsPages {
		t.Run(page, func(t *testing.T) {
			s := nativeFixture()
			s.page = page
			tt := ui.NewTester(s.view, 1100, 760)
			if !tt.HasText(page) {
				t.Fatalf("missing page %s: %q", page, tt.Texts())
			}
			if page == "General" || page == "About" {
				return
			}
			kind := map[string]string{"Agents": "Agent", "Profiles": "Profile", "Providers": "Provider", "MCP Servers": "MCP Server", "Skills": "Skill", "Channels": "Channel"}[page]
			nativeClick(t, tt, "Add "+kind)
			if !s.editor.Open || !tt.HasText(kind+" ID") {
				t.Fatalf("native form unavailable for %s: %q", page, tt.Texts())
			}
			nativeClick(t, tt, "Cancel")
			if s.editor.Open {
				t.Fatal("cancel did not close editor")
			}
		})
	}
}

func TestNativeProviderRenamePersistsAndPreservesOptions(t *testing.T) {
	s := nativeFixture()
	backend := nativeTestBackend(t, s)
	s.page = "Providers"
	tt := ui.NewTester(s.view, 1100, 760)
	nativeClick(t, tt, "Edit Provider local")
	nativeType(t, tt, "Provider ID", "renamed")
	nativeClick(t, tt, "Save")
	if s.err != "" {
		t.Fatal(s.err)
	}
	if s.editor.Open {
		t.Fatal("successful save left form open")
	}
	saved, err := backend.LoadMiyaConfig()
	if err != nil {
		t.Fatal(err)
	}
	provider := saved.Providers["renamed"]
	if saved.Providers["local"] != nil || provider == nil || provider.APIKey != "fixture-secret" ||
		provider.Protocol != "chat" || provider.Models["public"] != "upstream" || saved.Profiles["default"].Provider != "renamed" {
		t.Fatal("rename dropped credentials, advanced options, or dependent profile references")
	}
}

func nativeTestBackend(t *testing.T, s *nativeUI) *App {
	t.Helper()
	oldPath, oldFile := agentsconfig.ConfigPath, agentsconfig.ConfigFile
	agentsconfig.ConfigPath = t.TempDir()
	agentsconfig.ConfigFile = filepath.Join(agentsconfig.ConfigPath, "config.json")
	t.Cleanup(func() { agentsconfig.ConfigPath, agentsconfig.ConfigFile = oldPath, oldFile })
	service := miyaconfig.NewService()
	for _, profile := range s.config.Profiles {
		profile.Workspace = t.TempDir()
	}
	if err := service.Save(s.config); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	b := NewApp(s.emit)
	b.ctx, b.config = ctx, service
	b.manager = agent.New(ctx, service.Load, s.emit)
	b.channels = channelservice.NewService(service.Load, s.emit)
	s.backend = b
	t.Cleanup(func() { cancel(); b.shutdown() })
	return b
}

func TestNativeArgumentsAndDraftValidation(t *testing.T) {
	cfg := nativeFixture().config
	err := applyEditor(cfg, configEditor{Kind: "Agent", ID: "external", Command: "/path with spaces/agent", Args: `["acp","--cwd","/workspace with spaces"]`, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Agents[0].Args) != 3 || cfg.Agents[0].Args[2] != "/workspace with spaces" {
		t.Fatal("arguments were split at spaces")
	}
	if err := applyEditor(cfg, configEditor{Kind: "Agent", ID: "default", Command: "other"}); err == nil {
		t.Fatal("profile/agent ID collision accepted")
	}
	for _, value := range []string{"-1", "NaN", "+Inf", "not-a-number"} {
		if _, err := parseDecimal(value, "temperature"); err == nil {
			t.Fatalf("invalid numeric setting %q accepted", value)
		}
	}
	if _, err := parsePairs("TOKEN"); err == nil {
		t.Fatal("malformed environment line accepted")
	}
	args, err := parseArgs(encodeArgs(nil))
	if err != nil || len(args) != 0 {
		t.Fatal("an empty argument list became a literal null argument")
	}
}

func TestNativeChatTimelineAndMultilineComposition(t *testing.T) {
	s := nativeFixture()
	s.selected, s.connectedAgent = "default:s1", "default"
	s.conversations[s.selected] = conversation.Snapshot{StopReason: "end_turn", Conversation: conversation.Conversation{
		ID: s.selected, ACPSessionID: "s1", Title: "Native timeline", Usage: &acp.UsageUpdate{Used: 250, Size: 128000},
		Messages: []conversation.Message{
			{ID: "user", Role: "user", Status: "complete", Blocks: []conversation.Block{{ID: "b1", Type: "text", Content: "Question"}}},
			{ID: "answer", Role: "assistant", Status: "complete", Blocks: []conversation.Block{
				{ID: "b2", Type: "markdown", Content: "**Native answer**"},
				{ID: "b3", Type: "thought", Content: "Inspecting the project"},
				{ID: "b4", Type: "tool_call", Tool: &acp.ToolCall{ToolCallID: "tool1", Title: "Read file", Status: "completed", RawOutput: json.RawMessage(`{"ok":true}`)}},
				{ID: "b5", Type: "plan", Plan: &acp.Plan{Entries: []acp.PlanEntry{{Content: "Finish native UI", Status: "completed", Priority: "high"}}}},
			}},
		},
	}}
	tt := ui.NewTester(s.view, 1100, 900)
	if !tt.HasText("Native answer") || !tt.HasText("Read file") || !tt.HasText("Finish native UI") || !tt.HasText("Stopped: end_turn") {
		t.Fatalf("timeline blocks missing: %q", tt.Texts())
	}
	nativeClick(t, tt, "Thoughts")
	tt.Advance(time.Second)
	if !tt.HasText("Inspecting the project") {
		t.Fatal("thought disclosure did not reveal content")
	}
	nativeType(t, tt, "Message", "你好，Miya")
	tt.Key(ui.Shift, ui.KeyEnter)
	tt.Type("second line")
	if s.draft != "你好，Miya\nsecond line" {
		t.Fatalf("multiline composer = %q", s.draft)
	}
	nativeClick(t, tt, "Copy message answer")
	if !strings.Contains(tt.Clipboard(), "Native answer") {
		t.Fatal("native copy action lost Markdown text")
	}
}

func TestNativeNotificationsOnlyForNewBackgroundCompletion(t *testing.T) {
	s := nativeFixture()
	s.active = false
	var notes []DesktopNotification
	s.notify = func(note DesktopNotification) { notes = append(notes, note) }
	snapshot := conversation.Snapshot{Conversation: conversation.Conversation{ID: "default:s1", Messages: []conversation.Message{
		{ID: "m1", Role: "assistant", Status: "streaming", Blocks: []conversation.Block{{Type: "text", Content: "Done"}}},
	}}}
	s.receive(snapshot)
	// Snapshots are immutable; do not mutate the previous snapshot's slice.
	completed := conversation.Snapshot{Conversation: conversation.Conversation{ID: "default:s1", Messages: []conversation.Message{
		{ID: "m1", Role: "assistant", Status: "complete", Blocks: []conversation.Block{{Type: "text", Content: "Done"}}},
	}}}
	s.receive(completed)
	s.receive(completed)
	if len(notes) != 1 || notes[0].Body != "Done" {
		t.Fatalf("notifications = %#v", notes)
	}
}

func TestNativeConversationThroughEmbeddedACP(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "unexpected endpoint", 404)
			return
		}
		requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"chat1\",\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Native response\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"id\":\"chat1\",\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	s := nativeFixture()
	s.config.Providers["local"].APIBase = server.URL
	s.config.Providers["local"].Protocol = ""
	nativeTestBackend(t, s)
	// Queue backend events so only this test's UI thread applies state.
	var mu sync.Mutex
	var pending []func()
	s.dispatch = func(commit func()) { mu.Lock(); pending = append(pending, commit); mu.Unlock() }
	flush := func() {
		for {
			mu.Lock()
			batch := pending
			pending = nil
			mu.Unlock()
			if len(batch) == 0 {
				return
			}
			for _, commit := range batch {
				commit()
			}
		}
	}
	s.newSession()
	flush()
	if s.selected == "" || s.err != "" {
		t.Fatalf("new session: %s", s.err)
	}
	tt := ui.NewTester(s.view, 1100, 760)
	nativeType(t, tt, "Message", "Hello from native UI")
	tt.Key(0, ui.KeyEnter)
	flush()
	tt.Frame()
	if s.err != "" {
		t.Fatal(s.err)
	}
	if requestCount.Load() != 1 || !tt.HasText("Native response") {
		t.Fatalf("native ACP/model response missing: count=%d texts=%q", requestCount.Load(), tt.Texts())
	}
	if s.sending || s.draft != "" {
		t.Fatal("composer did not return to its ready state")
	}
	if s.conversations[s.selected].StopReason == "" {
		t.Fatal("completion reason did not reach native state")
	}
}

func TestNativeScreenshots(t *testing.T) {
	dir := os.Getenv("MIYA_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MIYA_SCREENSHOT_DIR to render review images")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	s := nativeFixture()
	s.page = "About"
	s.logo, _ = ui.DecodeBitmap(appIcon)
	tt := ui.NewTester(s.view, 1100, 760)
	file, err := os.Create(filepath.Join(dir, "native-about.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestNativeAttachmentBytesAndPreviewLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attachment.txt")
	content := []byte("native attachment bytes")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		filePath = "/" + filePath
	}
	target := (&url.URL{Scheme: "file", Path: filePath}).String()
	attachment, err := NewApp(nil).ReadAttachment(target, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(attachment.Data, content) {
		t.Fatal("native attachment API did not return raw bytes")
	}
	if _, err := NewApp(nil).ReadAttachment(target, 3); err == nil {
		t.Fatal("attachment preview size limit was ignored")
	}
	if runtime.GOOS != "windows" {
		if _, err := os.Stat("/dev/zero"); err == nil {
			if _, err := NewApp(nil).ReadAttachment("file:///dev/zero", 16); err == nil {
				t.Fatal("size limit did not apply to a stream with a zero stat size")
			}
		}
	}
}

func TestNativeStopCancelsEmbeddedTurn(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"stream\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Partial answer\"}}]}\n\n")
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	s := nativeFixture()
	s.config.Providers["local"].APIBase = server.URL
	s.config.Providers["local"].Protocol = ""
	nativeTestBackend(t, s)
	commits := make(chan func(), 256)
	s.dispatch = func(commit func()) { commits <- commit }
	s.newSession()
	for len(commits) > 0 {
		(<-commits)()
	}
	if s.selected == "" || s.err != "" {
		t.Fatalf("create session: %s", s.err)
	}
	s.launch = func(job func()) { go job() }
	tt := ui.NewTester(s.view, 1100, 760)
	nativeType(t, tt, "Message", "Wait until I stop")
	tt.Key(0, ui.KeyEnter)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("provider request never started")
	}
	nativeClick(t, tt, "Stop")
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for s.sending {
		select {
		case commit := <-commits:
			commit()
		case <-deadline.C:
			t.Fatal("Stop did not release the composer")
		}
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel the provider request")
	}
	if s.conversations[s.selected].StopReason != "cancelled" {
		t.Fatalf("stop reason: %q; error: %s", s.conversations[s.selected].StopReason, s.err)
	}
	tt.Frame()
	if !tt.HasText("Send") {
		t.Fatal("send action did not return after cancellation")
	}
}
