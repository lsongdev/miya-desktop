package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed defaults/AGENTS.md skills/registry.json skills/miya-desktop/SKILL.md
var desktopDefaults embed.FS

func (a *App) InitializeDefaultWorkspace() error {
	workspace := a.ensureDefaultWorkspace()
	if err := installEmbeddedFileIfMissing("defaults/AGENTS.md", filepath.Join(workspace, "AGENTS.md")); err != nil {
		return fmt.Errorf("initialize workspace instructions: %w", err)
	}
	if err := a.ensureBundledDesktopSkill(); err != nil {
		return fmt.Errorf("initialize desktop skill: %w", err)
	}
	return nil
}

func (a *App) ensureBundledDesktopSkill() error {
	return installEmbeddedFileIfMissing(
		"skills/miya-desktop/SKILL.md",
		filepath.Join(a.SkillsDirectory(), "miya-desktop", "SKILL.md"),
	)
}

func installEmbeddedFileIfMissing(source, destination string) error {
	if _, err := os.Stat(destination); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := fs.ReadFile(desktopDefaults, source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0644)
}
