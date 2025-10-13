package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitializeConfigurationCreatesLocalFile(t *testing.T) {
	workingDirectory := t.TempDir()
	options := InitOptions{WorkingDirectory: workingDirectory, Target: InitTargetLocal}
	path, err := InitializeConfiguration(options)
	if err != nil {
		t.Fatalf("InitializeConfiguration error: %v", err)
	}
	expectedPath := filepath.Join(workingDirectory, "config.yaml")
	if path != expectedPath {
		t.Fatalf("expected path %s, got %s", expectedPath, path)
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if !strings.Contains(string(content), "tree:") {
		t.Fatalf("unexpected configuration content: %s", string(content))
	}
}

func TestInitializeConfigurationHonorsGlobalTarget(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	path, err := InitializeConfiguration(InitOptions{Target: InitTargetGlobal, Force: true})
	if err != nil {
		t.Fatalf("InitializeConfiguration error: %v", err)
	}
	if !strings.HasPrefix(path, homeDir) {
		t.Fatalf("expected configuration under home dir, got %s", path)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected file to exist at %s: %v", path, statErr)
	}
}

func TestInitializeConfigurationPreventsOverwriteWithoutForce(t *testing.T) {
	workingDirectory := t.TempDir()
	path := filepath.Join(workingDirectory, "config.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("write seed config: %v", err)
	}
	_, err := InitializeConfiguration(InitOptions{WorkingDirectory: workingDirectory, Target: InitTargetLocal, Force: false})
	if err == nil {
		t.Fatalf("expected error when configuration already exists")
	}
}
