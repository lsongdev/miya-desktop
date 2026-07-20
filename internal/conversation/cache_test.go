package conversation

import (
	"encoding/json"
	"testing"

	"github.com/lsongdev/miya-agents/acp"
)

func TestCacheRoundTripPreservesRenderablePayloads(t *testing.T) {
	cache := NewCache(t.TempDir())
	conversation := Conversation{
		ID: "agent:session",
		Messages: []Message{{
			ID: "message-1",
			Blocks: []Block{{
				ID:   "block-1",
				Type: BlockImage,
				Data: "large-base64-data",
				Raw:  json.RawMessage(`{"event":true}`),
				Tool: &acp.ToolCall{
					Title:     "exec",
					RawInput:  json.RawMessage(`{"command":"pwd"}`),
					RawOutput: json.RawMessage(`"/tmp"`),
				},
			}},
		}},
	}

	if err := cache.Save(conversation); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := cache.Load(conversation.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded == nil || len(loaded.Messages) != 1 {
		t.Fatalf("Load() = %#v", loaded)
	}
	block := loaded.Messages[0].Blocks[0]
	if block.Data != "large-base64-data" || len(block.Raw) != 0 {
		t.Fatalf("cached block payload = %#v", block)
	}
	if block.Tool == nil || string(block.Tool.RawInput) != `{"command":"pwd"}` || string(block.Tool.RawOutput) != `"/tmp"` {
		t.Fatalf("cached tool payload = %#v", block.Tool)
	}
	if conversation.Messages[0].Blocks[0].Data == "" {
		t.Fatal("Save() mutated the source conversation")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(t.TempDir())
	conversation := Conversation{ID: "agent:session"}
	if err := cache.Save(conversation); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := cache.Delete(conversation.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	loaded, err := cache.Load(conversation.ID)
	if err != nil || loaded != nil {
		t.Fatalf("Load() after delete = (%#v, %v)", loaded, err)
	}
}
