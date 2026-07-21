package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func configureExecutablePath() error {
	home, _ := os.UserHomeDir()
	path := mergeExecutablePath(os.Getenv("PATH"), defaultExecutablePaths(home))
	return os.Setenv("PATH", path)
}

func defaultExecutablePaths(home string) []string {
	paths := make([]string, 0, 10)
	if home != "" {
		paths = append(paths,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, "go", "bin"),
			filepath.Join(home, ".volta", "bin"),
			filepath.Join(home, ".npm-global", "bin"),
		)
	}

	switch runtime.GOOS {
	case "darwin":
		if home != "" {
			paths = append(paths, filepath.Join(home, "Library", "pnpm"))
		}
		paths = append(paths, "/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin")
	case "linux":
		paths = append(paths, "/home/linuxbrew/.linuxbrew/bin", "/usr/local/bin", "/snap/bin")
	}
	return paths
}

func mergeExecutablePath(current string, extras []string) string {
	paths := filepath.SplitList(current)
	seen := make(map[string]struct{}, len(paths)+len(extras))
	for _, path := range paths {
		if key := executablePathKey(path); key != "" {
			seen[key] = struct{}{}
		}
	}
	for _, path := range extras {
		key := executablePathKey(path)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		paths = append(paths, path)
		seen[key] = struct{}{}
	}
	return strings.Join(paths, string(os.PathListSeparator))
}

func executablePathKey(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}
