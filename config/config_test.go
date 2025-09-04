package config

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// writeTestFile creates a file with the specified content, failing the test on error.
func writeTestFile(testingHandle *testing.T, filePath string, content string) {
	testingHandle.Helper()
	if writeError := os.WriteFile(filePath, []byte(content), 0o644); writeError != nil {
		testingHandle.Fatalf("failed to write %s: %v", filePath, writeError)
	}
}

// TestLoadRecursiveIgnorePatternsNestedIgnore verifies that ignore patterns from nested .ignore files are aggregated with prefixed paths.
func TestLoadRecursiveIgnorePatternsNestedIgnore(testingHandle *testing.T) {
	const (
		rootPatternName   = "root.txt"
		nestedPatternName = "nested.txt"
		nestedDirName     = "nested"
	)

	rootDirectory := testingHandle.TempDir()
	writeTestFile(testingHandle, filepath.Join(rootDirectory, ignoreFileName), rootPatternName+"\n")

	nestedDirectoryPath := filepath.Join(rootDirectory, nestedDirName)
	if makeDirErr := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirErr != nil {
		testingHandle.Fatalf("failed to create nested directory: %v", makeDirErr)
	}
	writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, ignoreFileName), nestedPatternName+"\n")

	patternList, loadErr := LoadRecursiveIgnorePatterns(rootDirectory, "", false, true, false)
	if loadErr != nil {
		testingHandle.Fatalf("LoadRecursiveIgnorePatterns failed: %v", loadErr)
	}

	sort.Strings(patternList)
	expectedPatterns := []string{rootPatternName, nestedDirName + "/" + nestedPatternName, gitDirectoryPattern}
	sort.Strings(expectedPatterns)
	if !reflect.DeepEqual(patternList, expectedPatterns) {
		testingHandle.Fatalf("unexpected patterns: got %v want %v", patternList, expectedPatterns)
	}
}

// TestLoadRecursiveIgnorePatternsNestedGitIgnore verifies that ignore patterns from nested .gitignore files are aggregated with prefixed paths.
func TestLoadRecursiveIgnorePatternsNestedGitIgnore(testingHandle *testing.T) {
	const (
		rootGitPattern   = "root.md"
		nestedGitPattern = "nested.md"
		nestedGitDir     = "deep"
	)

	rootDirectory := testingHandle.TempDir()
	writeTestFile(testingHandle, filepath.Join(rootDirectory, gitIgnoreFileName), rootGitPattern+"\n")

	nestedDirectoryPath := filepath.Join(rootDirectory, nestedGitDir)
	if makeDirErr := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirErr != nil {
		testingHandle.Fatalf("failed to create nested directory: %v", makeDirErr)
	}
	writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, gitIgnoreFileName), nestedGitPattern+"\n")

	patternList, loadErr := LoadRecursiveIgnorePatterns(rootDirectory, "", true, false, false)
	if loadErr != nil {
		testingHandle.Fatalf("LoadRecursiveIgnorePatterns failed: %v", loadErr)
	}

	sort.Strings(patternList)
	expectedPatterns := []string{rootGitPattern, nestedGitDir + "/" + nestedGitPattern, gitDirectoryPattern}
	sort.Strings(expectedPatterns)
	if !reflect.DeepEqual(patternList, expectedPatterns) {
		testingHandle.Fatalf("unexpected patterns: got %v want %v", patternList, expectedPatterns)
	}
}
