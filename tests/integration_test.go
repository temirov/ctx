package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

	var stdOutErrBuffer bytes.Buffer
	command.Stdout = &stdOutErrBuffer
	command.Stderr = &stdOutErrBuffer

	runError := command.Run()
	outputString := stdOutErrBuffer.String()

	if runError != nil {
		exitError, isExitError := runError.(*exec.ExitError)
		errorDetails := fmt.Sprintf("Command '%s %s' failed in dir '%s'", filepath.Base(binaryPath), strings.Join(args, " "), workDir)
		if isExitError {
			errorDetails += fmt.Sprintf("\nExit Code: %d", exitError.ExitCode())
		} else {
			errorDetails += fmt.Sprintf("\nError Type: %T", runError)
		}
		errorDetails += fmt.Sprintf("\nError: %v\nOutput:\n%s", runError, outputString)
		testSetup.Fatalf(errorDetails)
	} else if strings.Contains(outputString, "Warning:") {
		testSetup.Logf("Command '%s %s' succeeded but produced warnings:\n%s", filepath.Base(binaryPath), strings.Join(args, " "), outputString)
	}

	return outputString
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

	var stdOutErrBuffer bytes.Buffer
	command.Stdout = &stdOutErrBuffer
	command.Stderr = &stdOutErrBuffer

	runError := command.Run()
	outputString := stdOutErrBuffer.String()

	if runError != nil {
		exitError, isExitError := runError.(*exec.ExitError)
		errorDetails := fmt.Sprintf("Command '%s %s' failed unexpectedly in dir '%s'", filepath.Base(binaryPath), strings.Join(args, " "), workDir)
		if isExitError {
			errorDetails += fmt.Sprintf("\nExit Code: %d", exitError.ExitCode())
		} else {
			errorDetails += fmt.Sprintf("\nError Type: %T", runError)
		}
		errorDetails += fmt.Sprintf("\nError: %v\nOutput:\n%s", runError, outputString)
		testSetup.Fatalf(errorDetails)
	}

	if !strings.Contains(outputString, "Warning:") {
		testSetup.Fatalf("Command '%s %s' succeeded but did not produce expected warnings.\nOutput:\n%s", filepath.Base(binaryPath), strings.Join(args, " "), outputString)
	}

	return outputString
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

func TestTreeCommandIntegration_NoIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"file1.txt":        "Hello",
		"subdir/":          "",
		"subdir/file2.txt": "World",
	})

	output := runCommand(testValue, binary, []string{"tree", testDir}, testDir)

	absTestDir, _ := filepath.Abs(testDir)
	expectedTreeHeader := fmt.Sprintf("--- Directory Tree: %s ---", absTestDir)

	if !strings.Contains(output, expectedTreeHeader) {
		testValue.Errorf("Tree output missing expected header '%s'.\nOutput: %s", expectedTreeHeader, output)
	}
	if !strings.Contains(output, "file1.txt") {
		testValue.Errorf("Tree output missing 'file1.txt'.\nOutput: %s", output)
	}
	if !strings.Contains(output, "subdir") {
		testValue.Errorf("Tree output missing 'subdir'.\nOutput: %s", output)
	}
	if !strings.Contains(output, "└── file2.txt") && !strings.Contains(output, "├── file2.txt") {
		testValue.Errorf("Tree output missing 'file2.txt' likely not nested correctly.\nOutput: %s", output)
	}
}

func TestContentCommandIntegration_NoIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"file1.txt": "Hello Content",
	})
	output := runCommand(testValue, binary, []string{"content", testDir}, testDir)

	expectedFileHeader := fmt.Sprintf("File: %s", filepath.Join(testDir, "file1.txt"))
	if !strings.Contains(output, expectedFileHeader) {
		testValue.Errorf("Content output missing expected file header '%s'.\nOutput: %s", expectedFileHeader, output)
	}
	if !strings.Contains(output, "Hello Content") {
		testValue.Errorf("Content output did not include 'Hello Content'.\nOutput: %s", output)
	}
}

func TestMultiDir_Content(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"dir1/":          "",
		"dir1/file1.txt": "Content from Dir1",
		"dir2/":          "",
		"dir2/file2.txt": "Content from Dir2",
	})

	output := runCommand(testValue, binary, []string{"content", "dir1", "dir2"}, testDir)

	if !strings.Contains(output, "Content from Dir1") {
		testValue.Errorf("Output missing content from dir1.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Content from Dir2") {
		testValue.Errorf("Output missing content from dir2.\nOutput: %s", output)
	}
	if strings.Index(output, "Content from Dir1") > strings.Index(output, "Content from Dir2") {
		testValue.Errorf("Output content order seems incorrect.\nOutput: %s", output)
	}
}

func TestMultiDir_Tree(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"dirA/":          "",
		"dirA/itemA.txt": "A",
		"dirB/":          "",
		"dirB/itemB.txt": "B",
	})

	output := runCommand(testValue, binary, []string{"tree", "dirA", "dirB"}, testDir)

	absDirA, _ := filepath.Abs(filepath.Join(testDir, "dirA"))
	absDirB, _ := filepath.Abs(filepath.Join(testDir, "dirB"))
	expectedHeaderA := fmt.Sprintf("--- Directory Tree: %s ---", absDirA)
	expectedHeaderB := fmt.Sprintf("--- Directory Tree: %s ---", absDirB)

	if !strings.Contains(output, expectedHeaderA) {
		testValue.Errorf("Output missing tree header for dirA: %s\nOutput: %s", expectedHeaderA, output)
	}
	if !strings.Contains(output, expectedHeaderB) {
		testValue.Errorf("Output missing tree header for dirB: %s\nOutput: %s", expectedHeaderB, output)
	}
	if !strings.Contains(output, "itemA.txt") {
		testValue.Errorf("Output missing itemA.txt from dirA tree.\nOutput: %s", output)
	}
	if !strings.Contains(output, "itemB.txt") {
		testValue.Errorf("Output missing itemB.txt from dirB tree.\nOutput: %s", output)
	}
	if strings.Index(output, expectedHeaderA) > strings.Index(output, expectedHeaderB) {
		testValue.Errorf("Tree output order seems incorrect.\nOutput: %s", output)
	}
	if !(strings.Index(output, "itemA.txt") > strings.Index(output, expectedHeaderA) && strings.Index(output, "itemA.txt") < strings.Index(output, expectedHeaderB)) {
		testValue.Errorf("itemA.txt does not appear under headerA and before headerB.\nOutput: %s", output)
	}
}

func TestMixedInput_Tree(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"fileA.txt":           "Content A",
		"dirB/":               "",
		"dirB/itemB1.txt":     "B1",
		"dirB/sub/":           "",
		"dirB/sub/itemB2.txt": "B2",
		"fileC.txt":           "Content C",
	})

	output := runCommand(testValue, binary, []string{"tree", "fileA.txt", "dirB", "fileC.txt"}, testDir)

	absFileA, _ := filepath.Abs(filepath.Join(testDir, "fileA.txt"))
	absDirB, _ := filepath.Abs(filepath.Join(testDir, "dirB"))
	absFileC, _ := filepath.Abs(filepath.Join(testDir, "fileC.txt"))

	expectedFileA := fmt.Sprintf("[File] %s", absFileA)
	expectedDirBHeader := fmt.Sprintf("--- Directory Tree: %s ---", absDirB)
	expectedFileC := fmt.Sprintf("[File] %s", absFileC)

	if !strings.Contains(output, expectedFileA) {
		testValue.Errorf("Output missing file marker for fileA: %s\nOutput: %s", expectedFileA, output)
	}
	if !strings.Contains(output, expectedDirBHeader) {
		testValue.Errorf("Output missing tree header for dirB: %s\nOutput: %s", expectedDirBHeader, output)
	}
	if !strings.Contains(output, "itemB1.txt") {
		testValue.Errorf("Output missing itemB1.txt from dirB tree.\nOutput: %s", output)
	}
	if !strings.Contains(output, "sub") {
		testValue.Errorf("Output missing sub dir from dirB tree.\nOutput: %s", output)
	}
	if !strings.Contains(output, "itemB2.txt") {
		testValue.Errorf("Output missing itemB2.txt from dirB tree.\nOutput: %s", output)
	}
	if !strings.Contains(output, expectedFileC) {
		testValue.Errorf("Output missing file marker for fileC: %s\nOutput: %s", expectedFileC, output)
	}

	idxA := strings.Index(output, expectedFileA)
	idxBHeader := strings.Index(output, expectedDirBHeader)
	idxC := strings.Index(output, expectedFileC)

	if !(idxA < idxBHeader && idxBHeader < idxC) {
		testValue.Errorf("Output order incorrect. Indices: FileA=%d, DirBHeader=%d, FileC=%d\nOutput: %s", idxA, idxBHeader, idxC, output)
	}
}

func TestMixedInput_Content(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"fileA.txt":           "Content A",
		"dirB/":               "",
		"dirB/itemB1.txt":     "Content B1",
		"dirB/sub/":           "",
		"dirB/sub/itemB2.txt": "Content B2",
		"fileC.txt":           "Content C",
	})

	output := runCommand(testValue, binary, []string{"content", "fileA.txt", "dirB", "fileC.txt"}, testDir)

	absFileA, _ := filepath.Abs(filepath.Join(testDir, "fileA.txt"))
	absFileB1, _ := filepath.Abs(filepath.Join(testDir, "dirB", "itemB1.txt"))
	absFileB2, _ := filepath.Abs(filepath.Join(testDir, "dirB", "sub", "itemB2.txt"))
	absFileC, _ := filepath.Abs(filepath.Join(testDir, "fileC.txt"))

	expectedHeaderA := fmt.Sprintf("File: %s", absFileA)
	expectedHeaderB1 := fmt.Sprintf("File: %s", absFileB1)
	expectedHeaderB2 := fmt.Sprintf("File: %s", absFileB2)
	expectedHeaderC := fmt.Sprintf("File: %s", absFileC)

	if !strings.Contains(output, "Content A") {
		testValue.Errorf("Output missing content from fileA.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Content B1") {
		testValue.Errorf("Output missing content from dirB/itemB1.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Content B2") {
		testValue.Errorf("Output missing content from dirB/sub/itemB2.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Content C") {
		testValue.Errorf("Output missing content from fileC.\nOutput: %s", output)
	}

	if !strings.Contains(output, expectedHeaderA) {
		testValue.Errorf("Output missing header for fileA.\nOutput: %s", output)
	}
	if !strings.Contains(output, expectedHeaderB1) {
		testValue.Errorf("Output missing header for itemB1.\nOutput: %s", output)
	}
	if !strings.Contains(output, expectedHeaderB2) {
		testValue.Errorf("Output missing header for itemB2.\nOutput: %s", output)
	}
	if !strings.Contains(output, expectedHeaderC) {
		testValue.Errorf("Output missing header for fileC.\nOutput: %s", output)
	}

	idxA := strings.Index(output, expectedHeaderA)
	idxB1 := strings.Index(output, expectedHeaderB1)
	idxB2 := strings.Index(output, expectedHeaderB2)
	idxC := strings.Index(output, expectedHeaderC)

	if !(idxA >= 0 && idxA < idxB1 && idxA < idxB2 && (idxB1 < idxC || idxB2 < idxC) && idxC >= 0) {
		testValue.Errorf("Output order incorrect. Indices: HeaderA=%d, HeaderB1=%d, HeaderB2=%d, HeaderC=%d\nOutput: %s", idxA, idxB1, idxB2, idxC, output)
	}
}

func TestMixedInput_IgnoreScope_Content(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"my_dir/":                   "",
		"my_dir/.ignore":            "*.log\nignored_in_dir.txt",
		"my_dir/app.log":            "Log content (ignored in dir)",
		"my_dir/ignored_in_dir.txt": "Ignored text (ignored in dir)",
		"my_dir/explicit.log":       "Explicit log content",
		"my_dir/kept.txt":           "Kept text",
	})

	output := runCommand(testValue, binary, []string{"content", "my_dir/explicit.log", "my_dir"}, testDir)

	if !strings.Contains(output, "Explicit log content") {
		testValue.Errorf("Explicitly listed file 'my_dir/explicit.log' was incorrectly ignored.\nOutput: %s", output)
	}
	absExplicitLog, _ := filepath.Abs(filepath.Join(testDir, "my_dir/explicit.log"))
	if !strings.Contains(output, fmt.Sprintf("File: %s", absExplicitLog)) {
		testValue.Errorf("Missing header for explicitly listed file 'my_dir/explicit.log'.\nOutput: %s", output)
	}

	if strings.Contains(output, "Log content (ignored in dir)") {
		testValue.Errorf("File 'my_dir/app.log' should have been ignored during directory traversal.\nOutput: %s", output)
	}
	if strings.Contains(output, "Ignored text (ignored in dir)") {
		testValue.Errorf("File 'my_dir/ignored_in_dir.txt' should have been ignored during directory traversal.\nOutput: %s", output)
	}

	if !strings.Contains(output, "Kept text") {
		testValue.Errorf("File 'my_dir/kept.txt' should have been included during directory traversal.\nOutput: %s", output)
	}
}

func TestMixedInput_IgnoreScope_Tree(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"my_dir/":                    "",
		"my_dir/.ignore":             "ignored_in_tree.txt",
		"my_dir/ignored_in_tree.txt": "ignored",
		"my_dir/shown_in_tree.txt":   "shown",
	})

	output := runCommand(testValue, binary, []string{"tree", "my_dir/ignored_in_tree.txt", "my_dir"}, testDir)

	absIgnoredFile, _ := filepath.Abs(filepath.Join(testDir, "my_dir/ignored_in_tree.txt"))
	absMyDir, _ := filepath.Abs(filepath.Join(testDir, "my_dir"))

	expectedFileMarker := fmt.Sprintf("[File] %s", absIgnoredFile)
	expectedDirHeader := fmt.Sprintf("--- Directory Tree: %s ---", absMyDir)

	if !strings.Contains(output, expectedFileMarker) {
		testValue.Errorf("Output missing file marker for explicitly listed 'ignored_in_tree.txt': %s\nOutput: %s", expectedFileMarker, output)
	}

	if !strings.Contains(output, expectedDirHeader) {
		testValue.Errorf("Output missing directory tree header for 'my_dir': %s\nOutput: %s", expectedDirHeader, output)
	}

	treePartStartIndex := strings.Index(output, expectedDirHeader)
	if treePartStartIndex == -1 {
		testValue.Fatalf("Could not find start of directory tree output.")
	}
	treePart := output[treePartStartIndex:]

	if strings.Contains(treePart, "ignored_in_tree.txt") {
		testValue.Errorf("The tree view for 'my_dir' should have omitted 'ignored_in_tree.txt'.\nTree Part:\n%s", treePart)
	}
	if !strings.Contains(treePart, "shown_in_tree.txt") {
		testValue.Errorf("The tree view for 'my_dir' should have included 'shown_in_tree.txt'.\nTree Part:\n%s", treePart)
	}
}

func TestMixedInput_EFlagScope(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"config.log":      "Explicit top-level log",
		"src/":            "",
		"src/main.go":     "Go source",
		"src/log/":        "",
		"src/log/app.log": "App log inside src/log",
		"vendor/":         "",
		"vendor/lib.go":   "Vendor lib",
	})

	output := runCommand(testValue, binary, []string{"content", "config.log", "src", "-e", "log"}, testDir)

	if !strings.Contains(output, "Explicit top-level log") {
		testValue.Errorf("Explicitly listed file 'config.log' was incorrectly ignored by -e.\nOutput: %s", output)
	}

	if !strings.Contains(output, "Go source") {
		testValue.Errorf("File 'src/main.go' should have been included during directory traversal.\nOutput: %s", output)
	}

	if strings.Contains(output, "App log inside src/log") {
		testValue.Errorf("File 'src/log/app.log' should have been excluded by -e=log during directory traversal.\nOutput: %s", output)
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

func TestInput_NonExistentDir(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{})

	output := runCommandExpectError(testValue, binary, []string{"content", "no_such_dir/"}, testDir)

	if !strings.Contains(output, "no_such_dir") || !strings.Contains(output, "does not exist") {
		testValue.Errorf("Expected error about non-existent directory, got:\n%s", output)
	}
}

func TestInput_UnreadableFile_Content(testValue *testing.T) {
	if runtime.GOOS == "windows" {
		testValue.Skip("Skipping unreadable file test on Windows")
	}

	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"readable.txt":   "Readable content",
		"unreadable.txt": "<UNREADABLE>",
	})

	output := runCommandWithWarnings(testValue, binary, []string{"content", "readable.txt", "unreadable.txt"}, testDir)

	if !strings.Contains(output, "Readable content") {
		testValue.Errorf("Readable content missing.\nOutput: %s", output)
	}

	absUnreadable, _ := filepath.Abs(filepath.Join(testDir, "unreadable.txt"))
	expectedWarning := fmt.Sprintf("Warning: Failed to read file %s", absUnreadable)
	if !strings.Contains(output, expectedWarning) {
		testValue.Errorf("Expected warning about unreadable file '%s' not found.\nOutput: %s", absUnreadable, output)
	}

	if strings.Contains(output, "cannot read this") {
		testValue.Errorf("Unreadable content should not be present.\nOutput: %s", output)
	}

	warningIndex := strings.Index(output, expectedWarning)
	separatorIndex := strings.Index(output[warningIndex:], "----------------------------------------")
	if warningIndex == -1 || separatorIndex == -1 {
		testValue.Errorf("Separator '---' not found after unreadable file warning.\nOutput: %s", output)
	}
}

func TestExclusionFlagRootVsNested(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"log/":                   "",
		"log/ignore.txt":         "top log ignored",
		"pkg/":                   "",
		"pkg/log/":               "",
		"pkg/log/nested_log.txt": "nested log not ignored by -e",
		"pkg/data.txt":           "pkg data",
	})

	output := runCommand(testValue, binary, []string{"content", ".", "-e", "log"}, testDir)

	if strings.Contains(output, "top log ignored") {
		testValue.Errorf("Top-level log folder should be excluded by -e=log.\nOutput: %s", output)
	}
	if !strings.Contains(output, "nested log not ignored by -e") {
		testValue.Errorf("Nested log folder should *not* be excluded by -e=log.\nOutput: %s", output)
	}
	if !strings.Contains(output, "pkg data") {
		testValue.Errorf("Content from pkg/data.txt should be included.\nOutput: %s", output)
	}
}

func TestMultiDir_NoDirsProvided(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"local_file.txt": "Local Content",
		"sub/":           "",
		"sub/nested.txt": "Nested Content",
	})

	output := runCommand(testValue, binary, []string{"content"}, testDir)

	if !strings.Contains(output, "Local Content") {
		testValue.Errorf("Output missing 'Local Content' when run in current dir.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Nested Content") {
		testValue.Errorf("Output missing 'Nested Content' when run in current dir.\nOutput: %s", output)
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

	output := runCommandExpectError(testValue, binary, []string{"content", "dir1", "-e", "log", "dir2"}, testDir)
	if !strings.Contains(output, "Positional argument 'dir2' found after flags") {
		testValue.Errorf("Expected error about positional arg after flags, got:\n%s", output)
	}

	validOutput := runCommand(testValue, binary, []string{"content", "dir1", "dir2", "-e", "log"}, testDir)
	if !strings.Contains(validOutput, "A") || !strings.Contains(validOutput, "B") {
		testValue.Errorf("Valid command order failed unexpectedly.\nOutput: %s", validOutput)
	}
}
