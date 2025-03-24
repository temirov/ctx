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
//
// #nosec G204: we intentionally invoke "go build" with variable arguments
func buildBinary(testValue *testing.T) string {
	testValue.Helper()
	temporaryDirectory := testValue.TempDir()
	binaryPath := filepath.Join(temporaryDirectory, "content_integration")

	currentDirectory, directoryError := os.Getwd()
	if directoryError != nil {
		testValue.Fatalf("Failed to get working directory: %v", directoryError)
	}
	moduleRoot := filepath.Dir(currentDirectory)

	buildCommand := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCommand.Dir = moduleRoot

	outputData, buildErr := buildCommand.CombinedOutput()
	if buildErr != nil {
		testValue.Fatalf("Failed to build binary: %v, output: %s", buildErr, string(outputData))
	}
	return binaryPath
}

// runCommand executes the binary with the given args in workDir and returns combined stdout+stderr.
func runCommand(testValue *testing.T, binary string, args []string, workDir string) string {
	testValue.Helper()
	command := exec.Command(binary, args...)
	command.Dir = workDir

	var stdoutBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stdoutBuffer

	runError := command.Run()
	if runError != nil {
		testValue.Fatalf("Command %v failed: %v\nOutput: %s", args, runError, stdoutBuffer.String())
	}
	return stdoutBuffer.String()
}

// TestTreeCommandIntegration_NoIgnore verifies that without .ignore, tree prints all entries.
func TestTreeCommandIntegration_NoIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	file1Path := filepath.Join(tempDir, "file1.txt")
	writeError := os.WriteFile(file1Path, []byte("Hello"), 0o600) // G306 fix
	if writeError != nil {
		testValue.Fatalf("Failed to create file1.txt: %v", writeError)
	}

	subDirectory := filepath.Join(tempDir, "subdir")
	makeError := os.Mkdir(subDirectory, 0o750) // G301 fix
	if makeError != nil {
		testValue.Fatalf("Failed to create subdir: %v", makeError)
	}

	file2Path := filepath.Join(subDirectory, "file2.txt")
	writeError = os.WriteFile(file2Path, []byte("World"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create file2.txt: %v", writeError)
	}

	output := runCommand(testValue, binary, []string{"tree", tempDir}, tempDir)
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "subdir") {
		testValue.Errorf("Tree output missing expected entries.\nOutput: %s", output)
	}
}

// TestContentCommandIntegration_NoIgnore verifies that without .ignore, content prints file contents.
func TestContentCommandIntegration_NoIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	file1Path := filepath.Join(tempDir, "file1.txt")
	writeError := os.WriteFile(file1Path, []byte("Hello"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create file1.txt: %v", writeError)
	}

	output := runCommand(testValue, binary, []string{"content", tempDir}, tempDir)
	if !strings.Contains(output, "Hello") {
		testValue.Errorf("Content output did not include expected content.\nOutput: %s", output)
	}
}

//nolint:dupl
func TestTreeCommandIntegration_WithIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	logDir := filepath.Join(tempDir, "log")
	makeError := os.Mkdir(logDir, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create log directory: %v", makeError)
	}
	ignoredFilePath := filepath.Join(logDir, "ignored.txt")
	writeError := os.WriteFile(ignoredFilePath, []byte("ignore me"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create ignored.txt: %v", writeError)
	}

	dataDir := filepath.Join(tempDir, "data")
	makeError = os.Mkdir(dataDir, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create data directory: %v", makeError)
	}
	includedFilePath := filepath.Join(dataDir, "included.txt")
	writeError = os.WriteFile(includedFilePath, []byte("include me"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create included.txt: %v", writeError)
	}

	ignoreFilePath := filepath.Join(tempDir, ".ignore")
	writeError = os.WriteFile(ignoreFilePath, []byte("log/\n"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to write .ignore: %v", writeError)
	}

	output := runCommand(testValue, binary, []string{"tree", tempDir}, tempDir)
	if strings.Contains(output, "log") {
		testValue.Errorf("Tree output should not include 'log' directory.\nOutput: %s", output)
	}
	if !strings.Contains(output, "data") {
		testValue.Errorf("Tree output should include 'data' directory.\nOutput: %s", output)
	}
}

//nolint:dupl
func TestContentCommandIntegration_WithIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	logDir := filepath.Join(tempDir, "log")
	makeError := os.Mkdir(logDir, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create log directory: %v", makeError)
	}
	ignoredFilePath := filepath.Join(logDir, "ignored.txt")
	writeError := os.WriteFile(ignoredFilePath, []byte("ignore me"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create ignored.txt: %v", writeError)
	}

	dataDir := filepath.Join(tempDir, "data")
	makeError = os.Mkdir(dataDir, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create data directory: %v", makeError)
	}
	includedFilePath := filepath.Join(dataDir, "included.txt")
	writeError = os.WriteFile(includedFilePath, []byte("include me"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create included.txt: %v", writeError)
	}

	ignoreFilePath := filepath.Join(tempDir, ".ignore")
	writeError = os.WriteFile(ignoreFilePath, []byte("log/\n"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to write .ignore: %v", writeError)
	}

	output := runCommand(testValue, binary, []string{"content", tempDir}, tempDir)
	if strings.Contains(output, "ignore me") {
		testValue.Errorf("Content output should not include content from log/ignored.txt.\nOutput: %s", output)
	}
	if !strings.Contains(output, "include me") {
		testValue.Errorf("Content output should include content from data/included.txt.\nOutput: %s", output)
	}
}

func TestExclusionFlagIntegration(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	pkgDirectory := filepath.Join(tempDir, "pkg")
	makeError := os.Mkdir(pkgDirectory, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create pkg directory: %v", makeError)
	}
	logDirectory := filepath.Join(pkgDirectory, "log")
	makeError = os.Mkdir(logDirectory, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create pkg/log directory: %v", makeError)
	}
	ignoredFilePath := filepath.Join(logDirectory, "ignore.txt")
	writeError := os.WriteFile(ignoredFilePath, []byte("should be ignored"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create pkg/log/ignore.txt: %v", writeError)
	}
	includedFilePath := filepath.Join(pkgDirectory, "include.txt")
	writeError = os.WriteFile(includedFilePath, []byte("should be included"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create pkg/include.txt: %v", writeError)
	}

	ignoreFilePath := filepath.Join(tempDir, ".ignore")
	writeError = os.WriteFile(ignoreFilePath, []byte(""), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to write .ignore: %v", writeError)
	}

	output := runCommand(testValue, binary, []string{"content", pkgDirectory, "-e", "log"}, tempDir)
	if strings.Contains(output, "should be ignored") {
		testValue.Errorf("Content output should not include content from pkg/log when -e is set.\nOutput: %s", output)
	}
	if !strings.Contains(output, "should be included") {
		testValue.Errorf("Content output should include content from pkg/include.txt.\nOutput: %s", output)
	}
}

func TestExclusionFlagRootVsNested(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	topLogDirectory := filepath.Join(tempDir, "log")
	makeError := os.Mkdir(topLogDirectory, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create top-level log directory: %v", makeError)
	}
	topLogFile := filepath.Join(topLogDirectory, "ignore.txt")
	writeError := os.WriteFile(topLogFile, []byte("top log ignored"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create top-level log file: %v", writeError)
	}

	pkgDirectory := filepath.Join(tempDir, "pkg")
	makeError = os.Mkdir(pkgDirectory, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create pkg directory: %v", makeError)
	}
	nestedLogDirectory := filepath.Join(pkgDirectory, "log")
	makeError = os.Mkdir(nestedLogDirectory, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create nested log directory: %v", makeError)
	}
	nestedLogFile := filepath.Join(nestedLogDirectory, "nested_ignore.txt")
	writeError = os.WriteFile(nestedLogFile, []byte("nested log not ignored"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create nested log file: %v", writeError)
	}

	ignoreFilePath := filepath.Join(tempDir, ".ignore")
	writeError = os.WriteFile(ignoreFilePath, []byte(""), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to write .ignore: %v", writeError)
	}

	output := runCommand(testValue, binary, []string{"content", tempDir, "-e", "log"}, tempDir)
	if strings.Contains(output, "top log ignored") {
		testValue.Errorf("Top-level log folder should be excluded when -e=log.\nOutput: %s", output)
	}
	if !strings.Contains(output, "nested log not ignored") {
		testValue.Errorf("Nested log folder should not be excluded when -e=log.\nOutput: %s", output)
	}
}

func TestIgnoreAllPemFilesGlobally(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	topPemPath := filepath.Join(tempDir, "cert.pem")
	writeError := os.WriteFile(topPemPath, []byte("top-level PEM"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create top-level cert.pem: %v", writeError)
	}

	nestedDirectory := filepath.Join(tempDir, "nested")
	makeError := os.Mkdir(nestedDirectory, 0o750)
	if makeError != nil {
		testValue.Fatalf("Failed to create nested directory: %v", makeError)
	}
	nestedPemPath := filepath.Join(nestedDirectory, "key.pem")
	writeError = os.WriteFile(nestedPemPath, []byte("nested PEM"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create nested key.pem: %v", writeError)
	}

	includedPath := filepath.Join(tempDir, "keep.txt")
	writeError = os.WriteFile(includedPath, []byte("I am not a PEM"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to create keep.txt: %v", writeError)
	}

	ignoreFilePath := filepath.Join(tempDir, ".ignore")
	writeError = os.WriteFile(ignoreFilePath, []byte("*.pem\n"), 0o600)
	if writeError != nil {
		testValue.Fatalf("Failed to write .ignore: %v", writeError)
	}

	output := runCommand(testValue, binary, []string{"content", tempDir}, tempDir)
	if strings.Contains(output, "top-level PEM") {
		testValue.Errorf("Should have excluded top-level .pem file.\nOutput: %s", output)
	}
	if strings.Contains(output, "nested PEM") {
		testValue.Errorf("Should have excluded nested .pem file.\nOutput: %s", output)
	}
	if !strings.Contains(output, "I am not a PEM") {
		testValue.Errorf("Should have included keep.txt.\nOutput: %s", output)
	}
}

func TestExclusionFlagTrailingSlash(testValue *testing.T) {
	binary := buildBinary(testValue)
	tempDir := testValue.TempDir()

	// Create top-level directory "memory-bank" with a file inside.
	memoryBankDir := filepath.Join(tempDir, "memory-bank")
	if err := os.Mkdir(memoryBankDir, 0750); err != nil {
		testValue.Fatalf("Failed to create memory-bank directory: %v", err)
	}
	fileInMemoryBank := filepath.Join(memoryBankDir, "file.txt")
	if err := os.WriteFile(fileInMemoryBank, []byte("should be excluded"), 0600); err != nil {
		testValue.Fatalf("Failed to create file in memory-bank: %v", err)
	}

	// Create a file outside "memory-bank" that should always be included.
	fileOutside := filepath.Join(tempDir, "outside.txt")
	if err := os.WriteFile(fileOutside, []byte("should be included"), 0600); err != nil {
		testValue.Fatalf("Failed to create file outside memory-bank: %v", err)
	}

	// Test with exclusion flag without trailing slash.
	outputNoSlash := runCommand(testValue, binary, []string{"content", tempDir, "-e", "memory-bank"}, tempDir)
	if strings.Contains(outputNoSlash, "should be excluded") {
		testValue.Errorf("Content output should not include content from memory-bank when exclusion flag is 'memory-bank'.\nOutput: %s", outputNoSlash)
	}
	if !strings.Contains(outputNoSlash, "should be included") {
		testValue.Errorf("Content output should include content outside memory-bank when exclusion flag is 'memory-bank'.\nOutput: %s", outputNoSlash)
	}

	// Test with exclusion flag with trailing slash.
	outputWithSlash := runCommand(testValue, binary, []string{"content", tempDir, "-e", "memory-bank/"}, tempDir)
	if strings.Contains(outputWithSlash, "should be excluded") {
		testValue.Errorf("Content output should not include content from memory-bank when exclusion flag is 'memory-bank/'.\nOutput: %s", outputWithSlash)
	}
	if !strings.Contains(outputWithSlash, "should be included") {
		testValue.Errorf("Content output should include content outside memory-bank when exclusion flag is 'memory-bank/'.\nOutput: %s", outputWithSlash)
	}
}
