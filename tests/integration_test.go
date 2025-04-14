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

	appTypes "github.com/temirov/content/types"
)

// #nosec G204
func buildBinary(testSetup *testing.T) string {
	testSetup.Helper()
	temporaryDirectory := testSetup.TempDir()
	binaryName := "content_integration_test_binary"
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

		switch content {
		case "":
			if _, statErr := os.Stat(absolutePath); os.IsNotExist(statErr) {
				mkdirErr = os.Mkdir(absolutePath, 0755)
				if mkdirErr != nil {
					testSetup.Fatalf("Failed to create directory %s: %v", absolutePath, mkdirErr)
				}
			}
		case "<UNREADABLE>":
			initialContent := []byte("content should be unreadable")
			writeErr := os.WriteFile(absolutePath, initialContent, 0644)
			if writeErr != nil {
				testSetup.Fatalf("Failed to write initial content for unreadable file %s: %v", absolutePath, writeErr)
			}
			chmodErr := os.Chmod(absolutePath, 0000)
			if chmodErr != nil {
				testSetup.Logf("Warning: Failed to set file %s permissions to 0000: %v", absolutePath, chmodErr)
			}
		default:
			writeErr := os.WriteFile(absolutePath, []byte(content), 0644)
			if writeErr != nil {
				testSetup.Fatalf("Failed to write file %s: %v", absolutePath, writeErr)
			}
		}
	}
	return temporaryDirectoryRoot
}

func TestRawFormat_TreeCommand_Default(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":             "File A content",
		"directoryB/":           "",
		"directoryB/itemB1.txt": "File B1 content",
	})

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandTree, "fileA.txt", "directoryB"}, testDirectory)

	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteDirectoryBPath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB"))

	expectedFileAMarker := fmt.Sprintf("[File] %s", absoluteFileAPath)
	expectedDirectoryBHeader := fmt.Sprintf("--- Directory Tree: %s ---", absoluteDirectoryBPath)
	expectedItemB1TreeLine := "itemB1.txt"

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
		testInstance.Errorf("Raw tree output for directoryB missing expected child item '%s'.\nActual Output Section:\n%s", expectedItemB1TreeLine, dirBTreeSection)
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
	expectedFooterA := fmt.Sprintf("End of file: %s", absoluteFileAPath)
	expectedHeaderB1 := fmt.Sprintf("File: %s", absoluteItemB1Path)
	expectedFooterB1 := fmt.Sprintf("End of file: %s", absoluteItemB1Path)
	expectedSeparator := "----------------------------------------"

	if !strings.Contains(output, expectedHeaderA) {
		testInstance.Errorf("Raw content output missing header for fileA.\nExpected: %s\nOutput:\n%s", expectedHeaderA, output)
	}
	if !strings.Contains(output, "Content A") {
		testInstance.Errorf("Raw content output missing content for fileA.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, expectedFooterA) {
		testInstance.Errorf("Raw content output missing footer for fileA.\nExpected: %s\nOutput:\n%s", expectedFooterA, output)
	}

	if !strings.Contains(output, expectedHeaderB1) {
		testInstance.Errorf("Raw content output missing header for itemB1.\nExpected: %s\nOutput:\n%s", expectedHeaderB1, output)
	}
	if !strings.Contains(output, "Content B1") {
		testInstance.Errorf("Raw content output missing content for itemB1.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, expectedFooterB1) {
		testInstance.Errorf("Raw content output missing footer for itemB1.\nExpected: %s\nOutput:\n%s", expectedFooterB1, output)
	}

	if strings.Count(output, expectedSeparator) != 2 {
		testInstance.Errorf("Expected 2 separators in raw content output, found %d.\nOutput:\n%s", strings.Count(output, expectedSeparator), output)
	}
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

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandContent, "onlyfile.txt", "--format", appTypes.FormatRaw}, testDirectory)

	absoluteFilePath, _ := filepath.Abs(filepath.Join(testDirectory, "onlyfile.txt"))
	expectedHeader := fmt.Sprintf("File: %s", absoluteFilePath)
	expectedContent := "Explicit raw content"

	if !strings.Contains(output, expectedHeader) || !strings.Contains(output, expectedContent) {
		testInstance.Errorf("Explicit --format raw did not produce expected raw content output.\nOutput:\n%s", output)
	}
}

func TestJsonFormat_ContentCommand_Mixed(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":              "Content A",
		"directoryB/":            "",
		"directoryB/.ignore":     "ignored.txt",
		"directoryB/itemB1.txt":  "Content B1",
		"directoryB/ignored.txt": "Ignored Content B",
		"fileC.txt":              "Content C",
	})

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandContent, "fileA.txt", "directoryB", "fileC.txt", "--format", appTypes.FormatJSON}, testDirectory)

	var results []appTypes.FileOutput
	jsonUnmarshalError := json.Unmarshal([]byte(output), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}

	expectedCount := 3
	if len(results) != expectedCount {
		testInstance.Fatalf("Expected %d items in JSON array, got %d.\nOutput:\n%s", expectedCount, len(results), output)
	}

	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteItemB1Path, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB", "itemB1.txt"))
	absoluteFileCPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileC.txt"))
	absoluteIgnoreFilePath := filepath.Join(testDirectory, "directoryB", ".ignore")
	absoluteIgnoredFilePath := filepath.Join(testDirectory, "directoryB", "ignored.txt")

	foundA, foundB1, foundC := false, false, false
	for _, item := range results {
		if item.Type != appTypes.NodeTypeFile {
			testInstance.Errorf("Unexpected item type '%s' in content JSON output for path %s", item.Type, item.Path)
		}
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
		case absoluteIgnoreFilePath:
			testInstance.Errorf(".ignore file itself was included in JSON output: %s", item.Path)
		case absoluteIgnoredFilePath:
			testInstance.Errorf("Ignored file 'ignored.txt' was included in JSON output: %s", item.Path)
		}
	}

	if !foundA || !foundB1 || !foundC {
		testInstance.Errorf("Missing expected items in JSON output. Found A: %t, Found B1: %t, Found C: %t", foundA, foundB1, foundC)
	}
}

func TestJsonFormat_TreeCommand_Mixed(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{
		"fileA.txt":                 "Content A",
		"directoryB/":               "",
		"directoryB/.ignore":        "ignored.txt\nsub/",
		"directoryB/itemB1.txt":     "Content B1",
		"directoryB/ignored.txt":    "Ignored B",
		"directoryB/sub/":           "",
		"directoryB/sub/itemB2.txt": "Content B2",
		"fileC.txt":                 "Content C",
	})

	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandTree, "fileA.txt", "directoryB", "fileC.txt", "--format", appTypes.FormatJSON}, testDirectory)

	var results []appTypes.TreeOutputNode
	jsonUnmarshalError := json.Unmarshal([]byte(output), &results)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}

	expectedTopLevelCount := 3
	if len(results) != expectedTopLevelCount {
		testInstance.Fatalf("Expected %d top-level items in JSON array, got %d.\nOutput:\n%s", expectedTopLevelCount, len(results), output)
	}

	absoluteFileAPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileA.txt"))
	absoluteDirectoryBPath, _ := filepath.Abs(filepath.Join(testDirectory, "directoryB"))
	absoluteFileCPath, _ := filepath.Abs(filepath.Join(testDirectory, "fileC.txt"))

	if results[0].Path != absoluteFileAPath || results[0].Type != appTypes.NodeTypeFile || results[0].Name != "fileA.txt" {
		testInstance.Errorf("JSON item 0 mismatch for fileA. Got: %+v", results[0])
	}
	if len(results[0].Children) != 0 {
		testInstance.Errorf("Expected fileA node to have no children, got %d", len(results[0].Children))
	}

	if results[1].Path != absoluteDirectoryBPath || results[1].Type != appTypes.NodeTypeDirectory || results[1].Name != "directoryB" {
		testInstance.Errorf("JSON item 1 mismatch for directoryB. Got: %+v", results[1])
	}
	expectedChildrenCountB := 1
	if len(results[1].Children) != expectedChildrenCountB {
		var childNames []string
		for _, child := range results[1].Children {
			childNames = append(childNames, child.Name)
		}
		testInstance.Fatalf("Expected %d children for directoryB, got %d. Children: %v\nOutput:\n%s",
			expectedChildrenCountB, len(results[1].Children), childNames, output)
	}
	childB1 := results[1].Children[0]
	if childB1.Name != "itemB1.txt" || childB1.Type != appTypes.NodeTypeFile || len(childB1.Children) != 0 {
		testInstance.Errorf("Child itemB1 mismatch in directoryB. Got: %+v", childB1)
	}

	if results[2].Path != absoluteFileCPath || results[2].Type != appTypes.NodeTypeFile || results[2].Name != "fileC.txt" {
		testInstance.Errorf("JSON item 2 mismatch for fileC. Got: %+v", results[2])
	}
	if len(results[2].Children) != 0 {
		testInstance.Errorf("Expected fileC node to have no children, got %d", len(results[2].Children))
	}
}

func TestJsonFormat_Content_UnreadableFile(testInstance *testing.T) {
	if runtime.GOOS == "windows" {
		testInstance.Skip("Skipping unreadable file permission test on Windows due to different permission models.")
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

	expectedCount := 1
	if len(results) != expectedCount {
		testInstance.Fatalf("Expected %d item (readable.txt) in JSON array, got %d.\nJSON Output:\n%s", expectedCount, len(results), standardOutput)
	}

	absoluteReadablePath, _ := filepath.Abs(filepath.Join(testDirectory, "readable.txt"))
	if results[0].Path != absoluteReadablePath || results[0].Content != "Readable content OK" || results[0].Type != appTypes.NodeTypeFile {
		testInstance.Errorf("JSON item 0 mismatch for readable.txt. Got: %+v", results[0])
	}
}

func TestInvalidFormatValue(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{"a.txt": "A"})

	output := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, "a.txt", "--format", "invalid-format"}, testDirectory)

	expectedErrorFragment := "Invalid format value 'invalid-format'"
	if !strings.Contains(output, expectedErrorFragment) {
		testInstance.Errorf("Expected error message containing '%s', but got:\n%s", expectedErrorFragment, output)
	}
}

func TestInput_NonExistentFile(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{})

	nonExistentFileName := "non_existent_file.abc"
	output := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, nonExistentFileName}, testDirectory)

	expectedErrorFragment1 := nonExistentFileName
	expectedErrorFragment2 := "does not exist"
	if !strings.Contains(output, expectedErrorFragment1) || !strings.Contains(output, expectedErrorFragment2) {
		testInstance.Errorf("Expected error message containing '%s' and '%s', but got:\n%s", expectedErrorFragment1, expectedErrorFragment2, output)
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

	output := runCommandExpectError(testInstance, binaryPath, []string{appTypes.CommandContent, "dir1", "--format", appTypes.FormatJSON, "dir2"}, testDirectory)

	expectedErrorFragment := "Positional argument 'dir2' found after flags"
	if !strings.Contains(output, expectedErrorFragment) {
		testInstance.Errorf("Expected error message containing '%s', got:\n%s", expectedErrorFragment, output)
	}
}

func getModuleRoot(testSetup *testing.T) string {
	testSetup.Helper()
	currentDirectory, err := os.Getwd()
	if err != nil {
		testSetup.Fatalf("Failed to get working directory: %v", err)
	}

	moduleRoot := filepath.Dir(currentDirectory)
	if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); os.IsNotExist(err) {
		testSetup.Fatalf("Could not find go.mod in assumed root directory: %s", moduleRoot)
	}
	return moduleRoot
}

func TestCallChain_Raw(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	moduleRoot := getModuleRoot(testInstance)

	targetFunction := "github.com/temirov/content/commands.GetContentData"
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandCallChain, targetFunction, "--format", appTypes.FormatRaw}, moduleRoot)

	if !strings.Contains(output, "----- CALLCHAIN METADATA -----") {
		testInstance.Errorf("Callchain raw output missing metadata header.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "Target Function: ") || !strings.Contains(output, "commands.GetContentData") {
		testInstance.Errorf("Callchain raw output missing correct target function identifier.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "Callers:") {
		testInstance.Errorf("Callchain raw output missing 'Callers:' section.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "github.com/temirov/content.runContentTool") {
		testInstance.Errorf("Callchain raw output missing expected caller 'runContentTool'.\nOutput:\n%s", output)
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
}

func TestCallChain_JSON(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	moduleRoot := getModuleRoot(testInstance)

	targetFunctionInput := "github.com/temirov/content/commands.GetContentData"
	output := runCommand(testInstance, binaryPath, []string{appTypes.CommandCallChain, targetFunctionInput, "--format", appTypes.FormatJSON}, moduleRoot)

	var result appTypes.CallChainOutput
	jsonUnmarshalError := json.Unmarshal([]byte(output), &result)
	if jsonUnmarshalError != nil {
		testInstance.Fatalf("Failed to unmarshal Callchain JSON output: %v\nOutput:\n%s", jsonUnmarshalError, output)
	}

	if !strings.HasSuffix(result.TargetFunction, "commands.GetContentData") {
		testInstance.Errorf("JSON output TargetFunction mismatch. Expected suffix 'commands.GetContentData', got '%s'", result.TargetFunction)
	}
	if len(result.Callers) == 0 {
		testInstance.Errorf("JSON output expected callers for GetContentData, but got none.")
	} else {
		foundCaller := false
		expectedCaller := "github.com/temirov/content.runContentTool"
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

	if len(result.Functions) == 0 {
		testInstance.Errorf("JSON output expected Functions map to be non-empty, but it was.")
	}
	targetFunctionResolved := result.TargetFunction
	if _, ok := result.Functions[targetFunctionResolved]; !ok {
		testInstance.Errorf("JSON output Functions map missing entry for resolved target function '%s'", targetFunctionResolved)
	}
	if !strings.Contains(result.Functions[targetFunctionResolved], "func GetContentData") {
		testInstance.Errorf("JSON output Functions map entry for target function doesn't seem to contain its source code.")
	}
}

func TestVersionFlag(testInstance *testing.T) {
	binaryPath := buildBinary(testInstance)
	testDirectory := setupTestDirectory(testInstance, map[string]string{})

	output := runCommand(testInstance, binaryPath, []string{"--version"}, testDirectory)

	expectedVersionPrefix := "content version:"
	if !strings.HasPrefix(output, expectedVersionPrefix) {
		testInstance.Errorf("Expected version output to start with '%s', got:\n%s", expectedVersionPrefix, output)
	}
	versionString := strings.TrimSpace(strings.TrimPrefix(output, expectedVersionPrefix))
	if versionString == "" {
		testInstance.Errorf("Version output did not contain a version string after the prefix.\nOutput:\n%s", output)
	}
	testInstance.Logf("Detected version: %s", versionString)
}
