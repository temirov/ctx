//go:build python_helpers

package tokenizer

import (
	"os"
	"os/exec"
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

func TestPythonHelperAnthropic(t *testing.T) {
	python := pythonExecutable(t)
	if err := ensurePythonModule(python, "anthropic_tokenizer"); err != nil {
		t.Skipf("python module anthropic_tokenizer not available: %v", err)
	}

	t.Setenv("CTX_PYTHON", python)
	counter, model, err := NewCounter(Config{Model: "claude-3-5-sonnet"})
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
	if err := ensurePythonModule(python, "sentencepiece"); err != nil {
		t.Skipf("python module sentencepiece not available: %v", err)
	}

	spModelPath := os.Getenv("CTX_TEST_SPM_MODEL")
	if spModelPath == "" {
		t.Skip("CTX_TEST_SPM_MODEL not set; skipping llama helper test")
	}
	if _, err := os.Stat(spModelPath); err != nil {
		t.Skipf("sentencepiece model unavailable: %v", err)
	}

	t.Setenv("CTX_PYTHON", python)
	t.Setenv("CTX_SPM_MODEL", spModelPath)
	counter, model, err := NewCounter(Config{Model: "llama-3.1-8b"})
	if err != nil {
		t.Fatalf("NewCounter error: %v", err)
	}
	if model != "llama-3.1-8b" {
		t.Fatalf("expected resolved model llama-3.1-8b, got %q", model)
	}

	tokens, err := counter.CountString("sentencepiece integration test")
	if err != nil {
		t.Fatalf("CountString error: %v", err)
	}
	if tokens <= 0 {
		t.Fatalf("expected sentencepiece helper token output > 0, got %d", tokens)
	}
}
