package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadIgnoreFilePatternsLegacy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".ignore")
	content := "# comment\nfoo\nbar\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write ignore file: %v", err)
	}
	patterns, err := LoadIgnoreFilePatterns(path)
	if err != nil {
		t.Fatalf("LoadIgnoreFilePatterns returned error: %v", err)
	}
	expectedIgnore := []string{"foo", "bar"}
	if !reflect.DeepEqual(patterns.Ignore, expectedIgnore) {
		t.Fatalf("unexpected ignore patterns: %#v", patterns.Ignore)
	}
	if len(patterns.Binary) != 0 {
		t.Fatalf("expected no binary patterns, got %#v", patterns.Binary)
	}
}

func TestLoadIgnoreFilePatternsWithSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".ignore")
	content := "pre\n[binary]\nbin1\n[ignore]\nig1\n[binary]\nbin2\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write ignore file: %v", err)
	}
	patterns, err := LoadIgnoreFilePatterns(path)
	if err != nil {
		t.Fatalf("LoadIgnoreFilePatterns returned error: %v", err)
	}
	expectedIgnore := []string{"pre", "ig1"}
	expectedBinary := []string{"bin1", "bin2"}
	if !reflect.DeepEqual(patterns.Ignore, expectedIgnore) {
		t.Fatalf("unexpected ignore patterns: %#v", patterns.Ignore)
	}
	if !reflect.DeepEqual(patterns.Binary, expectedBinary) {
		t.Fatalf("unexpected binary patterns: %#v", patterns.Binary)
	}
}
