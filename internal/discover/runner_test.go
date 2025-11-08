package discover

import (
	"path/filepath"
	"testing"
)

func TestNewRunnerDefaultsOutputDirToDocsDependencies(t *testing.T) {
	rootDirectory := t.TempDir()

	runner := NewRunner(Options{RootPath: rootDirectory})

	expected := filepath.Join(rootDirectory, "docs", "dependencies")
	if runner.options.OutputDir != expected {
		t.Fatalf("expected output dir %s, got %s", expected, runner.options.OutputDir)
	}
}
