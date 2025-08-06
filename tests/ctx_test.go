// Package tests contains the integration‑level test‑suite for ctx.
package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	appTypes "github.com/temirov/ctx/types"
)

func buildBinary(testingHandle *testing.T) string {
	testingHandle.Helper()

	temporaryDirectory := testingHandle.TempDir()
	binaryName := "ctx_integration_binary"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(temporaryDirectory, binaryName)

	currentDirectory, directoryError := os.Getwd()
	if directoryError != nil {
		testingHandle.Fatalf("failed to determine current working directory: %v", directoryError)
	}

	moduleRoot := filepath.Dir(currentDirectory)
	buildCommand := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCommand.Dir = moduleRoot

	combinedOutput, buildError := buildCommand.CombinedOutput()
	if buildError != nil {
		testingHandle.Fatalf("build failed in %s: %v\n%s", moduleRoot, buildError, string(combinedOutput))
	}

	return binaryPath
}

func runCommand(testingHandle *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testingHandle.Helper()

	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory

	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer

	runError := command.Run()

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	if runError != nil {
		if exitError, ok := runError.(*exec.ExitError); ok {
			testingHandle.Fatalf("command failed (%d): %v\nstdout:\n%s\nstderr:\n%s",
				exitError.ExitCode(), runError, stdout, stderr)
		}
		testingHandle.Fatalf("command failed: %v\nstdout:\n%s\nstderr:\n%s", runError, stdout, stderr)
	}

	if strings.Contains(stderr, "Warning:") {
		testingHandle.Logf("command produced warnings:\n%s", stderr)
	}

	return stdout
}

func runCommandExpectError(testingHandle *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testingHandle.Helper()

	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory

	var buffer bytes.Buffer
	command.Stdout = &buffer
	command.Stderr = &buffer

	runError := command.Run()
	output := buffer.String()

	if runError == nil {
		testingHandle.Fatalf("command succeeded unexpectedly\noutput:\n%s", output)
	}

	return output
}

func runCommandWithWarnings(testingHandle *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testingHandle.Helper()

	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory

	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer

	runError := command.Run()

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	if runError != nil {
		var exitError *exec.ExitError
		if errors.As(runError, &exitError) {
			testingHandle.Fatalf("command failed when warnings were expected (%d): %v\nstderr:\n%s",
				exitError.ExitCode(), runError, stderr)
		}
		testingHandle.Fatalf("command failed when warnings were expected: %v\nstderr:\n%s", runError, stderr)
	}

	if !strings.Contains(stderr, "Warning:") {
		testingHandle.Fatalf("expected warnings on stderr\nstderr:\n%s", stderr)
	}

	return stdout
}

func setupTestDirectory(testingHandle *testing.T, layout map[string]string) string {
	testingHandle.Helper()

	root := testingHandle.TempDir()

	for relativePath, content := range layout {
		absolutePath := filepath.Join(root, relativePath)

		if strings.HasSuffix(relativePath, "/") || content == "" {
			_ = os.MkdirAll(absolutePath, 0o755)
			continue
		}

		parent := filepath.Dir(absolutePath)
		_ = os.MkdirAll(parent, 0o755)

		if content == "<UNREADABLE>" {
			_ = os.WriteFile(absolutePath, []byte("unreadable placeholder"), 0o644)
			_ = os.Chmod(absolutePath, 0o000)
			continue
		}

		_ = os.WriteFile(absolutePath, []byte(content), 0o644)
	}

	return root
}

func getModuleRoot(testingHandle *testing.T) string {
	testingHandle.Helper()

	directory, err := os.Getwd()
	if err != nil {
		testingHandle.Fatalf("failed to determine working directory: %v", err)
	}

	for {
		goMod := filepath.Join(directory, "go.mod")
		if _, statErr := os.Stat(goMod); statErr == nil {
			return directory
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			testingHandle.Fatalf("could not locate go.mod from %s", directory)
		}
		directory = parent
	}
}

func TestCTX(testingHandle *testing.T) {
	binary := buildBinary(testingHandle)

	var explicitFilePath string

	testCases := []struct {
		name          string
		arguments     []string
		prepare       func(*testing.T) string
		expectError   bool
		expectWarning bool
		validate      func(*testing.T, string)
	}{
		{
			name: "DocFlagCallChainRaw",
			arguments: []string{
				"callchain",
				"github.com/temirov/ctx/commands.GetContentData",
				"--doc",
				"--format",
				"raw",
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				if strings.Count(output, "strings.ToLower") == 0 {
					t.Errorf("expected documentation entry for strings.ToLower")
				}
			},
		},
		{
			name: "DefaultFormatTreeCommandJSON",
			arguments: []string{
				appTypes.CommandTree,
				"fileA.txt",
				"directoryB",
			},
			prepare: func(t *testing.T) string {
				return setupTestDirectory(t, map[string]string{
					"fileA.txt":             "A",
					"directoryB/":           "",
					"directoryB/itemB1.txt": "B1",
				})
			},
			validate: func(t *testing.T, output string) {
				var nodes []appTypes.TreeOutputNode
				if err := json.Unmarshal([]byte(output), &nodes); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(nodes) != 2 {
					t.Fatalf("expected two top‑level nodes, got %d", len(nodes))
				}
			},
		},
		{
			name: "DefaultFormatContentCommandJSON",
			arguments: []string{
				appTypes.CommandContent,
				"fileA.txt",
				"directoryB",
			},
			prepare: func(t *testing.T) string {
				return setupTestDirectory(t, map[string]string{
					"fileA.txt":             "Content A",
					"directoryB/":           "",
					"directoryB/itemB1.txt": "Content B1",
				})
			},
			validate: func(t *testing.T, output string) {
				var files []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &files); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(files) != 2 {
					t.Fatalf("expected two items, got %d", len(files))
				}
			},
		},
		{
			name: "RawFormatExplicitFlag",
			arguments: []string{
				appTypes.CommandContent,
				"onlyfile.txt",
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string {
				dir := setupTestDirectory(t, map[string]string{
					"onlyfile.txt": "Explicit raw content",
				})
				explicitFilePath = filepath.Join(dir, "onlyfile.txt")
				return dir
			},
			validate: func(t *testing.T, output string) {
				if !(strings.Contains(output, "File: "+explicitFilePath) &&
					strings.Contains(output, "Explicit raw content") &&
					strings.Contains(output, "End of file: "+explicitFilePath)) {
					t.Fatalf("unexpected raw content output\n%s", output)
				}
			},
		},
		{
			name: "RawFormatExplicitTree",
			arguments: []string{
				appTypes.CommandTree,
				"onlyfile.txt",
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string {
				dir := setupTestDirectory(t, map[string]string{
					"onlyfile.txt": "Explicit raw content",
				})
				explicitFilePath = filepath.Join(dir, "onlyfile.txt")
				return dir
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "[File] "+explicitFilePath) {
					t.Fatalf("unexpected raw tree output\n%s", output)
				}
			},
		},
		{
			name: "CallChainRaw",
			arguments: []string{
				appTypes.CommandCallChain,
				"github.com/temirov/ctx/commands.GetContentData",
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "Target Function: github.com/temirov/ctx/commands.GetContentData") {
					t.Fatalf("missing target function in output")
				}
			},
		},
		{
			name: "CallChainJSON",
			arguments: []string{
				appTypes.CommandCallChain,
				"github.com/temirov/ctx/commands.GetContentData",
				"--format",
				appTypes.FormatJSON,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				var list []appTypes.CallChainOutput
				if err := json.Unmarshal([]byte(output), &list); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(list) == 0 {
					t.Fatalf("expected at least one element, got zero")
				}
				chain := list[0]
				if chain.TargetFunction != "github.com/temirov/ctx/commands.GetContentData" {
					t.Fatalf("unexpected target function %q", chain.TargetFunction)
				}
			},
		},
		{
			name:          "VersionFlag",
			arguments:     []string{"--version"},
			prepare:       func(t *testing.T) string { return setupTestDirectory(t, nil) },
			expectError:   false,
			expectWarning: false,
			validate: func(t *testing.T, output string) {
				const prefix = "ctx version:"
				if !strings.HasPrefix(output, prefix) {
					t.Fatalf("version output should start with %q\n%s", prefix, output)
				}
			},
		},
		{
			name: "JsonFormatContentUnreadableFile",
			arguments: []string{
				appTypes.CommandContent,
				"readable.txt",
				"unreadable.txt",
				"--format",
				appTypes.FormatJSON,
			},
			prepare: func(t *testing.T) string {
				if runtime.GOOS == "windows" || os.Geteuid() == 0 {
					t.Skip("Skipping unreadable file test on Windows or as root")
				}
				return setupTestDirectory(t, map[string]string{
					"readable.txt":   "OK",
					"unreadable.txt": "<UNREADABLE>",
				})
			},
			expectWarning: true,
			validate: func(t *testing.T, output string) {
				var files []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &files); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(files) != 1 {
					t.Fatalf("expected one readable item, got %d", len(files))
				}
			},
		},
		{
			name: "InvalidFormatValue",
			arguments: []string{
				appTypes.CommandContent,
				"a.txt",
				"--format",
				"yaml",
			},
			prepare:     func(t *testing.T) string { return setupTestDirectory(t, map[string]string{"a.txt": "A"}) },
			expectError: true,
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "Invalid format value 'yaml'") {
					t.Errorf("expected error about invalid format value, got:\n%s", output)
				}
			},
		},
		{
			name: "CallChainMainDoc",
			arguments: []string{
				appTypes.CommandCallChain,
				"main.main",
				"--doc",
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "main.main") {
					t.Fatalf("expected main.main documentation in output, got:\n%s", output)
				}

				for _, line := range strings.Split(output, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "Error:") {
						t.Fatalf("unexpected error message in output:\n%s", output)
					}
				}
			},
		},
	}

	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(t *testing.T) {
			workingDir := testCase.prepare(t)

			var output string
			if testCase.expectError {
				output = runCommandExpectError(t, binary, testCase.arguments, workingDir)
			} else if testCase.expectWarning {
				output = runCommandWithWarnings(t, binary, testCase.arguments, workingDir)
			} else {
				output = runCommand(t, binary, testCase.arguments, workingDir)
			}

			testCase.validate(t, output)
		})
	}
}
