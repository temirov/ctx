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

	output := runCommand(testValue, binary, []string{"tree", filepath.Base(testDir)}, filepath.Dir(testDir))

	expectedTreeHeader := fmt.Sprintf("--- Directory Tree: %s ---", filepath.Join(filepath.Dir(testDir), filepath.Base(testDir)))
	if !strings.Contains(output, expectedTreeHeader) {
		testValue.Errorf("Tree output missing expected header '%s'.\nOutput: %s", expectedTreeHeader, output)
	}
	if !strings.Contains(output, "file1.txt") {
		testValue.Errorf("Tree output missing 'file1.txt'.\nOutput: %s", output)
	}
	if !strings.Contains(output, "subdir") {
		testValue.Errorf("Tree output missing 'subdir'.\nOutput: %s", output)
	}
	if !strings.Contains(output, "file2.txt") {
		testValue.Errorf("Tree output missing 'file2.txt'.\nOutput: %s", output)
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
	if !strings.Contains(output, "End of file:") {
		testValue.Errorf("Content output missing 'End of file:' marker.\nOutput: %s", output)
	}
}

func TestTreeCommandIntegration_WithIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		".ignore":           "log/\n*.tmp",
		"log/":              "",
		"log/ignored.txt":   "ignore me",
		"data/":             "",
		"data/included.txt": "include me",
		"temp.tmp":          "temporary",
	})

	output := runCommand(testValue, binary, []string{"tree", "."}, testDir)

	expectedTreeHeader := fmt.Sprintf("--- Directory Tree: %s ---", testDir)
	if !strings.Contains(output, expectedTreeHeader) {
		testValue.Errorf("Tree output missing expected header '%s'.\nOutput: %s", expectedTreeHeader, output)
	}
	if strings.Contains(output, "log") {
		testValue.Errorf("Tree output should not include 'log' directory.\nOutput: %s", output)
	}
	if strings.Contains(output, "ignored.txt") {
		testValue.Errorf("Tree output should not include 'ignored.txt'.\nOutput: %s", output)
	}
	if strings.Contains(output, "temp.tmp") {
		testValue.Errorf("Tree output should not include 'temp.tmp'.\nOutput: %s", output)
	}
	if !strings.Contains(output, "data") {
		testValue.Errorf("Tree output should include 'data' directory.\nOutput: %s", output)
	}
	if !strings.Contains(output, "included.txt") {
		testValue.Errorf("Tree output should include 'included.txt'.\nOutput: %s", output)
	}
}

func TestContentCommandIntegration_WithIgnore(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		".ignore":           "log/\n*.tmp",
		"log/":              "",
		"log/ignored.txt":   "ignore me content",
		"data/":             "",
		"data/included.txt": "include me content",
		"temp.tmp":          "temporary content",
	})

	output := runCommand(testValue, binary, []string{"content", "."}, testDir)

	if strings.Contains(output, "ignore me content") {
		testValue.Errorf("Content output should not include content from log/ignored.txt.\nOutput: %s", output)
	}
	if strings.Contains(output, "temporary content") {
		testValue.Errorf("Content output should not include content from temp.tmp.\nOutput: %s", output)
	}
	if !strings.Contains(output, "include me content") {
		testValue.Errorf("Content output should include content from data/included.txt.\nOutput: %s", output)
	}
	expectedFileHeader := filepath.Join(testDir, "data", "included.txt")
	if !strings.Contains(output, expectedFileHeader) {
		testValue.Errorf("Content output missing expected file header for included.txt.\nOutput: %s", output)
	}
}

func TestExclusionFlagIntegration(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"pkg/":               "",
		"pkg/log/":           "",
		"pkg/log/ignore.txt": "should be ignored by -e",
		"pkg/include.txt":    "should be included",
		// Removed .ignore: "" to avoid creating a directory
	})

	output := runCommand(testValue, binary, []string{"content", "pkg", "-e", "log"}, testDir)

	if strings.Contains(output, "should be ignored by -e") {
		testValue.Errorf("Content output should not include content from pkg/log when -e=log is set.\nOutput: %s", output)
	}
	if !strings.Contains(output, "should be included") {
		testValue.Errorf("Content output should include content from pkg/include.txt.\nOutput: %s", output)
	}
	expectedFileHeader := filepath.Join(testDir, "pkg", "include.txt")
	if !strings.Contains(output, expectedFileHeader) {
		testValue.Errorf("Content output missing expected file header for include.txt.\nOutput: %s", output)
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
		// Removed .ignore: "" which caused the error by creating a directory
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
	expectedNestedHeader := filepath.Join(testDir, "pkg", "log", "nested_log.txt")
	if !strings.Contains(output, expectedNestedHeader) {
		testValue.Errorf("Content output missing expected file header for nested_log.txt.\nOutput: %s", output)
	}
}

func TestExclusionFlagTrailingSlash(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"exclude_me/":         "",
		"exclude_me/file.txt": "should be excluded",
		"include_me.txt":      "should be included",
	})

	outputNoSlash := runCommand(testValue, binary, []string{"content", ".", "-e", "exclude_me"}, testDir)
	if strings.Contains(outputNoSlash, "should be excluded") {
		testValue.Errorf("Content should be excluded when -e=exclude_me.\nOutput: %s", outputNoSlash)
	}
	if !strings.Contains(outputNoSlash, "should be included") {
		testValue.Errorf("Included content missing when -e=exclude_me.\nOutput: %s", outputNoSlash)
	}

	outputWithSlash := runCommand(testValue, binary, []string{"content", ".", "-e", "exclude_me/"}, testDir)
	if strings.Contains(outputWithSlash, "should be excluded") {
		testValue.Errorf("Content should be excluded when -e=exclude_me/.\nOutput: %s", outputWithSlash)
	}
	if !strings.Contains(outputWithSlash, "should be included") {
		testValue.Errorf("Included content missing when -e=exclude_me/.\nOutput: %s", outputWithSlash)
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

func TestMultiDir_IgnoreSpecificity(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"proj1/":           "",
		"proj1/.ignore":    "*.log",
		"proj1/app.log":    "proj1 log",
		"proj1/data.txt":   "proj1 data",
		"proj2/":           "",
		"proj2/.ignore":    "*.txt",
		"proj2/server.log": "proj2 log",
		"proj2/config.txt": "proj2 data",
	})

	output := runCommand(testValue, binary, []string{"content", "proj1", "proj2"}, testDir)

	if strings.Contains(output, "proj1 log") {
		testValue.Errorf("proj1 log should be ignored by proj1/.ignore.\nOutput: %s", output)
	}
	if !strings.Contains(output, "proj1 data") {
		testValue.Errorf("proj1 data should be included.\nOutput: %s", output)
	}

	if !strings.Contains(output, "proj2 log") {
		testValue.Errorf("proj2 log should be included (not ignored by proj2/.ignore).\nOutput: %s", output)
	}
	if strings.Contains(output, "proj2 data") {
		testValue.Errorf("proj2 data should be ignored by proj2/.ignore.\nOutput: %s", output)
	}
}

func TestMultiDir_GitignoreSpecificity(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"proj1/":           "",
		"proj1/.gitignore": "*.log",
		"proj1/app.log":    "proj1 log",
		"proj1/data.txt":   "proj1 data",
		"proj2/":           "",
		"proj2/.gitignore": "*.txt",
		"proj2/server.log": "proj2 log",
		"proj2/config.txt": "proj2 data",
	})

	output := runCommand(testValue, binary, []string{"content", "proj1", "proj2"}, testDir)

	if strings.Contains(output, "proj1 log") {
		testValue.Errorf("proj1 log should be ignored by proj1/.gitignore.\nOutput: %s", output)
	}
	if !strings.Contains(output, "proj1 data") {
		testValue.Errorf("proj1 data should be included.\nOutput: %s", output)
	}

	if !strings.Contains(output, "proj2 log") {
		testValue.Errorf("proj2 log should be included (not ignored by proj2/.gitignore).\nOutput: %s", output)
	}
	if strings.Contains(output, "proj2 data") {
		testValue.Errorf("proj2 data should be ignored by proj2/.gitignore.\nOutput: %s", output)
	}
}

func TestMultiDir_NoIgnoreFlag(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"dir1/":            "",
		"dir1/.ignore":     "ignored.txt",
		"dir1/ignored.txt": "dir1 ignored",
		"dir1/kept.txt":    "dir1 kept",
		"dir2/":            "",
		"dir2/.ignore":     "*.log",
		"dir2/app.log":     "dir2 ignored",
		"dir2/data.txt":    "dir2 kept",
	})

	output := runCommand(testValue, binary, []string{"content", "dir1", "dir2", "--no-ignore"}, testDir)

	if !strings.Contains(output, "dir1 ignored") {
		testValue.Errorf("dir1 ignored content missing, but --no-ignore was used.\nOutput: %s", output)
	}
	if !strings.Contains(output, "dir1 kept") {
		testValue.Errorf("dir1 kept content missing.\nOutput: %s", output)
	}
	if !strings.Contains(output, "dir2 ignored") {
		testValue.Errorf("dir2 ignored content missing, but --no-ignore was used.\nOutput: %s", output)
	}
	if !strings.Contains(output, "dir2 kept") {
		testValue.Errorf("dir2 kept content missing.\nOutput: %s", output)
	}
}

func TestMultiDir_ExclusionFlag(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"app1/":             "",
		"app1/logs/":        "",
		"app1/logs/err.log": "app1 error",
		"app1/main.go":      "app1 code",
		"app2/":             "",
		"app2/logs/":        "",
		"app2/logs/out.log": "app2 output",
		"app2/config.json":  "app2 config",
		"app3/":             "",
		"app3/README.md":    "app3 readme",
	})

	output := runCommand(testValue, binary, []string{"content", "app1", "app2", "app3", "-e", "logs"}, testDir)

	if strings.Contains(output, "app1 error") {
		testValue.Errorf("Content from app1/logs should be excluded by -e=logs.\nOutput: %s", output)
	}
	if strings.Contains(output, "app2 output") {
		testValue.Errorf("Content from app2/logs should be excluded by -e=logs.\nOutput: %s", output)
	}

	if !strings.Contains(output, "app1 code") {
		testValue.Errorf("Content from app1/main.go should be included.\nOutput: %s", output)
	}
	if !strings.Contains(output, "app2 config") {
		testValue.Errorf("Content from app2/config.json should be included.\nOutput: %s", output)
	}
	if !strings.Contains(output, "app3 readme") {
		testValue.Errorf("Content from app3/README.md should be included.\nOutput: %s", output)
	}
}

func TestMultiDir_DuplicateDirs(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"myproj/":         "",
		"myproj/file.txt": "Unique Content",
	})

	output := runCommand(testValue, binary, []string{"content", "myproj", "./myproj"}, testDir)

	count := strings.Count(output, "Unique Content")
	if count != 1 {
		testValue.Errorf("Expected content 'Unique Content' to appear exactly once, but found %d times.\nOutput: %s", count, output)
	}

	treeOutput := runCommand(testValue, binary, []string{"tree", "myproj", "./myproj"}, testDir)
	absMyProj, _ := filepath.Abs(filepath.Join(testDir, "myproj"))
	expectedHeader := fmt.Sprintf("--- Directory Tree: %s ---", absMyProj)
	headerCount := strings.Count(treeOutput, expectedHeader)
	if headerCount != 1 {
		testValue.Errorf("Expected tree header '%s' to appear exactly once, but found %d times.\nOutput: %s", expectedHeader, headerCount, treeOutput)
	}
}

func TestMultiDir_OneInvalidDir(testValue *testing.T) {
	binary := buildBinary(testValue)
	testDir := setupTestDirectory(testValue, map[string]string{
		"valid_dir/":      "",
		"valid_dir/a.txt": "A",
	})
	invalidPath := "non_existent_dir"

	output := runCommandExpectError(testValue, binary, []string{"content", "valid_dir", invalidPath}, testDir)

	if !strings.Contains(output, invalidPath) {
		testValue.Errorf("Expected error output to mention the invalid path '%s'.\nOutput: %s", invalidPath, output)
	}
	if !strings.Contains(output, "is not a valid directory") {
		testValue.Errorf("Expected error output to contain 'is not a valid directory'.\nOutput: %s", output)
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
	localHeader := filepath.Join(testDir, "local_file.txt")
	if !strings.Contains(output, localHeader) {
		testValue.Errorf("Output missing file header for local_file.txt.\nOutput: %s", output)
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
