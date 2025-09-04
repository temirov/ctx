package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/temirov/ctx/internal/config"
)

const (
	ignoreFileName             = ".ignore"
	gitIgnoreFileName          = ".gitignore"
	showBinaryContentDirective = "show-binary-content:"
	gitDirectoryPattern        = ".git/"
)

// writeTestFile creates a file with specified content, failing the test on error.
func writeTestFile(testingHandle *testing.T, filePath string, content string) {
	testingHandle.Helper()
	if writeError := os.WriteFile(filePath, []byte(content), 0o644); writeError != nil {
		testingHandle.Fatalf("failed to write %s: %v", filePath, writeError)
	}
}

// TestLoadRecursiveIgnorePatterns verifies pattern and binary content aggregation across ignore file variations.
func TestLoadRecursiveIgnorePatterns(testingHandle *testing.T) {
	const (
		nestedIgnoreTestName      = "nested ignore files"
		nestedGitignoreTestName   = "nested gitignore files"
		binaryContentTestName     = "binary content patterns"
		nestedIgnoreRootPattern   = "root.txt"
		nestedIgnoreNestedPattern = "nested.txt"
		nestedIgnoreDirectory     = "nested"
		nestedGitRootPattern      = "root.md"
		nestedGitNestedPattern    = "nested.md"
		nestedGitDirectory        = "deep"
		binaryPatternName         = "data.bin"
		binaryNestedDirectory     = "nested"
	)

	testCases := []struct {
		name                  string
		setup                 func(rootDirectoryPath string)
		useGitignore          bool
		useIgnoreFile         bool
		expectedPatterns      []string
		expectedBinaryContent []string
	}{
		{
			name: nestedIgnoreTestName,
			setup: func(rootDirectoryPath string) {
				writeTestFile(testingHandle, filepath.Join(rootDirectoryPath, ignoreFileName), nestedIgnoreRootPattern+"\n")
				nestedDirectoryPath := filepath.Join(rootDirectoryPath, nestedIgnoreDirectory)
				if makeDirectoryError := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirectoryError != nil {
					testingHandle.Fatalf("failed to create nested directory: %v", makeDirectoryError)
				}
				writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, ignoreFileName), nestedIgnoreNestedPattern+"\n")
			},
			useIgnoreFile: true,
			expectedPatterns: []string{
				nestedIgnoreRootPattern,
				nestedIgnoreDirectory + "/" + nestedIgnoreNestedPattern,
				gitDirectoryPattern,
			},
			expectedBinaryContent: []string{},
		},
		{
			name: nestedGitignoreTestName,
			setup: func(rootDirectoryPath string) {
				writeTestFile(testingHandle, filepath.Join(rootDirectoryPath, gitIgnoreFileName), nestedGitRootPattern+"\n")
				nestedDirectoryPath := filepath.Join(rootDirectoryPath, nestedGitDirectory)
				if makeDirectoryError := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirectoryError != nil {
					testingHandle.Fatalf("failed to create nested directory: %v", makeDirectoryError)
				}
				writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, gitIgnoreFileName), nestedGitNestedPattern+"\n")
			},
			useGitignore: true,
			expectedPatterns: []string{
				nestedGitRootPattern,
				nestedGitDirectory + "/" + nestedGitNestedPattern,
				gitDirectoryPattern,
			},
			expectedBinaryContent: []string{},
		},
		{
			name: binaryContentTestName,
			setup: func(rootDirectoryPath string) {
				writeTestFile(testingHandle, filepath.Join(rootDirectoryPath, ignoreFileName), showBinaryContentDirective+binaryPatternName+"\n")
				nestedDirectoryPath := filepath.Join(rootDirectoryPath, binaryNestedDirectory)
				if makeDirectoryError := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirectoryError != nil {
					testingHandle.Fatalf("failed to create nested directory: %v", makeDirectoryError)
				}
				writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, ignoreFileName), showBinaryContentDirective+binaryPatternName+"\n")
			},
			useIgnoreFile: true,
			expectedPatterns: []string{
				gitDirectoryPattern,
			},
			expectedBinaryContent: []string{
				binaryPatternName,
				binaryNestedDirectory + "/" + binaryPatternName,
			},
		},
	}

	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(subTestHandle *testing.T) {
			rootDirectory := subTestHandle.TempDir()
			testCase.setup(rootDirectory)
			patternList, binaryPatternList, loadError := config.LoadRecursiveIgnorePatterns(rootDirectory, "", testCase.useGitignore, testCase.useIgnoreFile, false)
			if loadError != nil {
				subTestHandle.Fatalf("LoadRecursiveIgnorePatterns failed: %v", loadError)
			}
			sort.Strings(patternList)
			sort.Strings(testCase.expectedPatterns)
			if !reflect.DeepEqual(patternList, testCase.expectedPatterns) {
				subTestHandle.Fatalf("unexpected patterns: got %v want %v", patternList, testCase.expectedPatterns)
			}
			sort.Strings(binaryPatternList)
			sort.Strings(testCase.expectedBinaryContent)
			if !reflect.DeepEqual(binaryPatternList, testCase.expectedBinaryContent) {
				subTestHandle.Fatalf("unexpected binary content patterns: got %v want %v", binaryPatternList, testCase.expectedBinaryContent)
			}
		})
	}
}
