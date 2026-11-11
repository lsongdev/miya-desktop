package conversation

import (
	"miya-desktop/internal/acpadapter"
	"testing"
)

func TestFailStreamingPreservesPartialOutputAndTerminatesMessage(t *testing.T) {
	store := NewStore()
	store.RegisterSession("s1", "/workspace")
	store.AddLocalUserMessage("s1", "Read the project")
	store.ApplyACPEvent("s1", &acpadapter.Event{Type: "agent_message_chunk", Content: &acpadapter.ContentChunk{Type: "text", Content: "Partial answer"}})
	snapshot := store.FailStreaming("s1", "provider disconnected")
	message := snapshot.Conversation.Messages[len(snapshot.Conversation.Messages)-1]
	if message.Status != MessageFailed || len(message.Blocks) != 2 ||
		message.Blocks[0].Content != "Partial answer" || message.Blocks[1].Type != BlockError ||
		message.Blocks[1].Content != "provider disconnected" {
		t.Fatalf("failed turn lost its output or error: %#v", message)
	}
}
