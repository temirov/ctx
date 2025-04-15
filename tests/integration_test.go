package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	appTypes "github.com/temirov/ctx/types"
)

// #nosec G204
func buildBinary(testSetup *testing.T) string {
	testSetup.Helper()
	temporaryDirectory := testSetup.TempDir()
	binaryName := "ctx_integration_test_binary"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(temporaryDirectory, binaryName)
	currentDirectory, directoryError := os.Getwd()
	if directoryError != nil {
		testSetup.Fatalf("Failed to get current working directory: %v", directoryError)
	}
	moduleRoot := filepath.Dir(currentDirectory)
	buildCommand := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCommand.Dir = moduleRoot
	outputData, buildErr := buildCommand.CombinedOutput()
	if buildErr != nil {
		testSetup.Fatalf("Failed to build binary in %s: %v\nBuild Output:\n%s", moduleRoot, buildErr, string(outputData))
	}
	return binaryPath
}

// #nosec G204
func runCommand(testSetup *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testSetup.Helper()
	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory
	var standardOutputBuffer, standardErrorBuffer bytes.Buffer
	command.Stdout = &standardOutputBuffer
	command.Stderr = &standardErrorBuffer
	runError := command.Run()
	standardOutputString := standardOutputBuffer.String()
	standardErrorString := standardErrorBuffer.String()
	combinedOutputForLog := fmt.Sprintf("--- Command ---\n%s %s\n--- Working Directory ---\n%s\n--- Standard Output ---\n%s\n--- Standard Error ---\n%s",
		filepath.Base(binaryPath), strings.Join(arguments, " "), workingDirectory, standardOutputString, standardErrorString)
	if runError != nil {
		exitError, isExitError := runError.(*exec.ExitError)
		errorDetails := fmt.Sprintf("Command failed unexpectedly.\n%s", combinedOutputForLog)
		if isExitError {
			errorDetails += fmt.Sprintf("\n--- Exit Code ---\n%d", exitError.ExitCode())
		} else {
			errorDetails += fmt.Sprintf("\n--- Error Type ---\n%T", runError)
		}
		errorDetails += fmt.Sprintf("\n--- Error ---\n%v", runError)
		testSetup.Fatalf(errorDetails)
	}
	if strings.Contains(standardErrorString, "Warning:") {
		testSetup.Logf("Command succeeded but produced warnings:\n%s", combinedOutputForLog)
	}
	return standardOutputString
}

// #nosec G204
func runCommandExpectError(testSetup *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testSetup.Helper()
	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory
	var combinedOutputBuffer bytes.Buffer
	command.Stdout = &combinedOutputBuffer
	command.Stderr = &combinedOutputBuffer
	runError := command.Run()
	outputString := combinedOutputBuffer.String()
	if runError == nil {
		testSetup.Fatalf("Command succeeded unexpectedly.\n--- Command ---\n%s %s\n--- Working Directory ---\n%s\n--- Combined Output ---\n%s",
			filepath.Base(binaryPath), strings.Join(arguments, " "), workingDirectory, outputString)
	}
	_, isExitError := runError.(*exec.ExitError)
	if !isExitError {
		testSetup.Logf("Command failed with non-ExitError type (%T): %v\n--- Command ---\n%s %s\n--- Working Directory ---\n%s",
			runError, runError, filepath.Base(binaryPath), strings.Join(arguments, " "), workingDirectory)
	}
	return outputString
}

// #nosec G204
func runCommandWithWarnings(testSetup *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testSetup.Helper()
	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory
	var standardOutputBuffer, standardErrorBuffer bytes.Buffer
	command.Stdout = &standardOutputBuffer
	command.Stderr = &standardErrorBuffer
	runError := command.Run()
	standardOutputString := standardOutputBuffer.String()
	standardErrorString := standardErrorBuffer.String()
	combinedOutputForLog := fmt.Sprintf("--- Command ---\n%s %s\n--- Working Directory ---\n%s\n--- Standard Output ---\n%s\n--- Standard Error ---\n%s",
		filepath.Base(binaryPath), strings.Join(arguments, " "), workingDirectory, standardOutputString, standardErrorString)
	if runError != nil {
		var exitError *exec.ExitError
		isExitError := errors.As(runError, &exitError)
		errorDetails := fmt.Sprintf("Command failed unexpectedly when warnings were expected.\n%s", combinedOutputForLog)
		if isExitError {
			errorDetails += fmt.Sprintf("\n--- Exit Code ---\n%d", exitError.ExitCode())
		} else {
			errorDetails += fmt.Sprintf("\n--- Error Type ---\n%T", runError)
		}
		errorDetails += fmt.Sprintf("\n--- Error ---\n%v", runError)
		testSetup.Fatalf(errorDetails)
	}
	if !strings.Contains(standardErrorString, "Warning:") {
		testSetup.Fatalf("Command succeeded but did not produce expected warnings on stderr.\n%s", combinedOutputForLog)
	}
	return standardOutputString
}

func setupTestDirectory(testSetup *testing.T, directoryStructure map[string]string) string {
	testSetup.Helper()
	temporaryDirectoryRoot := testSetup.TempDir()
	for relativePath, content := range directoryStructure {
		absolutePath := filepath.Join(temporaryDirectoryRoot, relativePath)
		directoryPath := filepath.Dir(absolutePath)
		mkdirErr := os.MkdirAll(directoryPath, 0755)
		if mkdirErr != nil {
			testSetup.Fatalf("Failed to create directory %s: %v", directoryPath, mkdirErr)
		}
		if strings.HasSuffix(relativePath, "/") || content == "" {
			if _, statErr := os.Stat(absolutePath); os.IsNotExist(statErr) {
				mkdirErr = os.Mkdir(absolutePath, 0755)
				if mkdirErr != nil {
					testSetup.Fatalf("Failed to create directory %s: %v", absolutePath, mkdirErr)
				}
			}
		} else if content == "<UNREADABLE>" {
			initialContent := []byte("content should be unreadable")
			writeErr := os.WriteFile(absolutePath, initialContent, 0644)
			if writeErr != nil {
				testSetup.Fatalf("Failed to write initial content for unreadable file %s: %v", absolutePath, writeErr)
			}
			chmodErr := os.Chmod(absolutePath, 0000)
			if chmodErr != nil {
				testSetup.Logf("Warning: Failed to set file %s permissions to 0000: %v. Test validity might be affected.", absolutePath, chmodErr)
			}
		} else {
			writeErr := os.WriteFile(absolutePath, []byte(content), 0644)
			if writeErr != nil {
				testSetup.Fatalf("Failed to write file %s: %v", absolutePath, writeErr)
			}
		}
	}
	return temporaryDirectoryRoot
}

func getModuleRoot(testSetup *testing.T) string {
	testSetup.Helper()
	currentDirectory, err := os.Getwd()
	if err != nil {
		testSetup.Fatalf("Failed to get working directory: %v", err)
	}
	directory := currentDirectory
	for {
		goModPath := filepath.Join(directory, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return directory
		}
		parentDirectory := filepath.Dir(directory)
		if parentDirectory == directory {
			testSetup.Fatalf("Could not find go.mod in or above test directory: %s", currentDirectory)
		}
		directory = parentDirectory
	}
}

// TestDefaultFormat_TreeCommand_Default verifies that when no format flag is provided the tree command outputs JSON.
func TestDefaultFormat_TreeCommand_Default(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":             "File A content",
		"directoryB/":           "",
		"directoryB/itemB1.txt": "File B1 content",
	})
	// No explicit --format flag so default (JSON) should be produced.
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandTree, "fileA.txt", "directoryB"}, testDirectory)
	var results []appTypes.TreeOutputNode
	jsonUnmarshalError := json.Unmarshal([]byte(output), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}
	expectedTopLevelCount := 2
	if len(results) != expectedTopLevelCount {
		testInstance.Errorf("Expected %d top-level items in JSON output, got %d.", expectedTopLevelCount, len(results))
	}
	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteDirectoryBPath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB"))
	foundFileA := false
	foundDirectoryB := false
	for _, item := range results {
		if item.Path == absoluteFileAPath && item.Type == appTypes.NodeTypeFile {
			foundFileA = true
		}
		if item.Path == absoluteDirectoryBPath && item.Type == appTypes.NodeTypeDirectory {
			foundDirectoryB = true
			if len(item.Children) != 1 {
				testInstance.Errorf("Expected directoryB to have 1 child, got %d", len(item.Children))
			}
		}
	}
	if !foundFileA || !foundDirectoryB {
		testInstance.Errorf("Expected fileA.txt and directoryB nodes in JSON output.")
	}
}

// TestDefaultFormat_ContentCommand_Default verifies that when no format flag is provided the content command outputs JSON.
func TestDefaultFormat_ContentCommand_Default(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":             "Content A",
		"directoryB/":           "",
		"directoryB/itemB1.txt": "Content B1",
	})
	// No explicit --format flag; default output should be JSON.
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandContent, "fileA.txt", "directoryB"}, testDirectory)
	var results []appTypes.FileOutput
	jsonUnmarshalError := json.Unmarshal([]byte(output), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}
	expectedCount := 2 // fileA.txt and directoryB/itemB1.txt
	if len(results) != expectedCount {
		testInstance.Fatalf("Expected %d items in JSON output, got %d.\nOutput:\n%s", expectedCount, len(results), output)
	}
	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteItemB1Path, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", "itemB1.txt"))
	foundA, foundB1 := false, false
	for _, item := range results {
		if item.Path == absoluteFileAPath {
			if item.Content != "Content A" {
				testInstance.Errorf("Content mismatch for fileA.txt. Expected 'Content A', got '%s'", item.Content)
			}
			foundA = true
		}
		if item.Path == absoluteItemB1Path {
			if item.Content != "Content B1" {
				testInstance.Errorf("Content mismatch for itemB1.txt. Expected 'Content B1', got '%s'", item.Content)
			}
			foundB1 = true
		}
	}
	if !foundA || !foundB1 {
		testInstance.Errorf("Missing expected items in JSON output. Found fileA.txt: %v, Found itemB1.txt: %v", foundA, foundB1)
	}
}

func TestRawFormat_FlagExplicit(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"onlyfile.txt": "Explicit raw content",
	})
	outputContent := runCommand(testInstance, binaryPath, []string{appTypes.CommandContent, "onlyfile.txt", "--format", appTypes.FormatRaw}, testDirectory)
	absoluteFilePath, _ := filepath.Abs(filepath.Join(testDirectory, "onlyfile.txt"))
	// When explicitly using raw format, the output is expected to be raw text and not JSON.
	if !strings.Contains(outputContent, "File: "+absoluteFilePath) || !strings.Contains(outputContent, "Explicit raw content") || !strings.Contains(outputContent, "End of file: "+absoluteFilePath) {
		testInstance.Errorf("Explicit --format raw did not produce expected raw content output.\nOutput:\n%s", outputContent)
	}
	outputTree := runCommand(testInstance, binaryPath, []string{appTypes.CommandTree, "onlyfile.txt", "--format", appTypes.FormatRaw}, testDirectory)
	if !strings.Contains(outputTree, "[File] "+absoluteFilePath) {
		testInstance.Errorf("Explicit --format raw did not produce expected raw tree output.\nOutput:\n%s", outputTree)
	}
}

func TestJsonFormat_ContentCommand_Mixed(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":              "Content A",
		"directoryB/":            "",
		"directoryB/.ignore":     "ignored.txt",       // File to be ignored
		"directoryB/itemB1.txt":  "Content B1",        // File to be included
		"directoryB/ignored.txt": "Ignored Content B", // Ignored content
		"fileC.txt":              "Content C",
	})
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandContent, "fileA.txt", "directoryB", "fileC.txt", "--format", appTypes.FormatJSON}, testDirectory)
	var results []appTypes.FileOutput
	jsonUnmarshalError := json.Unmarshal([]byte(output), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}
	expectedCount := 3 // fileA.txt, directoryB/itemB1.txt, fileC.txt
	if len(results) != expectedCount {
		testInstance.Fatalf("Expected %d items in JSON array, got %d.\nOutput:\n%s", expectedCount, len(results), output)
	}
	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteItemB1Path, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", "itemB1.txt"))
	absoluteFileCPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileC.txt"))
	foundA, foundB1, foundC := false, false, false
	pathsFound := make(map[string]string)
	for _, item := range results {
		if item.Type != appTypes.NodeTypeFile {
			testInstance.Errorf("Unexpected item type '%s' in JSON output for path %s", item.Type, item.Path)
		}
		pathsFound[item.Path] = item.Content
		switch item.Path {
		case absoluteFileAPath:
			if item.Content != "Content A" {
				testInstance.Errorf("Content mismatch for fileA.txt. Expected 'Content A', got '%s'", item.Content)
			}
			foundA = true
		case absoluteItemB1Path:
			if item.Content != "Content B1" {
				testInstance.Errorf("Content mismatch for itemB1.txt. Expected 'Content B1', got '%s'", item.Content)
			}
			foundB1 = true
		case absoluteFileCPath:
			if item.Content != "Content C" {
				testInstance.Errorf("Content mismatch for fileC.txt. Expected 'Content C', got '%s'", item.Content)
			}
			foundC = true
		}
	}
	if !foundA || !foundB1 || !foundC {
		testInstance.Errorf("Missing expected items in JSON output. Found fileA.txt: %v, Found itemB1.txt: %v, Found fileC.txt: %v", foundA, foundB1, foundC)
	}
	absoluteIgnoreFilePath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", ".ignore"))
	absoluteIgnoredFilePath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", "ignored.txt"))
	if _, exists := pathsFound[absoluteIgnoreFilePath]; exists {
		testInstance.Errorf(".ignore file was included in JSON output: %s", absoluteIgnoreFilePath)
	}
	if _, exists := pathsFound[absoluteIgnoredFilePath]; exists {
		testInstance.Errorf("Ignored file 'ignored.txt' was included in JSON output: %s", absoluteIgnoredFilePath)
	}
}

func TestJsonFormat_TreeCommand_Mixed(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":                 "Content A",
		"directoryB/":               "",
		"directoryB/.ignore":        "ignored.txt\nsub/", // Ignore file and sub directory
		"directoryB/itemB1.txt":     "Content B1",
		"directoryB/ignored.txt":    "Ignored B",
		"directoryB/sub/":           "",           // Ignored directory
		"directoryB/sub/itemB2.txt": "Content B2", // File within ignored directory
		"fileC.txt":                 "Content C",
	})
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandTree, "fileA.txt", "directoryB", "fileC.txt", "--format", appTypes.FormatJSON}, testDirectory)
	var results []appTypes.TreeOutputNode
	jsonUnmarshalError := json.Unmarshal([]byte(output), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}
	expectedTopLevelCount := 3 // fileA, directoryB, fileC
	if len(results) != expectedTopLevelCount {
		testInstance.Fatalf("Expected %d top-level items in JSON array, got %d.\nOutput:\n%s", expectedTopLevelCount, len(results), output)
	}
	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteDirectoryBPath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB"))
	absoluteFileCPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileC.txt"))
	if results[0].Path != absoluteFileAPath || results[0].Type != appTypes.NodeTypeFile || results[0].Name != "fileA.txt" {
		testInstance.Errorf("JSON item 0 mismatch for fileA.txt. Got: %+v", results[0])
	}
	if len(results[0].Children) != 0 {
		testInstance.Errorf("Expected fileA.txt node to have no children, got %d", len(results[0].Children))
	}
	if results[1].Path != absoluteDirectoryBPath || results[1].Type != appTypes.NodeTypeDirectory || results[1].Name != "directoryB" {
		testInstance.Errorf("JSON item 1 mismatch for directoryB. Got: %+v", results[1])
	}
	expectedChildrenCountB := 1 // Only itemB1.txt should be listed (.ignore, ignored.txt, sub/ are ignored)
	if len(results[1].Children) != expectedChildrenCountB {
		var childNames []string
		for _, child := range results[1].Children {
			childNames = append(childNames, child.Name)
		}
		testInstance.Fatalf("Expected %d children for directoryB, got %d. Children found: %v\nOutput:\n%s",
			expectedChildrenCountB, len(results[1].Children), childNames, output)
	}
	childB1 := results[1].Children[0]
	expectedChildB1Path := filepath.Join(absoluteDirectoryBPath, "itemB1.txt")
	if childB1.Path != expectedChildB1Path || childB1.Name != "itemB1.txt" || childB1.Type != appTypes.NodeTypeFile || len(childB1.Children) != 0 {
		testInstance.Errorf("Child itemB1 mismatch in directoryB. Expected Path: %s, Name: itemB1.txt, Type: file. Got: %+v", expectedChildB1Path, childB1)
	}
	if results[2].Path != absoluteFileCPath || results[2].Type != appTypes.NodeTypeFile || results[2].Name != "fileC.txt" {
		testInstance.Errorf("JSON item 2 mismatch for fileC.txt. Got: %+v", results[2])
	}
	if len(results[2].Children) != 0 {
		testInstance.Errorf("Expected fileC.txt node to have no children, got %d", len(results[2].Children))
	}
}

func TestJsonFormat_Content_UnreadableFile(testInstance *testing.T) {
	if runtime.GOOS == "windows" {
		testInstance.Skip("Skipping unreadable file permission test on Windows due to permission differences.")
	}
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"readable.txt":   "Readable content OK",
		"unreadable.txt": "<UNREADABLE>",
	})
	standardOutput := runCommandWithWarnings(testInstance, binaryPath, []string{appTypes.CommandContent, "readable.txt", "unreadable.txt", "--format", appTypes.FormatJSON}, testDirectory)
	var results []appTypes.FileOutput
	jsonUnmarshalError := json.Unmarshal([]byte(standardOutput), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON stdout: %v\nStdout:\n%s", jsonUnmarshalError, standardOutput)
	}
	expectedCount := 1 // Only readable.txt should be in the output
	if len(results) != expectedCount {
		testInstance.Fatalf("Expected %d item (readable.txt) in JSON output, got %d.\nJSON Output:\n%s", expectedCount, len(results), standardOutput)
	}
	absoluteReadablePath, _ := filepath.Abs(filepath.Join(testDirectory, "readable.txt"))
	if results[0].Path != absoluteReadablePath || results[0].Content != "Readable content OK" || results[0].Type != appTypes.NodeTypeFile {
		testInstance.Errorf("JSON item 0 mismatch for readable.txt. Got: %+v", results[0])
	}
}

func TestInvalidFormatValue(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{"a.txt": "A"})
	outputContent := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, "a.txt", "--format", "invalid-format"}, testDirectory)
	expectedErrorFragment := "Invalid format value 'invalid-format'"
	if !strings.Contains(outputContent, expectedErrorFragment) {
		testInstance.Errorf("Expected error message containing '%s' for content command, but got:\n%s", expectedErrorFragment, outputContent)
	}
	outputTree := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandTree, "a.txt", "--format", "xml"}, testDirectory)
	expectedErrorFragmentTree := "Invalid format value 'xml'"
	if !strings.Contains(outputTree, expectedErrorFragmentTree) {
		testInstance.Errorf("Expected error message containing '%s' for tree command, but got:\n%s", expectedErrorFragmentTree, outputTree)
	}
	moduleRoot := getModuleRoot(testInstance)
	outputCallchain := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandCallChain, "dummyFunc", "--format", "yaml"}, moduleRoot)
	expectedErrorFragmentCC := "Invalid format value 'yaml'"
	if !strings.Contains(outputCallchain, expectedErrorFragmentCC) {
		testInstance.Errorf("Expected error message containing '%s' for callchain command, but got:\n%s", expectedErrorFragmentCC, outputCallchain)
	}
}

func TestInput_NonExistentFile(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{})
	nonExistentFileName := "non_existent_file.abc"
	outputContent := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, nonExistentFileName}, testDirectory)
	expectedErrorFragment1 := fmt.Sprintf("path '%s'", nonExistentFileName)
	expectedErrorFragment2 := "does not exist"
	if !strings.Contains(outputContent, expectedErrorFragment1) || !strings.Contains(outputContent, expectedErrorFragment2) {
		testInstance.Errorf("Expected content error message containing '%s' and '%s', but got:\n%s", expectedErrorFragment1, expectedErrorFragment2, outputContent)
	}
	outputTree := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandTree, nonExistentFileName}, testDirectory)
	if !strings.Contains(outputTree, expectedErrorFragment1) || !strings.Contains(outputTree, expectedErrorFragment2) {
		testInstance.Errorf("Expected tree error message containing '%s' and '%s', but got:\n%s", expectedErrorFragment1, expectedErrorFragment2, outputTree)
	}
}

func TestArgs_PositionalAfterFlag(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"dir1/":      "",
		"dir1/a.txt": "A",
		"dir2/":      "",
		"dir2/b.txt": "B",
	})
	outputContent := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, "dir1", "--format", appTypes.FormatJSON, "dir2"}, testDirectory)
	expectedErrorFragment := "Positional argument 'dir2' found after flags"
	if !strings.Contains(outputContent, expectedErrorFragment) {
		testInstance.Errorf("Expected error message containing '%s' for content command, got:\n%s", expectedErrorFragment, outputContent)
	}
	outputTree := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandTree, "dir1", "--no-ignore", "dir2"}, testDirectory)
	if !strings.Contains(outputTree, expectedErrorFragment) {
		testInstance.Errorf("Expected error message containing '%s' for tree command, got:\n%s", expectedErrorFragment, outputTree)
	}
	moduleRoot := getModuleRoot(testInstance)
	outputCallchain := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandCallChain, "--format", appTypes.FormatJSON, "nonExistentFunctionForTest"}, moduleRoot)
	if strings.Contains(outputCallchain, "Positional argument") && strings.Contains(outputCallchain, "found after flags") {
		testInstance.Errorf("Callchain command incorrectly rejected positional argument after flag.\nOutput:\n%s", outputCallchain)
	}
	if !strings.Contains(outputCallchain, "target function") || !strings.Contains(outputCallchain, "not found") {
		testInstance.Errorf("Callchain command failed, but not with the expected 'function not found' error.\nOutput:\n%s", outputCallchain)
	} else {
		testInstance.Logf("Callchain command failed as expected (function not found), not due to arg parsing.\nOutput:\n%s", outputCallchain)
	}
}

func TestCallChain_Raw(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	moduleRoot := getModuleRoot(testInstance)
	targetFunction := "github.com/temirov/ctx/commands.GetContentData"
	expectedCaller := "github.com/temirov/ctx.runTreeOrContentCommand"
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandCallChain, targetFunction, "--format", appTypes.FormatRaw}, moduleRoot)
	if !strings.Contains(output, "----- CALLCHAIN METADATA -----") {
		testInstance.Errorf("Callchain raw output missing metadata header.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "Target Function: "+targetFunction) {
		testInstance.Errorf("Callchain raw output missing correct target function identifier. Expected '%s'.\nOutput:\n%s", targetFunction, output)
	}
	if !strings.Contains(output, "Callers:") {
		testInstance.Errorf("Callchain raw output missing 'Callers:' section.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, expectedCaller) {
		testInstance.Errorf("Callchain raw output missing expected caller '%s'.\nOutput:\n%s", expectedCaller, output)
	}
	if !strings.Contains(output, "Callees:") {
		testInstance.Errorf("Callchain raw output missing 'Callees:' section.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "----- FUNCTIONS -----") {
		testInstance.Errorf("Callchain raw output missing functions header.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "func GetContentData(rootDirectory string, ignorePatterns []string) ([]types.FileOutput, error)") {
		testInstance.Errorf("Callchain raw output missing source code signature for GetContentData.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "func runTreeOrContentCommand(commandName string, inputArguments []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) error") {
		testInstance.Errorf("Callchain raw output missing source code signature for caller '%s'.\nOutput:\n%s", expectedCaller, output)
	}
}

func TestCallChain_JSON(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	moduleRoot := getModuleRoot(testInstance)
	targetFunctionInput := "github.com/temirov/ctx/commands.GetContentData"
	expectedCaller := "github.com/temirov/ctx.runTreeOrContentCommand"
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandCallChain, targetFunctionInput, "--format", appTypes.FormatJSON}, moduleRoot)
	var result appTypes.CallChainOutput
	jsonUnmarshalError := json.Unmarshal([]byte(output), &result)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal Callchain JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}
	if result.TargetFunction != targetFunctionInput {
		testInstance.Errorf("JSON output TargetFunction mismatch. Expected '%s', got '%s'", targetFunctionInput, result.TargetFunction)
	}
	if len(result.Callers) == 0 {
		testInstance.Errorf("JSON output expected callers for GetContentData, but got none.")
	} else {
		foundCaller := false
		for _, caller := range result.Callers {
			if caller == expectedCaller {
				foundCaller = true
				break
			}
		}
		if !foundCaller {
			testInstance.Errorf("JSON output missing expected caller '%s' in callers list: %v", expectedCaller, result.Callers)
		}
	}
	if result.Callees == nil || len(*result.Callees) == 0 {
		testInstance.Logf("JSON output Callees list is empty or nil for GetContentData. This might be expected.")
	}
	if len(result.Functions) == 0 {
		testInstance.Errorf("JSON output expected Functions map to be non-empty, but it was.")
	}
	targetFunctionResolved := result.TargetFunction
	if _, ok := result.Functions[targetFunctionResolved]; !ok {
		testInstance.Errorf("JSON output Functions map missing entry for resolved target function '%s'", targetFunctionResolved)
	} else if !strings.Contains(result.Functions[targetFunctionResolved], "func GetContentData") {
		testInstance.Errorf("JSON output Functions map entry for target function '%s' doesn't seem to contain its source code.", targetFunctionResolved)
	}
	if _, ok := result.Functions[expectedCaller]; !ok {
		testInstance.Errorf("JSON output Functions map missing entry for expected caller '%s'", expectedCaller)
	} else if !strings.Contains(result.Functions[expectedCaller], "func runTreeOrContentCommand") {
		testInstance.Errorf("JSON output Functions map entry for caller '%s' doesn't seem to contain its source code.", expectedCaller)
	}
}

func TestVersionFlag(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{})
	output := runCommand(testInstance, binaryPath, []string{"--version"}, testDirectory)
	expectedVersionPrefix := "ctx version:"
	if !strings.HasPrefix(output, expectedVersionPrefix) {
		testInstance.Errorf("Expected version output to start with '%s', got:\n%s", expectedVersionPrefix, output)
	}
	versionString := strings.TrimSpace(strings.TrimPrefix(output, expectedVersionPrefix))
	if versionString == "" {
		testInstance.Errorf("Version output did not contain a version string after the prefix.\nOutput:\n%s", output)
	}
	testInstance.Logf("Detected version: %s", versionString)
	outputFirstArg := runCommand(testInstance, binaryPath, []string{"--version", appTypes.CommandContent}, testDirectory)
	if !strings.HasPrefix(outputFirstArg, expectedVersionPrefix) {
		testInstance.Errorf("Expected version output to start with '%s' when used as first argument, got:\n%s", expectedVersionPrefix, outputFirstArg)
	}
}
