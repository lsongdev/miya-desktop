package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureNativeReleaseMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mygo.json")
	if err := os.WriteFile(path, []byte(`{"name":"Miya Desktop","version":"0.1.0","linux":{"webview":false}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := configure(path, "26.10.29", "public-key", true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Version string `json:"version"`
		Updates struct {
			PublicKey string `json:"publicKey"`
		} `json:"updates"`
		Linux struct {
			Webview bool `json:"webview"`
		} `json:"linux"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Version != "26.10.29" || cfg.Updates.PublicKey != "public-key" || cfg.Linux.Webview {
		t.Fatalf("native package configuration lost: %s", data)
	}
	if strings.Contains(string(data), "PRIVATE") {
		t.Fatal("private key leaked into package configuration")
	}
}

func TestConfigureRejectsIncompleteSigningAndVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mygo.json")
	original := []byte(`{"name":"Miya Desktop","version":"0.1.0"}`)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"1.2", "01.2.3", "1.-2.3", "1.2.3.4"} {
		if err := configure(path, version, "", false); err == nil {
			t.Fatalf("invalid version %q accepted", version)
		}
	}
	if err := configure(path, "", "public-key", false); err == nil {
		t.Fatal("missing signing key accepted")
	}
	if err := configure(path, "", "", true); err == nil {
		t.Fatal("missing public key accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatal("failed configuration modified source metadata")
	}
}
