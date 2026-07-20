package main

import (
	"reflect"
	"testing"
)

func TestSafeRegistryFilePathRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	if _, err := safeRegistryFilePath(root, "../SKILL.md"); err == nil {
		t.Fatal("safeRegistryFilePath accepted a parent path")
	}
	if _, err := safeRegistryFilePath(root, "/tmp/SKILL.md"); err == nil {
		t.Fatal("safeRegistryFilePath accepted an absolute path")
	}
	if _, err := safeRegistryFilePath(root, "examples/demo.md"); err != nil {
		t.Fatalf("safeRegistryFilePath rejected a nested file: %v", err)
	}
}

func TestMCPInstallCandidatePrefersUnauthenticatedRemote(t *testing.T) {
	candidate, ok := mcpInstallCandidate(mcpRegistryServer{
		Name:    "io.example/docs",
		Title:   "Example Docs",
		Version: "1.0.0",
		Packages: []mcpRegistryPackage{{
			RegistryType: "npm",
			Identifier:   "example-docs",
		}},
		Remotes: []mcpRegistryRemote{{Type: "streamable-http", URL: "https://example.com/mcp"}},
	})
	if !ok || candidate.Config == nil {
		t.Fatalf("mcpInstallCandidate() = (%#v, %v)", candidate, ok)
	}
	if candidate.Config.Type != "streamablehttp" || candidate.Config.URL != "https://example.com/mcp" || candidate.InstallLabel != "Remote" {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestPackageMCPConfigBuildsNPXCommandAndRequiredInputs(t *testing.T) {
	config, label, required := packageMCPConfig(mcpRegistryPackage{
		RegistryType:         "npm",
		Identifier:           "@example/server",
		Version:              "2.1.0",
		RuntimeArguments:     []mcpRegistryArgument{{Value: "--quiet"}},
		PackageArguments:     []mcpRegistryArgument{{Type: "named", Name: "--root", IsRequired: true}},
		EnvironmentVariables: []mcpRegistryVariable{{Name: "TOKEN", IsRequired: true}},
	})
	if config == nil || label != "npx" {
		t.Fatalf("packageMCPConfig() = (%#v, %q, %#v)", config, label, required)
	}
	wantArgs := []string{"-y", "--quiet", "@example/server@2.1.0", "--root="}
	if !reflect.DeepEqual(config.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", config.Args, wantArgs)
	}
	wantRequired := []string{"Argument: --root", "Environment: TOKEN"}
	if !reflect.DeepEqual(required, wantRequired) {
		t.Fatalf("required = %#v, want %#v", required, wantRequired)
	}
}

func TestMCPRegistryEntryUsesOfficialLatestMetadata(t *testing.T) {
	entry := mcpRegistryEntry{Meta: map[string]struct {
		IsLatest bool `json:"isLatest"`
	}{
		"publisher/metadata":                        {IsLatest: true},
		"io.modelcontextprotocol.registry/official": {IsLatest: false},
	}}
	if mcpRegistryEntryIsLatest(entry) {
		t.Fatal("mcpRegistryEntryIsLatest ignored official metadata")
	}
}

func TestGitHubRawFileURLPinsCommitAndRejectsTraversal(t *testing.T) {
	got, err := githubRawFileURL("anthropics/skills", "abc123", "skills/frontend-design", "SKILL.md")
	if err != nil {
		t.Fatalf("githubRawFileURL() error = %v", err)
	}
	want := "https://raw.githubusercontent.com/anthropics/skills/abc123/skills/frontend-design/SKILL.md"
	if got != want {
		t.Fatalf("githubRawFileURL() = %q, want %q", got, want)
	}
	if _, err := githubRawFileURL("anthropics/skills", "abc123", "skills/frontend-design", "../secret"); err == nil {
		t.Fatal("githubRawFileURL accepted a traversal path")
	}
}
