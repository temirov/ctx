package config

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/temirov/ctx/internal/utils"
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
	writeTestFile(testingHandle, filepath.Join(rootDirectory, utils.IgnoreFileName), rootPatternName+"\n")

	nestedDirectoryPath := filepath.Join(rootDirectory, nestedDirName)
	if makeDirErr := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirErr != nil {
		testingHandle.Fatalf("failed to create nested directory: %v", makeDirErr)
	}
	writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, utils.IgnoreFileName), nestedPatternName+"\n")

	patternList, binaryPatternList, loadError := LoadRecursiveIgnorePatterns(rootDirectory, "", false, true, false)
	if loadError != nil {
		testingHandle.Fatalf("LoadRecursiveIgnorePatterns failed: %v", loadError)
	}

	sort.Strings(patternList)
	expectedPatterns := []string{rootPatternName, nestedDirName + "/" + nestedPatternName, gitDirectoryPattern}
	sort.Strings(expectedPatterns)
	if !reflect.DeepEqual(patternList, expectedPatterns) {
		testingHandle.Fatalf("unexpected patterns: got %v want %v", patternList, expectedPatterns)
	}
	if len(binaryPatternList) != 0 {
		testingHandle.Fatalf("expected no binary content patterns, got %v", binaryPatternList)
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
	writeTestFile(testingHandle, filepath.Join(rootDirectory, utils.GitIgnoreFileName), rootGitPattern+"\n")

	nestedDirectoryPath := filepath.Join(rootDirectory, nestedGitDir)
	if makeDirErr := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirErr != nil {
		testingHandle.Fatalf("failed to create nested directory: %v", makeDirErr)
	}
	writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, utils.GitIgnoreFileName), nestedGitPattern+"\n")

	patternList, binaryPatternList, loadError := LoadRecursiveIgnorePatterns(rootDirectory, "", true, false, false)
	if loadError != nil {
		testingHandle.Fatalf("LoadRecursiveIgnorePatterns failed: %v", loadError)
	}

	sort.Strings(patternList)
	expectedPatterns := []string{rootGitPattern, nestedGitDir + "/" + nestedGitPattern, gitDirectoryPattern}
	sort.Strings(expectedPatterns)
	if !reflect.DeepEqual(patternList, expectedPatterns) {
		testingHandle.Fatalf("unexpected patterns: got %v want %v", patternList, expectedPatterns)
	}
	if len(binaryPatternList) != 0 {
		testingHandle.Fatalf("expected no binary content patterns, got %v", binaryPatternList)
	}
}

// TestLoadRecursiveIgnorePatternsBinaryContent verifies aggregation of binary content patterns.
func TestLoadRecursiveIgnorePatternsBinaryContent(testingHandle *testing.T) {
	const (
		binaryPatternName   = "data.bin"
		nestedDirectoryName = "nested"
	)

	rootDirectory := testingHandle.TempDir()
	writeTestFile(testingHandle, filepath.Join(rootDirectory, utils.IgnoreFileName), showBinaryContentDirective+binaryPatternName+"\n")

	nestedDirectoryPath := filepath.Join(rootDirectory, nestedDirectoryName)
	if makeDirError := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirError != nil {
		testingHandle.Fatalf("failed to create nested directory: %v", makeDirError)
	}
	writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, utils.IgnoreFileName), showBinaryContentDirective+binaryPatternName+"\n")

	patternList, binaryPatternList, loadError := LoadRecursiveIgnorePatterns(rootDirectory, "", false, true, false)
	if loadError != nil {
		testingHandle.Fatalf("LoadRecursiveIgnorePatterns failed: %v", loadError)
	}

	if len(patternList) != 1 || patternList[0] != gitDirectoryPattern {
		testingHandle.Fatalf("unexpected ignore patterns: %v", patternList)
	}

	sort.Strings(binaryPatternList)
	expectedBinaryPatterns := []string{binaryPatternName, nestedDirectoryName + "/" + binaryPatternName}
	sort.Strings(expectedBinaryPatterns)
	if !reflect.DeepEqual(binaryPatternList, expectedBinaryPatterns) {
		testingHandle.Fatalf("unexpected binary content patterns: got %v want %v", binaryPatternList, expectedBinaryPatterns)
	}
}
