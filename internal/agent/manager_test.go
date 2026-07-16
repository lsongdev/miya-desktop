package agent

import (
	"context"
	"testing"
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
