package conversation

import (
	"encoding/json"
	"testing"
	"time"

	"wails-app/internal/acpadapter"

	"github.com/lsongdev/miya-agents/acp"
)

func TestStoreReducesStreamingMessageChunks(t *testing.T) {
	store := testStore()

	store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "agent_message_chunk",
		"messageId": "m1",
		"content": {"type": "text", "text": "hel"}
	}`))
	snap := store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "agent_message_chunk",
		"messageId": "m1",
		"content": {"type": "text", "text": "lo"}
	}`))

	msgs := snap.Conversation.Messages
	if len(msgs) != 1 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if got := msgs[0].Blocks[0].Content; got != "hello" {
		t.Fatalf("content = %q", got)
	}
}

func TestStoreSeparatesReplayedMessagesWithoutMessageID(t *testing.T) {
	store := testStore()

	store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "user_message_chunk",
		"content": {"type": "text", "text": "first"}
	}`))
	snap, ok := store.CompleteStreaming("s1")
	if !ok {
		t.Fatalf("missing snapshot")
	}
	if snap.Conversation.Messages[0].Status != MessageComplete {
		t.Fatalf("message was not completed")
	}
	snap = store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "user_message_chunk",
		"content": {"type": "text", "text": "second"}
	}`))

	if len(snap.Conversation.Messages) != 2 {
		t.Fatalf("messages = %d", len(snap.Conversation.Messages))
	}
}

func TestStoreResetSessionDropsPreviousReplay(t *testing.T) {
	store := testStore()

	store.ResetSessionWithACP("miya:s1", "s1", "/tmp/project")
	store.ApplyACPEvent("miya:s1", mustEvent(t, `{
		"sessionUpdate": "user_message_chunk",
		"content": {"type": "text", "text": "first"}
	}`))
	store.CompleteStreaming("miya:s1")

	snap := store.ResetSessionWithACP("miya:s1", "s1", "/tmp/project")
	if len(snap.Conversation.Messages) != 0 {
		t.Fatalf("messages after reset = %d", len(snap.Conversation.Messages))
	}
	if snap.Conversation.ACPSessionID != "s1" {
		t.Fatalf("acp session id = %q", snap.Conversation.ACPSessionID)
	}
	if snap.Conversation.Cwd != "/tmp/project" {
		t.Fatalf("cwd = %q", snap.Conversation.Cwd)
	}
}

func TestStoreHasMessages(t *testing.T) {
	store := testStore()

	store.RegisterSessionWithACP("miya:s1", "s1", "/tmp/project")
	if store.HasMessages("miya:s1") {
		t.Fatal("empty registered session should not have messages")
	}

	store.ApplyACPEvent("miya:s1", mustEvent(t, `{
		"sessionUpdate": "user_message_chunk",
		"content": {"type": "text", "text": "hello"}
	}`))
	if !store.HasMessages("miya:s1") {
		t.Fatal("session with replayed message should have messages")
	}
}

func TestStoreSetModelOnlySnapshotsChanges(t *testing.T) {
	store := testStore()
	store.RegisterSession("s1", "")

	snapshot, changed := store.SetModel("s1", "model-a")
	if !changed || snapshot.Conversation.Model != "model-a" {
		t.Fatalf("first SetModel = (%q, %v)", snapshot.Conversation.Model, changed)
	}
	if _, changed := store.SetModel("s1", "model-a"); changed {
		t.Fatal("unchanged model should not produce a snapshot")
	}
}

func TestStoreQuietApplyAndReplace(t *testing.T) {
	store := testStore()
	event := mustEvent(t, `{
		"sessionUpdate": "agent_message_chunk",
		"content": {"type": "text", "text": "cached"}
	}`)
	store.ApplyACPEventQuiet("s1", event)
	snapshot, ok := store.Snapshot("s1")
	if !ok || snapshot.Conversation.Messages[0].Blocks[0].Content != "cached" {
		t.Fatalf("quiet snapshot = %#v", snapshot)
	}

	replacement := Conversation{ID: "s1", Title: "replayed"}
	replaced := store.Replace(replacement, "replay_completed")
	if replaced.EventType != "replay_completed" || replaced.Conversation.Title != "replayed" {
		t.Fatalf("Replace() = %#v", replaced)
	}
	store.Delete("s1")
	if _, ok := store.Snapshot("s1"); ok {
		t.Fatal("Delete() retained the conversation")
	}
}

func TestStoreAddsThoughtAndToolBlocksToAssistantMessage(t *testing.T) {
	store := testStore()

	store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "agent_thought_chunk",
		"content": {"type": "text", "text": "think"}
	}`))
	snap := store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "tool_call",
		"toolCallId": "tc-1",
		"title": "Read",
		"kind": "read",
		"status": "pending",
		"content": []
	}`))

	msgs := snap.Conversation.Messages
	if len(msgs) != 1 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if len(msgs[0].Blocks) != 2 {
		t.Fatalf("blocks = %#v", msgs[0].Blocks)
	}
	if msgs[0].Blocks[0].Type != BlockThought || msgs[0].Blocks[0].Content != "think" {
		t.Fatalf("thought block = %#v", msgs[0].Blocks[0])
	}
	if msgs[0].Blocks[1].Type != BlockToolCall || msgs[0].Blocks[1].Tool.ToolCallID != acp.ToolCallID("tc-1") {
		t.Fatalf("tool block = %#v", msgs[0].Blocks[1])
	}
}

func TestStoreAddsResourceBlock(t *testing.T) {
	store := testStore()

	snap := store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "agent_message_chunk",
		"content": {
			"type": "resource",
			"name": "report.pdf",
			"mimeType": "application/pdf",
			"size": 4096,
			"uri": "file:///tmp/report.pdf"
		}
	}`))

	msgs := snap.Conversation.Messages
	if len(msgs) != 1 || len(msgs[0].Blocks) != 1 {
		t.Fatalf("messages = %#v", msgs)
	}
	block := msgs[0].Blocks[0]
	if block.Type != BlockResource || block.Name != "report.pdf" {
		t.Fatalf("block = %#v", block)
	}
	if block.Mime != "application/pdf" || block.Size != 4096 || block.URI != "file:///tmp/report.pdf" {
		t.Fatalf("metadata = %#v", block)
	}
}

func TestStoreMergesToolUpdates(t *testing.T) {
	store := testStore()

	store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "tool_call",
		"toolCallId": "tc-1",
		"title": "Read",
		"kind": "read",
		"status": "pending",
		"content": []
	}`))
	snap := store.ApplyACPEvent("s1", mustEvent(t, `{
		"sessionUpdate": "tool_call_update",
		"toolCallId": "tc-1",
		"status": "completed",
		"rawOutput": {"ok": true}
	}`))

	tool := snap.Conversation.Messages[0].Blocks[0].Tool
	if tool.Status != acp.ToolCallCompleted {
		t.Fatalf("status = %q", tool.Status)
	}
	if tool.Title != "Read" || tool.Kind != acp.ToolKindRead {
		t.Fatalf("existing fields were not preserved: %#v", tool)
	}
	if len(tool.RawOutput) == 0 {
		t.Fatalf("raw output was not merged")
	}
}

func TestStoreTracksSessionInfoUsageAndMode(t *testing.T) {
	store := testStore()

	title := "Project Chat"
	store.ApplyACPEvent("s1", &acpadapter.Event{
		Type: "session_info_update",
		Info: &acp.SessionInfoUpdate{Title: &title},
	})
	store.ApplyACPEvent("s1", &acpadapter.Event{
		Type:  "usage_update",
		Usage: &acp.UsageUpdate{Size: 1000, Used: 200},
	})
	snap := store.ApplyACPEvent("s1", &acpadapter.Event{
		Type: "current_mode_update",
		Mode: &acp.CurrentModeUpdate{CurrentModeID: "plan"},
	})

	if snap.Conversation.Title != title {
		t.Fatalf("title = %q", snap.Conversation.Title)
	}
	if snap.Conversation.Usage == nil || snap.Conversation.Usage.Used != 200 {
		t.Fatalf("usage = %#v", snap.Conversation.Usage)
	}
	if snap.Conversation.Mode == nil || snap.Conversation.Mode.CurrentModeID != "plan" {
		t.Fatalf("mode = %#v", snap.Conversation.Mode)
	}
}

func testStore() *Store {
	store := NewStore()
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	return store
}

func mustEvent(t *testing.T, raw string) *acpadapter.Event {
	t.Helper()
	event, err := acpadapter.ParseUpdate(json.RawMessage(raw))
	if err != nil {
		t.Fatalf("ParseUpdate error: %v", err)
	}
	return event
}
