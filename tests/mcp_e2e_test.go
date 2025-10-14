package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const ctxBinaryEnvVar = "CTX_TEST_BINARY"

func TestMCPEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MCP e2e test in short mode")
	}

	ctxBinary := resolveCtxBinary(t)
	projectDir := seedMCPProject(t)

	cmdCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	command := exec.CommandContext(cmdCtx, ctxBinary, "--mcp")
	command.Dir = projectDir
	stdoutPipe, stdoutErr := command.StdoutPipe()
	if stdoutErr != nil {
		t.Fatalf("stdout pipe: %v", stdoutErr)
	}
	command.Stderr = os.Stderr

	if err := command.Start(); err != nil {
		t.Fatalf("start ctx --mcp: %v", err)
	}

	address := waitForMCPStartup(t, stdoutPipe)

	assertEnvironmentEndpoint(t, address, projectDir)
	assertCapabilitiesPayload(t, address)
	assertTreeCommandOutput(t, address, projectDir)
	assertContentCommandOutput(t, address, projectDir)

	cancel()
	if err := command.Wait(); err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "signal") {
		t.Fatalf("ctx --mcp wait: %v", err)
	}
}

func resolveCtxBinary(t *testing.T) string {
	t.Helper()
	if custom := os.Getenv(ctxBinaryEnvVar); custom != "" {
		return custom
	}
	return buildBinary(t)
}

func seedMCPProject(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()

	goFile := filepath.Join(projectDir, "main.go")
	goContent := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}

func greet() string {
	return fmt.Sprintf("hi %s", "ctx")
}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0o600); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	jsFile := filepath.Join(projectDir, "module.js")
	jsContent := `/**
 * Returns the greeting text.
 */
export function greet(name) {
	return \\"hello \\\" + name;
}
`
	if err := os.WriteFile(jsFile, []byte(jsContent), 0o600); err != nil {
		t.Fatalf("write js file: %v", err)
	}

	pyFile := filepath.Join(projectDir, "module.py")
	pyContent := `"""Sample module docstring."""

def greet(name):
	"""Return greeting text."""
	return f"hello {name}"
`
	if err := os.WriteFile(pyFile, []byte(pyContent), 0o600); err != nil {
		t.Fatalf("write python file: %v", err)
	}

	goMod := filepath.Join(projectDir, "go.mod")
	goModContent := "module example.com/project\ngo 1.21\n"
	if err := os.WriteFile(goMod, []byte(goModContent), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	return projectDir
}

func waitForMCPStartup(t *testing.T, reader io.Reader) string {
	t.Helper()
	lineCh := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
		close(lineCh)
	}()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for MCP startup message")
		case err := <-errCh:
			if err != nil {
				t.Fatalf("read MCP startup output: %v", err)
			}
			t.Fatalf("MCP startup stream closed before reporting address")
		case line, ok := <-lineCh:
			if !ok {
				t.Fatalf("MCP startup stream closed before reporting address")
			}
			if strings.HasPrefix(line, "MCP server listening on ") {
				return strings.TrimPrefix(line, "MCP server listening on ")
			}
		}
	}
}

func assertEnvironmentEndpoint(t *testing.T, address, expectedRoot string) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	request, err := http.NewRequest(http.MethodGet, "http://"+address+"/environment", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("environment request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("environment status: %d", response.StatusCode)
	}
	var body struct {
		RootDirectory string `json:"rootDirectory"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode environment: %v", err)
	}
	if body.RootDirectory != expectedRoot {
		t.Fatalf("unexpected root directory: %s", body.RootDirectory)
	}
}

func assertCapabilitiesPayload(t *testing.T, address string) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	request, err := http.NewRequest(http.MethodGet, "http://"+address+"/capabilities", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("capabilities request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("capabilities status: %d", response.StatusCode)
	}
	var payload struct {
		Capabilities []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"capabilities"`
		RootDirectory string `json:"rootDirectory"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if len(payload.Capabilities) != 3 {
		t.Fatalf("expected 3 capabilities, got %d", len(payload.Capabilities))
	}
	lookup := map[string]string{}
	for _, capability := range payload.Capabilities {
		lookup[capability.Name] = capability.Description
	}
	expected := map[string]string{
		"tree":      "Display directory tree as JSON.",
		"content":   "Show file contents as JSON.",
		"callchain": "Analyze Go/Python/JavaScript call chains as JSON.",
	}
	for name, prefix := range expected {
		description, ok := lookup[name]
		if !ok {
			t.Fatalf("missing capability %s", name)
		}
		if !strings.HasPrefix(description, prefix) {
			t.Fatalf("capability %s description mismatch: %s", name, description)
		}
	}
}

func assertTreeCommandOutput(t *testing.T, address, projectDir string) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	escaped := strings.ReplaceAll(projectDir, `"`, `\"`)
	payload := bytes.NewBufferString(`{"paths":["` + escaped + `"]}`)
	request, err := http.NewRequest(http.MethodPost, "http://"+address+"/commands/tree", payload)
	if err != nil {
		t.Fatalf("tree request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("tree command: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("tree command status: %d", response.StatusCode)
	}
	var body struct {
		Output string `json:"output"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode tree output: %v", err)
	}
	if !strings.Contains(body.Output, projectDir) {
		t.Fatalf("tree output missing project path: %s", body.Output)
	}
}

func assertContentCommandOutput(t *testing.T, address, projectDir string) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	escaped := strings.ReplaceAll(projectDir, `"`, `\"`)
	payload := bytes.NewBufferString(`{"paths":["` + escaped + `"],"documentation":true}`)
	request, err := http.NewRequest(http.MethodPost, "http://"+address+"/commands/content", payload)
	if err != nil {
		t.Fatalf("content request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("content command: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("content command status: %d", response.StatusCode)
	}
	var body struct {
		Output string `json:"output"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode content output: %v", err)
	}
	if !strings.Contains(body.Output, "main.go") || !strings.Contains(body.Output, "module.js") || !strings.Contains(body.Output, "module.py") {
		t.Fatalf("content output missing expected files: %s", body.Output)
	}
}
