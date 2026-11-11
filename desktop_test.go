package main

import (
	"encoding/json/v2"
	"testing"

	"miya-desktop/internal/acpadapter"
	miyaconfig "miya-desktop/internal/config"
	"miya-desktop/internal/conversation"

	"github.com/lsongdev/miya-agents/acp"
)

func TestDesktopConfigJSONRoundTrip(t *testing.T) {
	// Shared fields and raw channel objects must retain the existing config shape.
	wire := []byte(`{
		"profiles":{"default":{"provider":"local","model":"test"}},
		"providers":{"local":{"type":"openai","apiKey":"test-secret","apiBase":"https://example.test/v1"}},
		"agents":[{"id":"external","type":"stdio","command":"agent","args":["acp"]}],
		"channels":[{"type":"telegram","config":{"token":"test-token"}}],
		"channelsEnabled":false
	}`)
	var cfg miyaconfig.Config
	if err := json.Unmarshal(wire, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Providers["local"].APIKey != "test-secret" || cfg.Profiles["default"].ModelName != "test" {
		t.Fatalf("shared config fields lost during JSON serialization")
	}
	if cfg.ChannelsEnabled == nil || *cfg.ChannelsEnabled || len(cfg.Channels) != 1 {
		t.Fatalf("desktop config fields lost during JSON serialization")
	}
	encoded, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	channels, ok := got["channels"].([]any)
	if !ok || len(channels) != 1 {
		t.Fatalf("raw channel data changed shape: %s", encoded)
	}
	channel, ok := channels[0].(map[string]any)
	if !ok || channel["type"] != "telegram" {
		t.Fatalf("raw channel data was not encoded as an object: %s", encoded)
	}
	var restored miyaconfig.Config
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Providers["local"].APIKey != "test-secret" ||
		restored.Profiles["default"].ModelName != "test" ||
		len(restored.Agents) != 1 || restored.Agents[0].Command != "agent" ||
		restored.ChannelsEnabled == nil || *restored.ChannelsEnabled {
		t.Fatal("config did not survive the response and subsequent save")
	}
}

func TestDesktopStreamingTimelineSnapshots(t *testing.T) {
	store := conversation.NewStore()
	store.RegisterSessionWithACP("default:s1", "s1", "/workspace")
	store.AddLocalUserMessage("default:s1", "Inspect the project")
	updates := []string{
		`{"sessionUpdate":"agent_message_chunk","messageId":"m1","content":{"type":"text","text":"Hello "}}`,
		`{"sessionUpdate":"agent_message_chunk","messageId":"m1","content":{"type":"text","text":"world"}}`,
		`{"sessionUpdate":"agent_thought_chunk","thought":"Inspecting files"}`,
		`{"sessionUpdate":"tool_call","toolCall":{"toolCallId":"t1","title":"Read file","status":"in_progress","rawInput":{"path":"main.go"}}}`,
		`{"sessionUpdate":"tool_call_update","toolCallUpdate":{"toolCallId":"t1","status":"completed","rawOutput":{"text":"package main"}}}`,
		`{"sessionUpdate":"plan","plan":{"entries":[{"content":"Inspect project","priority":"high","status":"completed"}]}}`,
	}
	state := newNativeUI()
	for _, wire := range updates {
		var update acp.SessionUpdate
		if err := json.Unmarshal([]byte(wire), &update); err != nil {
			t.Fatal(err)
		}
		event, err := acpadapter.ParseSessionUpdate(update)
		if err != nil {
			t.Fatal(err)
		}
		snapshot := store.ApplyACPEvent("default:s1", event)
		state.receive(snapshot)
	}
	snapshot, ok := store.CompleteStreaming("default:s1")
	if !ok {
		t.Fatal("missing completed timeline")
	}
	snapshot.StopReason = "end_turn"
	state.receive(snapshot)
	wire, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var restored conversation.Snapshot
	if err := json.Unmarshal(wire, &restored); err != nil {
		t.Fatal(err)
	}
	var text, thought string
	var tool *acp.ToolCall
	var plan *acp.Plan
	for _, message := range restored.Conversation.Messages {
		for _, block := range message.Blocks {
			switch block.Type {
			case "text", "markdown":
				if message.Role == "assistant" {
					text += block.Content
				}
			case "thought":
				thought += block.Content
			case conversation.BlockToolCall:
				tool = block.Tool
			case "plan":
				plan = block.Plan
			}
		}
	}
	if text != "Hello world" || thought != "Inspecting files" ||
		tool == nil || tool.Status != "completed" || plan == nil || len(plan.Entries) != 1 ||
		restored.StopReason != "end_turn" {
		t.Fatalf("streamed timeline lost text, thoughts, tools, plan or completion: %s", wire)
	}
}

func TestDesktopConversationEventContract(t *testing.T) {
	snapshot := conversation.Snapshot{
		Conversation: conversation.Conversation{ID: "default:s1", ACPSessionID: "s1"},
		EventType:    "complete", StopReason: "end_turn",
	}
	state := newNativeUI()
	state.receive(snapshot)
	if state.conversations["default:s1"].StopReason != "end_turn" {
		t.Fatal("native timeline lost completion metadata")
	}
	wire, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Conversation struct {
			ID string `json:"id"`
		} `json:"conversation"`
		StopReason string `json:"stopReason"`
	}
	if err := json.Unmarshal(wire, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Conversation.ID != snapshot.Conversation.ID || payload.StopReason != "end_turn" {
		t.Fatalf("conversation identity or stop reason lost: %s", wire)
	}
}
