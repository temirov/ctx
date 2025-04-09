package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/temirov/content/types"
)

// buildBinary compiles the binary from the module root and returns its path.
// #nosec G204: we intentionally invoke "go build" with variable arguments
func buildBinary(testSetup *testing.T) string {
	testSetup.Helper()
	temporaryDirectory := testSetup.TempDir()
	binaryName := "content_integration"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(temporaryDirectory, binaryName)

	currentDirectory, directoryError := os.Getwd()
	if directoryError != nil {
		testSetup.Fatalf("Failed to get working directory: %v", directoryError)
	}
	moduleRoot := filepath.Dir(currentDirectory)

	buildCommand := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCommand.Dir = moduleRoot

	outputData, buildErr := buildCommand.CombinedOutput()
	if buildErr != nil {
		testSetup.Fatalf("Failed to build binary in %s: %v, output: %s", moduleRoot, buildErr, string(outputData))
	}
	return binaryPath
}

// runCommand executes the binary with the given args in workDir.
// It returns combined stdout+stderr as a string.
// It fails the test if the command exits with an error.
func runCommand(testSetup *testing.T, binaryPath string, args []string, workDir string) string {
	testSetup.Helper()
	command := exec.Command(binaryPath, args...)
	command.Dir = workDir

	var stdOutBuffer, stdErrBuffer bytes.Buffer
	command.Stdout = &stdOutBuffer
	command.Stderr = &stdErrBuffer

	runError := command.Run()
	stdOutString := stdOutBuffer.String()
	stdErrString := stdErrBuffer.String()

	combinedOutputForLog := fmt.Sprintf("Stdout:\n%s\nStderr:\n%s", stdOutString, stdErrString)

	if runError != nil {
		exitError, isExitError := runError.(*exec.ExitError)
		errorDetails := fmt.Sprintf("Command '%s %s' failed in dir '%s'", filepath.Base(binaryPath), strings.Join(args, " "), workDir)
		if isExitError {
			errorDetails += fmt.Sprintf("\nExit Code: %d", exitError.ExitCode())
		} else {
			errorDetails += fmt.Sprintf("\nError Type: %T", runError)
		}
		errorDetails += fmt.Sprintf("\nError: %v\nOutput:\n%s", runError, combinedOutputForLog)
		testSetup.Fatalf(errorDetails)
	}

	if strings.Contains(stdErrString, "Warning:") {
		testSetup.Logf("Command '%s %s' succeeded but produced warnings:\n%s", filepath.Base(binaryPath), strings.Join(args, " "), stdErrString)
	}

	return stdOutString
}

// runCommandExpectError executes the binary, expecting a non-zero exit code.
// It returns combined stdout+stderr as a string.
// It fails the test if the command exits successfully (exit code 0).
func runCommandExpectError(testSetup *testing.T, binaryPath string, args []string, workDir string) string {
	testSetup.Helper()
	command := exec.Command(binaryPath, args...)
	command.Dir = workDir

	var stdOutErrBuffer bytes.Buffer
	command.Stdout = &stdOutErrBuffer
	command.Stderr = &stdOutErrBuffer

	runError := command.Run()
	outputString := stdOutErrBuffer.String()

	if runError == nil {
		testSetup.Fatalf("Command '%s %s' in dir '%s' succeeded unexpectedly.\nOutput:\n%s", filepath.Base(binaryPath), strings.Join(args, " "), workDir, outputString)
	}

	_, isExitError := runError.(*exec.ExitError)
	if !isExitError {
		testSetup.Logf("Command '%s %s' failed with non-ExitError type: %T, Error: %v", filepath.Base(binaryPath), strings.Join(args, " "), runError, runError)
	}

	return outputString
}

// runCommandWithWarnings executes the binary, expecting a zero exit code but warnings on stderr.
// It returns combined stdout+stderr as a string.
// It fails the test if the command exits with an error OR if no warnings are found.
func runCommandWithWarnings(testSetup *testing.T, binaryPath string, args []string, workDir string) string {
	testSetup.Helper()
	command := exec.Command(binaryPath, args...)
	command.Dir = workDir

	var stdOutBuffer, stdErrBuffer bytes.Buffer
	command.Stdout = &stdOutBuffer
	command.Stderr = &stdErrBuffer

	runError := command.Run()
	stdOutString := stdOutBuffer.String()
	stdErrString := stdErrBuffer.String()

	combinedOutputForLog := fmt.Sprintf("Stdout:\n%s\nStderr:\n%s", stdOutString, stdErrString)

	if runError != nil {
		exitError, isExitError := runError.(*exec.ExitError)
		errorDetails := fmt.Sprintf("Command '%s %s' failed unexpectedly in dir '%s'", filepath.Base(binaryPath), strings.Join(args, " "), workDir)
		if isExitError {
			errorDetails += fmt.Sprintf("\nExit Code: %d", exitError.ExitCode())
		} else {
			errorDetails += fmt.Sprintf("\nError Type: %T", runError)
		}
		errorDetails += fmt.Sprintf("\nError: %v\nOutput:\n%s", runError, combinedOutputForLog)
		testSetup.Fatalf(errorDetails)
	}

	if !strings.Contains(stdErrString, "Warning:") {
		testSetup.Fatalf("Command '%s %s' succeeded but did not produce expected warnings on stderr.\nStderr:\n%s", filepath.Base(binaryPath), strings.Join(args, " "), stdErrString)
	}

	return stdOutString
}

// setupTestDirectory creates a temporary directory structure for testing.
// Returns the path to the root temporary directory.
func setupTestDirectory(testSetup *testing.T, structure map[string]string) string {
	testSetup.Helper()
	tempDir := testSetup.TempDir()

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		dirPath := filepath.Dir(fullPath)

		mkdirErr := os.MkdirAll(dirPath, 0755)
		if mkdirErr != nil {
			testSetup.Fatalf("Failed to create directory %s: %v", dirPath, mkdirErr)
		}

		if content == "" {
			if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
				mkdirErr = os.Mkdir(fullPath, 0755)
				if mkdirErr != nil {
					testSetup.Fatalf("Failed to create directory %s: %v", fullPath, mkdirErr)
				}
			}
		} else if content == "<UNREADABLE>" {
			writeErr := os.WriteFile(fullPath, []byte("cannot read this"), 0644)
			if writeErr != nil {
				testSetup.Fatalf("Failed to write pre-unreadable file %s: %v", fullPath, writeErr)
			}
			chmodErr := os.Chmod(fullPath, 0000)
			if chmodErr != nil {
				testSetup.Logf("Warning: Failed to set file %s unreadable: %v", fullPath, chmodErr)
			}
		} else {
			writeErr := os.WriteFile(fullPath, []byte(content), 0644)
			if writeErr != nil {
				testSetup.Fatalf("Failed to write file %s: %v", fullPath, writeErr)
			}
		}
	}
	return tempDir
}

func TestRawFormat_TreeCommand_Default(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"fileA.txt":       "A",
		"dirB/":           "",
		"dirB/itemB1.txt": "B1",
	})

	output := runCommand(testValue, binary, []string{"tree", "fileA.txt", "dirB"}, testDir)

	absFileA, _ := filepath.Abs(filepath.Join(testDir, "fileA.txt"))
	absDirB, _ := filepath.Abs(filepath.Join(testDir, "dirB"))

	expectedFileA := fmt.Sprintf("[File] %s", absFileA)
	expectedDirBHeader := fmt.Sprintf("--- Directory Tree: %s ---", absDirB)

	if !strings.Contains(output, expectedFileA) {
		testValue.Errorf("Raw tree output missing file marker for fileA.\nOutput: %s", output)
	}
	if !strings.Contains(output, expectedDirBHeader) {
		testValue.Errorf("Raw tree output missing header for dirB.\nOutput: %s", output)
	}
	if !strings.Contains(output, "└── itemB1.txt") && !strings.Contains(output, "├── itemB1.txt") {
		testValue.Errorf("Raw tree output missing itemB1.txt from dirB tree.\nOutput: %s", output)
	}
}

func TestRawFormat_ContentCommand_Default(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"fileA.txt":       "Content A",
		"dirB/":           "",
		"dirB/itemB1.txt": "Content B1",
	})

	output := runCommand(testValue, binary, []string{"content", "fileA.txt", "dirB"}, testDir)

	absFileA, _ := filepath.Abs(filepath.Join(testDir, "fileA.txt"))
	absFileB1, _ := filepath.Abs(filepath.Join(testDir, "dirB", "itemB1.txt"))

	expectedHeaderA := fmt.Sprintf("File: %s", absFileA)
	expectedHeaderB1 := fmt.Sprintf("File: %s", absFileB1)
	sep := "----------------------------------------"

	if !strings.Contains(output, expectedHeaderA) || !strings.Contains(output, "Content A") || !strings.Contains(output, "End of file: "+absFileA) {
		testValue.Errorf("Raw content output missing parts for fileA.\nOutput: %s", output)
	}
	if !strings.Contains(output, expectedHeaderB1) || !strings.Contains(output, "Content B1") || !strings.Contains(output, "End of file: "+absFileB1) {
		testValue.Errorf("Raw content output missing parts for itemB1.\nOutput: %s", output)
	}
	if strings.Count(output, sep) != 2 {
		testValue.Errorf("Expected 2 separators in raw content output, found %d.\nOutput: %s", strings.Count(output, sep), output)
	}
	if strings.Index(output, expectedHeaderA) > strings.Index(output, expectedHeaderB1) {
		testValue.Errorf("Raw content output order seems incorrect.\nOutput: %s", output)
	}
}

func TestRawFormat_FlagExplicit(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"fileA.txt": "Content A",
	})

	output := runCommand(testValue, binary, []string{"content", "fileA.txt", "--format", "raw"}, testDir)

	absFileA, _ := filepath.Abs(filepath.Join(testDir, "fileA.txt"))
	expectedHeaderA := fmt.Sprintf("File: %s", absFileA)

	if !strings.Contains(output, expectedHeaderA) || !strings.Contains(output, "Content A") {
		testValue.Errorf("Raw content output missing parts for fileA when using --format raw.\nOutput: %s", output)
	}
}

func TestJsonFormat_ContentCommand_Mixed(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"fileA.txt":        "Content A",
		"dirB/":            "",
		"dirB/.ignore":     "ignored.txt",
		"dirB/itemB1.txt":  "Content B1",
		"dirB/ignored.txt": "Ignored Content B",
		"fileC.txt":        "Content C",
	})

	output := runCommand(testValue, binary, []string{"content", "fileA.txt", "dirB", "fileC.txt", "--format", "json"}, testDir)

	var results []types.FileOutput
	err := json.Unmarshal([]byte(output), &results)
	if err != nil {
		testValue.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", err, output)
	}

	if len(results) != 3 {
		testValue.Fatalf("Expected 3 items in JSON array, got %d.\nOutput: %s", len(results), output)
	}

	absFileA, _ := filepath.Abs(filepath.Join(testDir, "fileA.txt"))
	absFileB1, _ := filepath.Abs(filepath.Join(testDir, "dirB", "itemB1.txt"))
	absFileC, _ := filepath.Abs(filepath.Join(testDir, "fileC.txt"))

	if results[0].Path != absFileA || results[0].Content != "Content A" || results[0].Type != "file" {
		testValue.Errorf("JSON item 0 mismatch for fileA. Got: %+v", results[0])
	}
	if results[1].Path != absFileB1 || results[1].Content != "Content B1" || results[1].Type != "file" {
		testValue.Errorf("JSON item 1 mismatch for itemB1. Got: %+v", results[1])
	}
	if results[2].Path != absFileC || results[2].Content != "Content C" || results[2].Type != "file" {
		testValue.Errorf("JSON item 2 mismatch for fileC. Got: %+v", results[2])
	}

	if strings.Contains(output, "Ignored Content B") {
		testValue.Errorf("Ignored content from dirB/ignored.txt found in JSON output.\nOutput: %s", output)
	}
	if strings.Contains(output, `dirB/.ignore`) {
		testValue.Errorf("The .ignore file itself was included in JSON output.\nOutput: %s", output)
	}
}

func TestJsonFormat_TreeCommand_Mixed(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"fileA.txt":           "A",
		"dirB/":               "",
		"dirB/.ignore":        "ignored.txt",
		"dirB/itemB1.txt":     "B1",
		"dirB/ignored.txt":    "ignored",
		"dirB/sub/":           "",
		"dirB/sub/itemB2.txt": "B2",
		"fileC.txt":           "C",
	})

	output := runCommand(testValue, binary, []string{"tree", "fileA.txt", "dirB", "fileC.txt", "--format", "json"}, testDir)

	var results []types.TreeOutputNode
	err := json.Unmarshal([]byte(output), &results)
	if err != nil {
		testValue.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", err, output)
	}

	if len(results) != 3 {
		testValue.Fatalf("Expected 3 top-level items in JSON array, got %d.\nOutput: %s", len(results), output)
	}

	absFileA, _ := filepath.Abs(filepath.Join(testDir, "fileA.txt"))
	absDirB, _ := filepath.Abs(filepath.Join(testDir, "dirB"))
	absFileC, _ := filepath.Abs(filepath.Join(testDir, "fileC.txt"))

	if results[0].Path != absFileA || results[0].Type != "file" || results[0].Name != "fileA.txt" {
		testValue.Errorf("JSON item 0 mismatch for fileA. Got: %+v", results[0])
	}

	if results[1].Path != absDirB || results[1].Type != "directory" || results[1].Name != "dirB" {
		testValue.Errorf("JSON item 1 mismatch for dirB. Got: %+v", results[1])
	}
	if len(results[1].Children) != 2 {
		var childNames []string
		for _, ch := range results[1].Children {
			childNames = append(childNames, ch.Name)
		}
		testValue.Fatalf("Expected 2 children for dirB, got %d. Children: %v \nOutput:\n%s", len(results[1].Children), childNames, output)
	}
	childB1 := results[1].Children[0]
	if childB1.Name != "itemB1.txt" || childB1.Type != "file" || len(childB1.Children) != 0 {
		testValue.Errorf("Child itemB1 mismatch. Got: %+v", childB1)
	}
	childSub := results[1].Children[1]
	if childSub.Name != "sub" || childSub.Type != "directory" || len(childSub.Children) != 1 {
		testValue.Errorf("Child sub mismatch. Got: %+v", childSub)
	}
	if len(childSub.Children) > 0 {
		if childSub.Children[0].Name != "itemB2.txt" || childSub.Children[0].Type != "file" {
			testValue.Errorf("Grandchild itemB2 mismatch. Got: %+v", childSub.Children[0])
		}
	} else {
		testValue.Errorf("Grandchild itemB2 missing from sub")
	}

	if results[2].Path != absFileC || results[2].Type != "file" || results[2].Name != "fileC.txt" {
		testValue.Errorf("JSON item 2 mismatch for fileC. Got: %+v", results[2])
	}
}

func TestJsonFormat_Content_UnreadableFile(testValue *testing.T) {
	if runtime.GOOS == "windows" {
		testValue.Skip("Skipping unreadable file test on Windows")
	}
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"readable.txt":   "OK",
		"unreadable.txt": "<UNREADABLE>",
	})

	stdoutOutput := runCommandWithWarnings(testValue, binary, []string{"content", "readable.txt", "unreadable.txt", "--format", "json"}, testDir)

	var results []types.FileOutput
	err := json.Unmarshal([]byte(stdoutOutput), &results)
	if err != nil {
		testValue.Fatalf("Failed to unmarshal JSON stdout: %v\nStdout:\n%s", err, stdoutOutput)
	}

	if len(results) != 1 {
		testValue.Fatalf("Expected 1 item (readable.txt) in JSON array, got %d.\nJSON Output:\n%s", len(results), stdoutOutput)
	}

	absReadable, _ := filepath.Abs(filepath.Join(testDir, "readable.txt"))
	if results[0].Path != absReadable || results[0].Content != "OK" {
		testValue.Errorf("JSON item 0 mismatch for readable.txt. Got: %+v", results[0])
	}
}

func TestInvalidFormatValue(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{"a.txt": "A"})

	output := runCommandExpectError(testValue, binary, []string{"content", "a.txt", "--format", "yaml"}, testDir)

	if !strings.Contains(output, "Invalid format value 'yaml'") {
		testValue.Errorf("Expected error about invalid format value, got:\n%s", output)
	}
}

func TestInput_NonExistentFile(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{})

	output := runCommandExpectError(testValue, binary, []string{"content", "no_such_file.txt"}, testDir)

	if !strings.Contains(output, "no_such_file.txt") || !strings.Contains(output, "does not exist") {
		testValue.Errorf("Expected error about non-existent file, got:\n%s", output)
	}
}

func TestMultiDir_ArgsOrder(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"dir1/":      "",
		"dir1/a.txt": "A",
		"dir2/":      "",
		"dir2/b.txt": "B",
	})

	output := runCommandExpectError(testValue, binary, []string{"content", "dir1", "--format", "json", "dir2"}, testDir)
	if !strings.Contains(output, "Positional argument 'dir2' found after flags") {
		testValue.Errorf("Expected error about positional arg after flags, got:\n%s", output)
	}
}
