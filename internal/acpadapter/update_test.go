package acpadapter

import (
	"encoding/json"
	"testing"

	"github.com/lsongdev/miya-agents/acp"
)

func TestParseSessionUpdateMessageChunk(t *testing.T) {
	messageID := acp.MessageID("msg-1")
	event := mustParse(t, acp.SessionUpdate{
		SessionUpdate: "agent_message_chunk",
		MessageID:     &messageID,
		Content:       acp.ContentBlock{Type: "text", Text: "hello"},
	})

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

func TestParseSessionUpdateResourceChunk(t *testing.T) {
	uri := "file:///tmp/report.pdf"
	size := 4096
	event := mustParse(t, acp.SessionUpdate{
		SessionUpdate: "agent_message_chunk",
		Content: acp.ContentBlock{
			Type: "resource", Name: "report.pdf", MimeType: "application/pdf", Size: &size, URI: &uri,
		},
	})

	if event.Content == nil || event.Content.Type != "resource" || event.Content.Name != "report.pdf" {
		t.Fatalf("content = %#v", event.Content)
	}
	if event.Content.Mime != "application/pdf" || event.Content.Size != 4096 || event.Content.URI != uri {
		t.Fatalf("metadata = %#v", event.Content)
	}
}

func TestParseSessionUpdateThoughtContentBlock(t *testing.T) {
	event := mustParse(t, acp.SessionUpdate{
		SessionUpdate: "agent_thought_chunk",
		Content:       acp.ContentBlock{Type: "text", Text: "thinking"},
	})

	if event.Content == nil || event.Content.Thought != "thinking" {
		t.Fatalf("thought = %#v", event.Content)
	}
}

func TestParseSessionUpdateToolCall(t *testing.T) {
	event := mustParse(t, acp.SessionUpdate{
		SessionUpdate: "tool_call",
		ToolCall: &acp.ToolCall{
			ToolCallID: "tc-1", Title: "Read file", Kind: acp.ToolKindRead, Status: acp.ToolCallPending,
		},
	})

	if event.Tool == nil || event.Tool.ToolCallID != acp.ToolCallID("tc-1") {
		t.Fatalf("tool = %#v", event.Tool)
	}
	if event.Tool.Title != "Read file" || event.Tool.Kind != acp.ToolKindRead {
		t.Fatalf("tool fields = %#v", event.Tool)
	}
}

func TestParseSessionUpdateToolCallUpdate(t *testing.T) {
	title := "Read file"
	status := acp.ToolCallCompleted
	event := mustParse(t, acp.SessionUpdate{
		SessionUpdate: "tool_call_update",
		ToolCallUpdate: &acp.ToolCallUpdate{
			ToolCallID: "tc-1", Title: &title, Status: &status, RawOutput: json.RawMessage(`{"ok":true}`),
		},
	})

	if event.Tool == nil || event.Tool.ToolCallID != acp.ToolCallID("tc-1") {
		t.Fatalf("tool = %#v", event.Tool)
	}
	if event.Tool.Status != acp.ToolCallCompleted || len(event.Tool.RawOutput) == 0 {
		t.Fatalf("tool update = %#v", event.Tool)
	}
}

func TestParseSessionUpdateUsageAndMode(t *testing.T) {
	usage := &acp.UsageUpdate{Size: 1000, Used: 250}
	usageEvent := mustParse(t, acp.SessionUpdate{SessionUpdate: "usage_update", Usage: usage})
	if usageEvent.Usage != usage {
		t.Fatalf("usage = %#v", usageEvent.Usage)
	}

	mode := &acp.CurrentModeUpdate{CurrentModeID: "plan"}
	modeEvent := mustParse(t, acp.SessionUpdate{SessionUpdate: "current_mode_update", CurrentMode: mode})
	if modeEvent.Mode != mode {
		t.Fatalf("mode = %#v", modeEvent.Mode)
	}
}

func TestParseSessionUpdateRequiresDiscriminator(t *testing.T) {
	if _, err := ParseSessionUpdate(acp.SessionUpdate{}); err == nil {
		t.Fatal("ParseSessionUpdate accepted an empty discriminator")
	}
}

func mustParse(t *testing.T, update acp.SessionUpdate) *Event {
	t.Helper()
	event, err := ParseSessionUpdate(update)
	if err != nil {
		t.Fatalf("ParseSessionUpdate error: %v", err)
	}
	return event
}
