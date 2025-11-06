package tokenizer

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	anthropicScriptName = "anthropic_count.py"
	llamaScriptName     = "llama_count.py"
)

//go:embed helpers/*.py
var embeddedHelperScripts embed.FS

func materializeHelperScripts(targetDir string) (string, error) {
	if targetDir != "" {
		return targetDir, nil
	}

	tempDir, createErr := os.MkdirTemp("", "ctx-token-helpers-*")
	if createErr != nil {
		return "", fmt.Errorf("create helper dir: %w", createErr)
	}

	entries := []string{anthropicScriptName, llamaScriptName}
	for _, scriptName := range entries {
		content, readErr := fs.ReadFile(embeddedHelperScripts, filepath.Join("helpers", scriptName))
		if readErr != nil {
			return "", fmt.Errorf("%w: read embedded helper %s: %w", ErrHelperUnavailable, scriptName, readErr)
		}
		scriptPath := filepath.Join(tempDir, scriptName)
		if writeErr := os.WriteFile(scriptPath, content, 0o700); writeErr != nil {
			return "", fmt.Errorf("%w: write helper %s: %w", ErrHelperUnavailable, scriptName, writeErr)
		}
	}

	return tempDir, nil
}
