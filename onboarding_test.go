package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallEmbeddedFileIfMissingPreservesExistingFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(destination, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installEmbeddedFileIfMissing("defaults/AGENTS.md", destination); err != nil {
		t.Fatalf("installEmbeddedFileIfMissing() error = %v", err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "custom" {
		t.Fatalf("existing file was replaced with %q", data)
	}
}

func TestInstallEmbeddedFileIfMissingCreatesParents(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "workspace", "AGENTS.md")
	if err := installEmbeddedFileIfMissing("defaults/AGENTS.md", destination); err != nil {
		t.Fatalf("installEmbeddedFileIfMissing() error = %v", err)
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatalf("installed file: %v", err)
	}
}
