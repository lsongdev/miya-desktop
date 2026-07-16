package acpadapter

import (
	"encoding/json"
	"testing"

	"github.com/lsongdev/miya-agents/acp"
)

func TestParseUpdateMessageChunk(t *testing.T) {
	event := mustParse(t, `{
		"sessionUpdate": "agent_message_chunk",
		"messageId": "msg-1",
		"content": {"type": "text", "text": "hello"}
	}`)

	if event.Type != "agent_message_chunk" {
		t.Fatalf("type = %q", event.Type)
	}
	if event.Content == nil || event.Content.Content != "hello" {
		t.Fatalf("content = %#v", event.Content)
	}
	if event.Content.MessageID != "msg-1" {
		t.Fatalf("message id = %q", event.Content.MessageID)
	}
}

func TestParseUpdateResourceChunk(t *testing.T) {
	event := mustParse(t, `{
		"sessionUpdate": "agent_message_chunk",
		"content": {
			"type": "resource",
			"name": "report.pdf",
			"mimeType": "application/pdf",
			"size": 4096,
			"uri": "file:///tmp/report.pdf"
		}
	}`)

	if event.Content == nil {
		t.Fatalf("content is nil")
	}
	if event.Content.Type != "resource" || event.Content.Name != "report.pdf" {
		t.Fatalf("content = %#v", event.Content)
	}
	if event.Content.Mime != "application/pdf" || event.Content.Size != 4096 {
		t.Fatalf("metadata = %#v", event.Content)
	}
	if event.Content.URI != "file:///tmp/report.pdf" {
		t.Fatalf("uri = %q", event.Content.URI)
	}
}

func TestParseUpdateThoughtContentBlock(t *testing.T) {
	event := mustParse(t, `{
		"sessionUpdate": "agent_thought_chunk",
		"content": {"type": "text", "text": "thinking"}
	}`)

	if event.Content == nil || event.Content.Thought != "thinking" {
		t.Fatalf("thought = %#v", event.Content)
	}
}

func TestParseUpdateInlineToolCall(t *testing.T) {
	event := mustParse(t, `{
		"sessionUpdate": "tool_call",
		"toolCallId": "tc-1",
		"title": "Read file",
		"kind": "read",
		"status": "pending",
		"content": []
	}`)

	if event.Tool == nil || event.Tool.ToolCallID != acp.ToolCallID("tc-1") {
		t.Fatalf("tool = %#v", event.Tool)
	}
	if event.Tool.Title != "Read file" || event.Tool.Kind != acp.ToolKindRead {
		t.Fatalf("tool fields = %#v", event.Tool)
	}
}

func TestParseUpdateNestedToolCallUpdate(t *testing.T) {
	event := mustParse(t, `{
		"sessionUpdate": "tool_call_update",
		"toolCallUpdate": {
			"toolCallId": "tc-1",
			"title": "Read file",
			"status": "completed",
			"rawOutput": {"ok": true}
		}
	}`)

	if event.Tool == nil || event.Tool.ToolCallID != acp.ToolCallID("tc-1") {
		t.Fatalf("tool = %#v", event.Tool)
	}
	if event.Tool.Status != acp.ToolCallCompleted {
		t.Fatalf("status = %q", event.Tool.Status)
	}
	if len(event.Tool.RawOutput) == 0 {
		t.Fatalf("raw output was not preserved")
	}
}

func TestParseUpdateUsageFlatShape(t *testing.T) {
	event := mustParse(t, `{
		"sessionUpdate": "usage_update",
		"size": 1000,
		"used": 250
	}`)

	if event.Usage == nil || event.Usage.Size != 1000 || event.Usage.Used != 250 {
		t.Fatalf("usage = %#v", event.Usage)
	}
}

func TestParseUpdateModeFlatShape(t *testing.T) {
	event := mustParse(t, `{
		"sessionUpdate": "current_mode_update",
		"currentModeId": "plan"
	}`)

	if event.Mode == nil || event.Mode.CurrentModeID != acp.SessionModeID("plan") {
		t.Fatalf("mode = %#v", event.Mode)
	}
}

func mustParse(t *testing.T, raw string) *Event {
	t.Helper()
	event, err := ParseUpdate(json.RawMessage(raw))
	if err != nil {
		t.Fatalf("ParseUpdate error: %v", err)
	}
	return event
}
