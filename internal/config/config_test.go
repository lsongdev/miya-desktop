package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestServicePreservesDesktopChannelConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	initial := `{
  "profiles": {},
  "providers": {},
  "channels": [{"id":"telegram","type":"telegram","config":{"token":"secret"}}],
  "channelsEnabled": false
}`
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}
	service := &Service{path: path}

	cfg, err := service.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ChannelsEnabled == nil || *cfg.ChannelsEnabled {
		t.Fatalf("ChannelsEnabled = %#v, want false", cfg.ChannelsEnabled)
	}
	if len(cfg.Channels) != 1 {
		t.Fatalf("Channels = %#v", cfg.Channels)
	}
	if err := service.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]json.RawMessage
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("saved config is invalid: %v", err)
	}
	if len(saved["channels"]) == 0 || len(saved["channelsEnabled"]) == 0 {
		t.Fatalf("channel fields were lost: %s", data)
	}
}
