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

	appTypes "github.com/temirov/ctx/types"
)

// #nosec G204
func buildBinary(testSetup *testing.T) string {
	testSetup.Helper()
	temporaryDirectory := testSetup.TempDir()
	// Use the new name for the test binary
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

	// Log warnings if present, but don't fail the test for them
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

	// Capture combined output for error messages
	var combinedOutputBuffer bytes.Buffer
	command.Stdout = &combinedOutputBuffer
	command.Stderr = &combinedOutputBuffer

	runError := command.Run()
	outputString := combinedOutputBuffer.String()

	if runError == nil {
		testSetup.Fatalf("Command succeeded unexpectedly.\n--- Command ---\n%s %s\n--- Working Directory ---\n%s\n--- Combined Output ---\n%s",
			filepath.Base(binaryPath), strings.Join(arguments, " "), workingDirectory, outputString)
	}

	// Check if it's an ExitError, which is expected for validation failures etc.
	_, isExitError := runError.(*exec.ExitError)
	if !isExitError {
		// Log if it's a different kind of error (e.g., command not found)
		testSetup.Logf("Command failed with non-ExitError type (%T): %v\n--- Command ---\n%s %s\n--- Working Directory ---\n%s",
			runError, runError, filepath.Base(binaryPath), strings.Join(arguments, " "), workingDirectory)
	}

	// Return the combined output (stdout + stderr) which usually contains the error message
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
		exitError, isExitError := runError.(*exec.ExitError)
		errorDetails := fmt.Sprintf("Command failed unexpectedly when warnings were expected.\n%s", combinedOutputForLog)
		if isExitError {
			errorDetails += fmt.Sprintf("\n--- Exit Code ---\n%d", exitError.ExitCode())
		} else {
			errorDetails += fmt.Sprintf("\n--- Error Type ---\n%T", runError)
		}
		errorDetails += fmt.Sprintf("\n--- Error ---\n%v", runError)
		testSetup.Fatalf(errorDetails)
	}

	// Check specifically for "Warning:" in stderr
	if !strings.Contains(standardErrorString, "Warning:") {
		testSetup.Fatalf("Command succeeded but did not produce expected warnings on stderr.\n%s", combinedOutputForLog)
	}

	// Return only stdout, as stderr contains the warnings
	return standardOutputString
}

func setupTestDirectory(testSetup *testing.T, directoryStructure map[string]string) string {
	testSetup.Helper()
	temporaryDirectoryRoot := testSetup.TempDir()

	for relativePath, content := range directoryStructure {
		absolutePath := filepath.Join(temporaryDirectoryRoot, relativePath)
		directoryPath := filepath.Dir(absolutePath)

		// Create parent directories if they don't exist
		mkdirErr := os.MkdirAll(directoryPath, 0755) // Use 0755 for directories
		if mkdirErr != nil {
			testSetup.Fatalf("Failed to create directory %s: %v", directoryPath, mkdirErr)
		}

		// Handle file/directory creation based on content
		if strings.HasSuffix(relativePath, "/") || content == "" { // Treat trailing slash or empty content as directory
			// Check if it already exists (MkdirAll might have created it)
			if _, statErr := os.Stat(absolutePath); os.IsNotExist(statErr) {
				mkdirErr = os.Mkdir(absolutePath, 0755)
				if mkdirErr != nil {
					testSetup.Fatalf("Failed to create directory %s: %v", absolutePath, mkdirErr)
				}
			}
		} else if content == "<UNREADABLE>" {
			// Create an unreadable file (best effort)
			initialContent := []byte("content should be unreadable")
			writeErr := os.WriteFile(absolutePath, initialContent, 0644) // Write with readable perms first
			if writeErr != nil {
				testSetup.Fatalf("Failed to write initial content for unreadable file %s: %v", absolutePath, writeErr)
			}
			chmodErr := os.Chmod(absolutePath, 0000) // Set permissions to unreadable
			if chmodErr != nil {
				// On some systems (like Windows), this might not work as expected. Log a warning.
				testSetup.Logf("Warning: Failed to set file %s permissions to 0000: %v. Test validity might be affected.", absolutePath, chmodErr)
			}
		} else {
			// Create a regular file with content
			writeErr := os.WriteFile(absolutePath, []byte(content), 0644) // Use 0644 for files
			if writeErr != nil {
				testSetup.Fatalf("Failed to write file %s: %v", absolutePath, writeErr)
			}
		}
	}
	return temporaryDirectoryRoot
}

// Helper to get the module root directory (where go.mod is)
func getModuleRoot(testSetup *testing.T) string {
	testSetup.Helper()
	currentDirectory, err := os.Getwd()
	if err != nil {
		testSetup.Fatalf("Failed to get working directory: %v", err)
	}

	// Go up until we find go.mod
	dir := currentDirectory
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir // Found module root
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory without finding go.mod
			testSetup.Fatalf("Could not find go.mod in or above test directory: %s", currentDirectory)
		}
		dir = parent
	}
}

func TestRawFormat_TreeCommand_Default(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":             "File A content",
		"directoryB/":           "", // Explicitly a directory
		"directoryB/itemB1.txt": "File B1 content",
	})

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandTree, "fileA.txt", "directoryB"}, testDirectory)

	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteDirectoryBPath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB"))

	expectedFileAMarker := fmt.Sprintf("[File] %s", absoluteFileAPath)
	expectedDirectoryBHeader := fmt.Sprintf("--- Directory Tree: %s ---", absoluteDirectoryBPath)
	// Look for the structure indicating itemB1.txt is under directoryB
	expectedItemB1TreeLine := "└── itemB1.txt" // Assuming it's the last/only item

	if !strings.Contains(output, expectedFileAMarker) {
		testInstance.Errorf("Raw tree output missing expected file marker:\nExpected fragment: %s\nActual Output:\n%s", expectedFileAMarker, output)
	}
	if !strings.Contains(output, expectedDirectoryBHeader) {
		testInstance.Errorf("Raw tree output missing expected directory header:\nExpected fragment: %s\nActual Output:\n%s", expectedDirectoryBHeader, output)
	}
	dirBTreeSectionIndex := strings.Index(output, expectedDirectoryBHeader)
	if dirBTreeSectionIndex == -1 {
		testInstance.Fatalf("Directory B header not found in output.")
	}
	dirBTreeSection := output[dirBTreeSectionIndex:]
	if !strings.Contains(dirBTreeSection, expectedItemB1TreeLine) {
		testInstance.Errorf("Raw tree output for directoryB missing expected child tree line '%s'.\nActual Output Section:\n%s", expectedItemB1TreeLine, dirBTreeSection)
	}
}

func TestRawFormat_ContentCommand_Default(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":             "Content A",
		"directoryB/":           "",
		"directoryB/itemB1.txt": "Content B1",
	})

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandContent, "fileA.txt", "directoryB"}, testDirectory)

	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteItemB1Path, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", "itemB1.txt"))

	expectedHeaderA := fmt.Sprintf("File: %s", absoluteFileAPath)
	expectedContentA := "Content A"
	expectedFooterA := fmt.Sprintf("End of file: %s", absoluteFileAPath)
	expectedHeaderB1 := fmt.Sprintf("File: %s", absoluteItemB1Path)
	expectedContentB1 := "Content B1"
	expectedFooterB1 := fmt.Sprintf("End of file: %s", absoluteItemB1Path)
	expectedSeparator := "----------------------------------------"

	// Check for file A block
	if !strings.Contains(output, expectedHeaderA) || !strings.Contains(output, expectedContentA) || !strings.Contains(output, expectedFooterA) {
		testInstance.Errorf("Raw content output missing or incorrect for fileA.\nExpected Header: %s\nExpected Content: %s\nExpected Footer: %s\nOutput:\n%s", expectedHeaderA, expectedContentA, expectedFooterA, output)
	}
	// Check for item B1 block
	if !strings.Contains(output, expectedHeaderB1) || !strings.Contains(output, expectedContentB1) || !strings.Contains(output, expectedFooterB1) {
		testInstance.Errorf("Raw content output missing or incorrect for itemB1.\nExpected Header: %s\nExpected Content: %s\nExpected Footer: %s\nOutput:\n%s", expectedHeaderB1, expectedContentB1, expectedFooterB1, output)
	}

	// Check separator count
	if strings.Count(output, expectedSeparator) != 2 {
		testInstance.Errorf("Expected 2 separators in raw content output, found %d.\nOutput:\n%s", strings.Count(output, expectedSeparator), output)
	}

	// Check order (simple index check)
	indexOfHeaderA := strings.Index(output, expectedHeaderA)
	indexOfHeaderB1 := strings.Index(output, expectedHeaderB1)
	if indexOfHeaderA == -1 || indexOfHeaderB1 == -1 || indexOfHeaderA > indexOfHeaderB1 {
		testInstance.Errorf("Raw content output order seems incorrect (expected fileA before itemB1).\nOutput:\n%s", output)
	}
}

func TestRawFormat_FlagExplicit(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"onlyfile.txt": "Explicit raw content",
	})

	// Test content command with explicit raw format
	outputContent := runCommand(testInstance, binaryPath, []string{appTypes.CommandContent, "onlyfile.txt", "--format", appTypes.FormatRaw}, testDirectory)

	absoluteFilePath, _ := filepath.Abs(filepath.Join(testDirectory, "onlyfile.txt"))
	expectedHeader := fmt.Sprintf("File: %s", absoluteFilePath)
	expectedContent := "Explicit raw content"
	expectedFooter := fmt.Sprintf("End of file: %s", absoluteFilePath)

	if !strings.Contains(outputContent, expectedHeader) || !strings.Contains(outputContent, expectedContent) || !strings.Contains(outputContent, expectedFooter) {
		testInstance.Errorf("Explicit --format raw did not produce expected raw content output.\nOutput:\n%s", outputContent)
	}

	// Test tree command with explicit raw format
	outputTree := runCommand(testInstance, binaryPath, []string{appTypes.CommandTree, "onlyfile.txt", "--format", appTypes.FormatRaw}, testDirectory)
	expectedTreeMarker := fmt.Sprintf("[File] %s", absoluteFilePath)
	if !strings.Contains(outputTree, expectedTreeMarker) {
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
	// Paths that should NOT be present
	absoluteIgnoreFilePath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", ".ignore"))
	absoluteIgnoredFilePath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", "ignored.txt"))

	foundA, foundB1, foundC := false, false, false
	pathsFound := make(map[string]string) // Track found paths and their content
	for _, item := range results {
		if item.Type != appTypes.NodeTypeFile {
			testInstance.Errorf("Unexpected item type '%s' in content JSON output for path %s", item.Type, item.Path)
		}
		pathsFound[item.Path] = item.Content // Store path and content

		// Check content for expected paths
		switch item.Path {
		case absoluteFileAPath:
			if item.Content != "Content A" {
				testInstance.Errorf("Content mismatch for fileA. Expected 'Content A', got '%s'", item.Content)
			}
			foundA = true
		case absoluteItemB1Path:
			if item.Content != "Content B1" {
				testInstance.Errorf("Content mismatch for itemB1. Expected 'Content B1', got '%s'", item.Content)
			}
			foundB1 = true
		case absoluteFileCPath:
			if item.Content != "Content C" {
				testInstance.Errorf("Content mismatch for fileC. Expected 'Content C', got '%s'", item.Content)
			}
			foundC = true
		}
	}

	if !foundA || !foundB1 || !foundC {
		testInstance.Errorf("Missing expected items in JSON output. Found A: %t, Found B1: %t, Found C: %t", foundA, foundB1, foundC)
	}

	// Verify ignored files are NOT present
	if _, exists := pathsFound[absoluteIgnoreFilePath]; exists {
		testInstance.Errorf(".ignore file itself was included in JSON output: %s", absoluteIgnoreFilePath)
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

	// Check fileA (Item 0)
	if results[0].Path != absoluteFileAPath || results[0].Type != appTypes.NodeTypeFile || results[0].Name != "fileA.txt" {
		testInstance.Errorf("JSON item 0 mismatch for fileA. Got: %+v", results[0])
	}
	if len(results[0].Children) != 0 {
		testInstance.Errorf("Expected fileA node to have no children, got %d", len(results[0].Children))
	}

	// Check directoryB (Item 1)
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
	// Check the only child: itemB1.txt
	childB1 := results[1].Children[0]
	expectedChildB1Path := filepath.Join(absoluteDirectoryBPath, "itemB1.txt")
	if childB1.Path != expectedChildB1Path || childB1.Name != "itemB1.txt" || childB1.Type != appTypes.NodeTypeFile || len(childB1.Children) != 0 {
		testInstance.Errorf("Child itemB1 mismatch in directoryB. Expected Path: %s, Name: itemB1.txt, Type: file. Got: %+v", expectedChildB1Path, childB1)
	}

	// Check fileC (Item 2)
	if results[2].Path != absoluteFileCPath || results[2].Type != appTypes.NodeTypeFile || results[2].Name != "fileC.txt" {
		testInstance.Errorf("JSON item 2 mismatch for fileC. Got: %+v", results[2])
	}
	if len(results[2].Children) != 0 {
		testInstance.Errorf("Expected fileC node to have no children, got %d", len(results[2].Children))
	}
}

func TestJsonFormat_Content_UnreadableFile(testInstance *testing.T) {
	if runtime.GOOS == "windows" {
		testInstance.Skip("Skipping unreadable file permission test on Windows due to potentially different permission behavior.")
	}
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"readable.txt":   "Readable content OK",
		"unreadable.txt": "<UNREADABLE>", // File setup to be unreadable
	})

	// Expect warnings, so use runCommandWithWarnings
	standardOutput := runCommandWithWarnings(testInstance, binaryPath, []string{appTypes.CommandContent, "readable.txt", "unreadable.txt", "--format", appTypes.FormatJSON}, testDirectory)

	var results []appTypes.FileOutput
	jsonUnmarshalError := json.Unmarshal([]byte(standardOutput), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON stdout: %v\nStdout:\n%s", jsonUnmarshalError, standardOutput)
	}

	expectedCount := 1 // Only readable.txt should be in the output
	if len(results) != expectedCount {
		testInstance.Fatalf("Expected %d item (readable.txt) in JSON array, got %d.\nJSON Output:\n%s", expectedCount, len(results), standardOutput)
	}

	// Verify the content of the readable file
	absoluteReadablePath, _ := filepath.Abs(filepath.Join(testDirectory, "readable.txt"))
	if results[0].Path != absoluteReadablePath || results[0].Content != "Readable content OK" || results[0].Type != appTypes.NodeTypeFile {
		testInstance.Errorf("JSON item 0 mismatch for readable.txt. Got: %+v", results[0])
	}
}

func TestInvalidFormatValue(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{"a.txt": "A"})

	// Test with content command
	outputContent := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, "a.txt", "--format", "invalid-format"}, testDirectory)
	expectedErrorFragment := "Invalid format value 'invalid-format'"
	if !strings.Contains(outputContent, expectedErrorFragment) {
		testInstance.Errorf("Expected error message containing '%s' for content command, but got:\n%s", expectedErrorFragment, outputContent)
	}

	// Test with tree command
	outputTree := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandTree, "a.txt", "--format", "xml"}, testDirectory)
	expectedErrorFragmentTree := "Invalid format value 'xml'"
	if !strings.Contains(outputTree, expectedErrorFragmentTree) {
		testInstance.Errorf("Expected error message containing '%s' for tree command, but got:\n%s", expectedErrorFragmentTree, outputTree)
	}

	// Test with callchain command (needs a dummy function arg)
	moduleRoot := getModuleRoot(testInstance) // callchain needs to run where go.mod is usually
	outputCallchain := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandCallChain, "dummyFunc", "--format", "yaml"}, moduleRoot)
	expectedErrorFragmentCC := "Invalid format value 'yaml'"
	if !strings.Contains(outputCallchain, expectedErrorFragmentCC) {
		// Note: This might fail first on the dummyFunc not found if validation happens later.
		// The primary check is that an invalid format is rejected.
		testInstance.Errorf("Expected error message containing '%s' for callchain command, but got:\n%s", expectedErrorFragmentCC, outputCallchain)
	}
}

func TestInput_NonExistentFile(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{}) // Empty directory

	nonExistentFileName := "non_existent_file.abc"
	// Test with content command
	outputContent := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, nonExistentFileName}, testDirectory)
	expectedErrorFragment1 := fmt.Sprintf("path '%s'", nonExistentFileName) // Check specific path format
	expectedErrorFragment2 := "does not exist"
	if !strings.Contains(outputContent, expectedErrorFragment1) || !strings.Contains(outputContent, expectedErrorFragment2) {
		testInstance.Errorf("Expected content error message containing '%s' and '%s', but got:\n%s", expectedErrorFragment1, expectedErrorFragment2, outputContent)
	}

	// Test with tree command
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

	// This should fail for tree/content commands
	outputContent := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, "dir1", "--format", appTypes.FormatJSON, "dir2"}, testDirectory)
	expectedErrorFragment := "Positional argument 'dir2' found after flags"
	if !strings.Contains(outputContent, expectedErrorFragment) {
		testInstance.Errorf("Expected error message containing '%s' for content command, got:\n%s", expectedErrorFragment, outputContent)
	}

	outputTree := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandTree, "dir1", "--no-ignore", "dir2"}, testDirectory)
	if !strings.Contains(outputTree, expectedErrorFragment) { // Same error message expected
		testInstance.Errorf("Expected error message containing '%s' for tree command, got:\n%s", expectedErrorFragment, outputTree)
	}

	// This *should* be okay for callchain as the function name is the last arg
	moduleRoot := getModuleRoot(testInstance)
	// Expect error because function won't be found, NOT because of arg order
	outputCallchain := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandCallChain, "--format", appTypes.FormatJSON, "nonExistentFunctionForTest"}, moduleRoot)
	if strings.Contains(outputCallchain, "Positional argument") && strings.Contains(outputCallchain, "found after flags") {
		testInstance.Errorf("Callchain command incorrectly rejected positional argument after flag.\nOutput:\n%s", outputCallchain)
	}
	// Check it failed for the right reason (function not found)
	if !strings.Contains(outputCallchain, "target function") || !strings.Contains(outputCallchain, "not found") {
		testInstance.Errorf("Callchain command failed, but not with the expected 'function not found' error.\nOutput:\n%s", outputCallchain)
	} else {
		testInstance.Logf("Callchain command failed as expected (function not found), not due to arg parsing.\nOutput:\n%s", outputCallchain)
	}
}

func TestCallChain_Raw(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	moduleRoot := getModuleRoot(testInstance) // Run command from module root

	// Use a function from the actual codebase with the CORRECT path
	targetFunction := "github.com/temirov/ctx/commands.GetContentData"
	expectedCaller := "github.com/temirov/ctx.runTreeOrContentCommand" // Corrected expected caller path

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandCallChain, targetFunction, "--format", appTypes.FormatRaw}, moduleRoot)

	// Basic structure checks
	if !strings.Contains(output, "----- CALLCHAIN METADATA -----") {
		testInstance.Errorf("Callchain raw output missing metadata header.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "Target Function: "+targetFunction) { // Check exact target function
		testInstance.Errorf("Callchain raw output missing correct target function identifier. Expected '%s'.\nOutput:\n%s", targetFunction, output)
	}
	if !strings.Contains(output, "Callers:") {
		testInstance.Errorf("Callchain raw output missing 'Callers:' section.\nOutput:\n%s", output)
	}
	// Check for the specific known caller with CORRECT path
	if !strings.Contains(output, expectedCaller) {
		testInstance.Errorf("Callchain raw output missing expected caller '%s'.\nOutput:\n%s", expectedCaller, output)
	}
	if !strings.Contains(output, "Callees:") {
		testInstance.Errorf("Callchain raw output missing 'Callees:' section.\nOutput:\n%s", output)
	}
	// Check for a known callee (adjust if GetContentData changes)
	if !strings.Contains(output, "path/filepath.Abs") && !strings.Contains(output, "path/filepath.WalkDir") { // Check common callees
		testInstance.Logf("Callchain raw output might be missing common callees like filepath.Abs or filepath.WalkDir (could be indirect).\nOutput:\n%s", output)
	}

	if !strings.Contains(output, "----- FUNCTIONS -----") {
		testInstance.Errorf("Callchain raw output missing functions header.\nOutput:\n%s", output)
	}
	// Check if the source code for the target function is present
	if !strings.Contains(output, "func GetContentData(rootDirectory string, ignorePatterns []string) ([]types.FileOutput, error)") {
		testInstance.Errorf("Callchain raw output missing source code signature for GetContentData.\nOutput:\n%s", output)
	}
	// Check if source code for the caller is present
	if !strings.Contains(output, "func runTreeOrContentCommand(commandName string, inputArguments []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) error") {
		testInstance.Errorf("Callchain raw output missing source code signature for caller '%s'.\nOutput:\n%s", expectedCaller, output)
	}
}

func TestCallChain_JSON(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	moduleRoot := getModuleRoot(testInstance)

	// Use CORRECT paths
	targetFunctionInput := "github.com/temirov/ctx/commands.GetContentData"
	expectedCaller := "github.com/temirov/ctx.runTreeOrContentCommand"

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandCallChain, targetFunctionInput, "--format", appTypes.FormatJSON}, moduleRoot)

	var result appTypes.CallChainOutput
	jsonUnmarshalError := json.Unmarshal([]byte(output), &result)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal Callchain JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}

	// Check target function
	if result.TargetFunction != targetFunctionInput {
		testInstance.Errorf("JSON output TargetFunction mismatch. Expected '%s', got '%s'", targetFunctionInput, result.TargetFunction)
	}

	// Check callers using CORRECT path
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

	// Check callees (presence check is usually sufficient unless specific callees are critical)
	if result.Callees == nil || len(*result.Callees) == 0 {
		testInstance.Logf("JSON output Callees list is empty or nil for GetContentData. This might be expected.")
	}

	// Check functions map
	if len(result.Functions) == 0 {
		testInstance.Errorf("JSON output expected Functions map to be non-empty, but it was.")
	}
	// Check if target function source is present
	targetFunctionResolved := result.TargetFunction // Use the resolved name from the output
	if _, ok := result.Functions[targetFunctionResolved]; !ok {
		testInstance.Errorf("JSON output Functions map missing entry for resolved target function '%s'", targetFunctionResolved)
	} else if !strings.Contains(result.Functions[targetFunctionResolved], "func GetContentData") {
		testInstance.Errorf("JSON output Functions map entry for target function '%s' doesn't seem to contain its source code.", targetFunctionResolved)
	}
	// Check if caller function source is present using CORRECT path
	if _, ok := result.Functions[expectedCaller]; !ok {
		testInstance.Errorf("JSON output Functions map missing entry for expected caller '%s'", expectedCaller)
	} else if !strings.Contains(result.Functions[expectedCaller], "func runTreeOrContentCommand") {
		testInstance.Errorf("JSON output Functions map entry for caller '%s' doesn't seem to contain its source code.", expectedCaller)
	}
}

func TestVersionFlag(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{}) // Doesn't matter for version flag

	output := runCommand(testInstance, binaryPath, []string{"--version"}, testDirectory)

	// Use CORRECT prefix
	expectedVersionPrefix := "ctx version:"
	if !strings.HasPrefix(output, expectedVersionPrefix) {
		testInstance.Errorf("Expected version output to start with '%s', got:\n%s", expectedVersionPrefix, output)
	}
	// Check that *some* version string follows the prefix
	versionString := strings.TrimSpace(strings.TrimPrefix(output, expectedVersionPrefix))
	if versionString == "" {
		testInstance.Errorf("Version output did not contain a version string after the prefix.\nOutput:\n%s", output)
	}
	// It's hard to check the exact version without knowing build context, so just log it
	testInstance.Logf("Detected version: %s", versionString)

	// Test with version flag as first argument
	outputFirstArg := runCommand(testInstance, binaryPath, []string{"--version", appTypes.CommandContent}, testDirectory)
	if !strings.HasPrefix(outputFirstArg, expectedVersionPrefix) {
		testInstance.Errorf("Expected version output to start with '%s' when used as first arg, got:\n%s", expectedVersionPrefix, outputFirstArg)
	}
}
