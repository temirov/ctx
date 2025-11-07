package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tyemirov/ctx/internal/utils"
)

func TestDetectMimeType(t *testing.T) {
	tempDir := t.TempDir()
	textPath := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(textPath, []byte("plain text"), 0o600); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	mimeType := utils.DetectMimeType(textPath)
	if mimeType != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain mime type, got %q", mimeType)
	}

	missingPath := filepath.Join(tempDir, "missing.txt")
	if result := utils.DetectMimeType(missingPath); result != utils.UnknownMimeType {
		t.Fatalf("expected unknown mime type for missing file, got %q", result)
	}
}
