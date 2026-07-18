package agent

import (
	"context"
	"testing"

	miyaconfig "wails-app/internal/config"
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
