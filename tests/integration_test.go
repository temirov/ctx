package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles the binary from the module root and returns its path.
// It assumes tests are in a subdirectory so that the module root is the parent.
func buildBinary(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "content_integration")

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	moduleRoot := filepath.Dir(currentDir)
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v, output: %s", err, string(output))
	}
	return binaryPath
}

// runCommand executes the binary with the given arguments in workDir and returns its combined output.
func runCommand(t *testing.T, binary string, args []string, workDir string) string {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = workDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("Command %v failed: %v\nOutput: %s", args, err, stdout.String())
	}
	return stdout.String()
}

// TestTreeCommandIntegration_NoIgnore verifies that without .contentignore, tree command prints all entries.
func TestTreeCommandIntegration_NoIgnore(t *testing.T) {
	binary := buildBinary(t)
	tempDir := t.TempDir()

	// Setup: Create tempDir/file1.txt and tempDir/subdir/file2.txt.
	if err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("Hello"), 0644); err != nil {
		t.Fatalf("Failed to create file1.txt: %v", err)
	}
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("World"), 0644); err != nil {
		t.Fatalf("Failed to create file2.txt: %v", err)
	}

	output := runCommand(t, binary, []string{"tree", tempDir}, tempDir)
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "subdir") {
		t.Errorf("Tree output missing expected entries.\nOutput: %s", output)
	}
}

// TestContentCommandIntegration_NoIgnore verifies that without .contentignore, content command prints file contents.
func TestContentCommandIntegration_NoIgnore(t *testing.T) {
	binary := buildBinary(t)
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("Hello"), 0644); err != nil {
		t.Fatalf("Failed to create file1.txt: %v", err)
	}

	output := runCommand(t, binary, []string{"content", tempDir}, tempDir)
	if !strings.Contains(output, "Hello") {
		t.Errorf("Content output did not include expected content.\nOutput: %s", output)
	}
}

// TestTreeCommandIntegration_WithIgnore verifies that .contentignore excludes directories for tree.
func TestTreeCommandIntegration_WithIgnore(t *testing.T) {
	binary := buildBinary(t)
	tempDir := t.TempDir()

	// Setup: Create tempDir/log/ignored.txt and tempDir/data/included.txt.
	logDir := filepath.Join(tempDir, "log")
	if err := os.Mkdir(logDir, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "ignored.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatalf("Failed to create ignored.txt: %v", err)
	}
	dataDir := filepath.Join(tempDir, "data")
	if err := os.Mkdir(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "included.txt"), []byte("include me"), 0644); err != nil {
		t.Fatalf("Failed to create included.txt: %v", err)
	}

	// Write .contentignore to ignore "log/".
	ignoreFilePath := filepath.Join(tempDir, ".contentignore")
	if err := os.WriteFile(ignoreFilePath, []byte("log/\n"), 0644); err != nil {
		t.Fatalf("Failed to write .contentignore: %v", err)
	}

	output := runCommand(t, binary, []string{"tree", tempDir}, tempDir)
	if strings.Contains(output, "log") {
		t.Errorf("Tree output should not include 'log' directory.\nOutput: %s", output)
	}
	if !strings.Contains(output, "data") {
		t.Errorf("Tree output should include 'data' directory.\nOutput: %s", output)
	}
}

// TestContentCommandIntegration_WithIgnore verifies that .contentignore excludes files for content.
func TestContentCommandIntegration_WithIgnore(t *testing.T) {
	binary := buildBinary(t)
	tempDir := t.TempDir()

	// Setup: Create tempDir/log/ignored.txt and tempDir/data/included.txt.
	logDir := filepath.Join(tempDir, "log")
	if err := os.Mkdir(logDir, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "ignored.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatalf("Failed to create ignored.txt: %v", err)
	}
	dataDir := filepath.Join(tempDir, "data")
	if err := os.Mkdir(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "included.txt"), []byte("include me"), 0644); err != nil {
		t.Fatalf("Failed to create included.txt: %v", err)
	}

	// Write .contentignore to ignore "log/".
	ignoreFilePath := filepath.Join(tempDir, ".contentignore")
	if err := os.WriteFile(ignoreFilePath, []byte("log/\n"), 0644); err != nil {
		t.Fatalf("Failed to write .contentignore: %v", err)
	}

	output := runCommand(t, binary, []string{"content", tempDir}, tempDir)
	if strings.Contains(output, "ignore me") {
		t.Errorf("Content output should not include content from log/ignored.txt.\nOutput: %s", output)
	}
	if !strings.Contains(output, "include me") {
		t.Errorf("Content output should include content from data/included.txt.\nOutput: %s", output)
	}
}

// TestExclusionFlagIntegration verifies that when -e flag is applied, directories directly under the root are skipped.
func TestExclusionFlagIntegration(t *testing.T) {
	binary := buildBinary(t)
	tempDir := t.TempDir()

	// Setup: Create tempDir/pkg/ with pkg/include.txt and pkg/log/ignore.txt.
	pkgDir := filepath.Join(tempDir, "pkg")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create pkg directory: %v", err)
	}
	logDir := filepath.Join(pkgDir, "log")
	if err := os.Mkdir(logDir, 0755); err != nil {
		t.Fatalf("Failed to create pkg/log directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "ignore.txt"), []byte("should be ignored"), 0644); err != nil {
		t.Fatalf("Failed to create pkg/log/ignore.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "include.txt"), []byte("should be included"), 0644); err != nil {
		t.Fatalf("Failed to create pkg/include.txt: %v", err)
	}

	// Write an empty .contentignore in tempDir.
	ignoreFilePath := filepath.Join(tempDir, ".contentignore")
	if err := os.WriteFile(ignoreFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write .contentignore: %v", err)
	}

	// Run with -e flag "log" (applied relative to pkg).
	output := runCommand(t, binary, []string{"content", pkgDir, "-e", "log"}, tempDir)
	if strings.Contains(output, "should be ignored") {
		t.Errorf("Content output should not include content from pkg/log when -e flag is applied.\nOutput: %s", output)
	}
	if !strings.Contains(output, "should be included") {
		t.Errorf("Content output should include content from pkg/include.txt.\nOutput: %s", output)
	}
}

// TestExclusionFlagOnlyAtRoot verifies that -e flag excludes only directories directly under the working directory.
func TestExclusionFlagOnlyAtRoot(t *testing.T) {
	binary := buildBinary(t)
	tempDir := t.TempDir()

	// Setup: Create top-level log directory and nested pkg/log.
	topLogDir := filepath.Join(tempDir, "log")
	if err := os.Mkdir(topLogDir, 0755); err != nil {
		t.Fatalf("Failed to create top-level log directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topLogDir, "ignore.txt"), []byte("top log ignored"), 0644); err != nil {
		t.Fatalf("Failed to create top-level log file: %v", err)
	}

	pkgDir := filepath.Join(tempDir, "pkg")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create pkg directory: %v", err)
	}
	nestedLogDir := filepath.Join(pkgDir, "log")
	if err := os.Mkdir(nestedLogDir, 0755); err != nil {
		t.Fatalf("Failed to create nested log directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedLogDir, "nested_ignore.txt"), []byte("nested log not ignored"), 0644); err != nil {
		t.Fatalf("Failed to create nested log file: %v", err)
	}

	// Write an empty .contentignore in tempDir.
	ignoreFilePath := filepath.Join(tempDir, ".contentignore")
	if err := os.WriteFile(ignoreFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write .contentignore: %v", err)
	}

	// Run with -e flag "log".
	output := runCommand(t, binary, []string{"content", tempDir, "-e", "log"}, tempDir)
	if strings.Contains(output, "top log ignored") {
		t.Errorf("Top-level log folder should be excluded when -e flag is applied.\nOutput: %s", output)
	}
	if !strings.Contains(output, "nested log not ignored") {
		t.Errorf("Nested log folder should not be excluded when -e flag is applied.\nOutput: %s", output)
	}
}
