//go:build python_helpers

package tokenizer

import (
	"os"
	"os/exec"
	"testing"
)

func requireHelperExecution(t *testing.T) {
	if os.Getenv("CTX_TEST_RUN_HELPERS") != "1" {
		t.Skip("set CTX_TEST_RUN_HELPERS=1 to run uv helper integration tests")
	}
}

func uvExecutable(t *testing.T) string {
	candidate := os.Getenv("CTX_TEST_UV")
	if candidate == "" {
		candidate = "uv"
	}
	path, err := exec.LookPath(candidate)
	if err != nil {
		t.Skipf("uv executable %q unavailable: %v", candidate, err)
	}
	return path
}

func TestPythonHelperAnthropic(t *testing.T) {
	requireHelperExecution(t)
	uv := uvExecutable(t)
	t.Setenv("CTX_UV", uv)

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
	requireHelperExecution(t)
	uv := uvExecutable(t)
	t.Setenv("CTX_UV", uv)
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
