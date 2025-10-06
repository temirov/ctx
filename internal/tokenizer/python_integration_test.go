//go:build python_helpers

package tokenizer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func pythonExecutable(t *testing.T) string {
	python := os.Getenv("CTX_TEST_PYTHON")
	if python == "" {
		python = "python3"
	}
	cmd := exec.Command(python, "-c", "import sys")
	if err := cmd.Run(); err != nil {
		t.Skipf("python executable %q unavailable: %v", python, err)
	}
	return python
}

func ensurePythonModule(t *testing.T, python string, module string) {
	cmd := exec.Command(python, "-c", "import "+module)
	if err := cmd.Run(); err != nil {
		t.Skipf("python module %s not installed: %v", module, err)
	}
}

func TestPythonHelperAnthropic(t *testing.T) {
	python := pythonExecutable(t)
	ensurePythonModule(t, python, "anthropic_tokenizer")

	counter, model, err := NewCounter(Config{Model: "claude-3-5-sonnet", PythonExecutable: python})
	if err != nil {
		t.Fatalf("NewCounter error: %v", err)
	}
	if model != "claude-3-5-sonnet" {
		t.Fatalf("expected resolved model claude-3-5-sonnet, got %q", model)
	}
	tokens, err := counter.CountString("Integration tests for anthropic helper")
	if err != nil {
		t.Fatalf("CountString error: %v", err)
	}
	if tokens <= 0 {
		t.Fatalf("expected anthropic helper token output > 0, got %d", tokens)
	}
}

func TestPythonHelperLlama(t *testing.T) {
	python := pythonExecutable(t)
	ensurePythonModule(t, python, "sentencepiece")

	spModelPath := os.Getenv("CTX_TEST_SPM_MODEL")
	if spModelPath == "" {
		t.Skip("CTX_TEST_SPM_MODEL not set; skipping llama helper test")
	}
	if _, err := os.Stat(spModelPath); err != nil {
		t.Skipf("sentencepiece model unavailable: %v", err)
	}

	cfg := Config{
		Model:                  "llama-3.1-8b",
		PythonExecutable:       python,
		SentencePieceModelPath: spModelPath,
	}
	counter, model, err := NewCounter(cfg)
	if err != nil {
		t.Fatalf("NewCounter error: %v", err)
	}
	if model != "llama-3.1-8b" {
		t.Fatalf("expected resolved model llama-3.1-8b, got %q", model)
	}

	tempDir := t.TempDir()
	samplePath := filepath.Join(tempDir, "sample.txt")
	sampleContent := "sentencepiece integration test"
	if writeErr := os.WriteFile(samplePath, []byte(sampleContent), 0o600); writeErr != nil {
		t.Fatalf("writing sample file: %v", writeErr)
	}

	tokens, err := counter.CountString(sampleContent)
	if err != nil {
		t.Fatalf("CountString error: %v", err)
	}
	if tokens <= 0 {
		t.Fatalf("expected sentencepiece helper token output > 0, got %d", tokens)
	}
}
