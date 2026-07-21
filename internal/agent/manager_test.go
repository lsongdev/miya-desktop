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

func TestStoreForUpdateEmitsEveryLiveUpdate(t *testing.T) {
	manager := New(context.Background(), nil, nil)

	for range 2 {
		store, emit := manager.storeForUpdate("agent:session")
		if store != manager.store || !emit {
			t.Fatalf("live update = (%p, %v), want (%p, true)", store, emit, manager.store)
		}
	}
}
