package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentDocsAttemptFetchesRemoteDocumentation(t *testing.T) {
	if testing.Short() {
		t.Skip("skip docs attempt test in short mode")
	}

	binary := buildBinary(t)

	server := startGitHubMockServer(t, "remotedocs", "lib", "v1.2.3", "docs", map[string]string{
		"docs/overview.md": "# Overview\n\nRemote library overview.",
	})
	defer server.Close()

	workingDirectory := t.TempDir()
	writeFile(t, filepath.Join(workingDirectory, "go.mod"), `module example.com/app

go 1.21

require github.com/remotedocs/lib v1.2.3
`)
	writeFile(t, filepath.Join(workingDirectory, "main.go"), `package main

import "github.com/remotedocs/lib/pkg"

func use() {
	_ = pkg.Name
}
`)

	output := runCommand(t, binary, []string{
		"content",
		"--doc=full",
		"--docs-attempt",
		"--docs-api-base", server.URL,
		"--format", "json",
		".",
	}, workingDirectory)

	if !strings.Contains(output, "Remote library overview.") {
		t.Fatalf("expected remote documentation in output:\n%s", output)
	}

	if strings.Count(output, "Remote library overview.") != 1 {
		t.Fatalf("expected a single remote documentation entry\n%s", output)
	}
	if !strings.Contains(output, "\"type\": \"module\"") {
		t.Fatalf("expected module documentation kind in output\n%s", output)
	}
	if !strings.Contains(output, "github.com/remotedocs/lib/overview.md") {
		t.Fatalf("expected remote documentation path in output\n%s", output)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
