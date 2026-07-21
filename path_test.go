package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMergeExecutablePathPreservesExistingOrderAndAddsMissingPaths(t *testing.T) {
	current := filepath.Join("existing", "bin") + string(os.PathListSeparator) + filepath.Join("other", "bin")
	extra := filepath.Join("extra", "bin")

	got := filepath.SplitList(mergeExecutablePath(current, []string{extra}))
	want := []string{filepath.Join("existing", "bin"), filepath.Join("other", "bin"), extra}
	if !slices.Equal(got, want) {
		t.Fatalf("mergeExecutablePath() = %#v, want %#v", got, want)
	}
}

func TestMergeExecutablePathSkipsEmptyAndDuplicatePaths(t *testing.T) {
	existing := filepath.Join("existing", "bin")
	current := existing + string(os.PathListSeparator) + filepath.Join("other", "bin")

	got := filepath.SplitList(mergeExecutablePath(current, []string{"", existing, filepath.Join("existing", ".", "bin")}))
	want := filepath.SplitList(current)
	if !slices.Equal(got, want) {
		t.Fatalf("mergeExecutablePath() = %#v, want %#v", got, want)
	}
}

func TestDefaultExecutablePathsIncludeLocalInstallLocations(t *testing.T) {
	home := filepath.Join("users", "miya")
	paths := defaultExecutablePaths(home)

	for _, want := range []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, "go", "bin"),
	} {
		if !slices.Contains(paths, want) {
			t.Errorf("defaultExecutablePaths() missing %q", want)
		}
	}
}
