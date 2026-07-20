package agent

import (
	"context"
	"testing"

	miyaconfig "wails-app/internal/config"
	"wails-app/internal/conversation"
)

func TestGetConversationMissingReturnsNil(t *testing.T) {
	manager := New(context.Background(), nil, nil)

	conversation, err := manager.GetConversation("missing")
	if err != nil {
		t.Fatalf("GetConversation returned error: %v", err)
	}
	if conversation != nil {
		t.Fatalf("GetConversation returned conversation: %#v", conversation)
	}
}

func TestConnectEndpointRejectsIDsThatBreakSessionKeys(t *testing.T) {
	manager := New(context.Background(), nil, nil)
	err := manager.ConnectEndpoint(miyaconfig.ACPAgentConfig{ID: "miya:default", Type: "builtin"})
	if err == nil {
		t.Fatal("ConnectEndpoint succeeded with colon in id")
	}
}

func TestShouldEmitConversationUpdateThrottlesPerSession(t *testing.T) {
	manager := New(context.Background(), nil, nil)

	if !manager.shouldEmitConversationUpdate("s1") {
		t.Fatal("first update should be emitted")
	}
	if manager.shouldEmitConversationUpdate("s1") {
		t.Fatal("immediate repeated update should be throttled")
	}
	if !manager.shouldEmitConversationUpdate("s2") {
		t.Fatal("a different session should be emitted independently")
	}
}

func TestGetConversationHydratesPersistentCache(t *testing.T) {
	manager := New(context.Background(), nil, nil)
	manager.cache = conversation.NewCache(t.TempDir())
	want := conversation.Conversation{
		ID: "agent:session",
		Messages: []conversation.Message{{
			ID:     "message-1",
			Role:   conversation.RoleAssistant,
			Status: conversation.MessageComplete,
			Blocks: []conversation.Block{{ID: "block-1", Type: conversation.BlockText, Content: "cached"}},
		}},
	}
	if err := manager.cache.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := manager.GetConversation(want.ID)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if got == nil || len(got.Messages) != 1 || got.Messages[0].Blocks[0].Content != "cached" {
		t.Fatalf("GetConversation() = %#v", got)
	}
	if !manager.store.HasMessages(want.ID) {
		t.Fatal("cached conversation was not hydrated into the active store")
	}
}

func TestStoreForUpdateUsesReplayStoreAndBatchesProgress(t *testing.T) {
	manager := New(context.Background(), nil, nil)
	replayStore := conversation.NewStore()
	manager.replays["agent:session"] = &sessionReplay{store: replayStore, showProgress: true}

	got, emit := manager.storeForUpdate("agent:session")
	if got != replayStore || !emit {
		t.Fatalf("first replay update = (%p, %v), want (%p, true)", got, emit, replayStore)
	}
	got, emit = manager.storeForUpdate("agent:session")
	if got != replayStore || emit {
		t.Fatalf("immediate replay update = (%p, %v), want (%p, false)", got, emit, replayStore)
	}

	manager.replays["agent:session"].showProgress = false
	got, emit = manager.storeForUpdate("agent:session")
	if got != replayStore || emit {
		t.Fatalf("cached replay update = (%p, %v), want (%p, false)", got, emit, replayStore)
	}
}
