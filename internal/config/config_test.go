package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/temirov/ctx/internal/config"
	"github.com/temirov/ctx/internal/utils"
)

const (
	gitDirectoryPattern        = utils.GitDirectoryName + "/"
	showBinaryContentDirective = "show-binary-content:"
)

// writeTestFile creates a file with the specified content, failing the test on error.
func writeTestFile(testingHandle *testing.T, filePath string, content string) {
	testingHandle.Helper()
	if writeError := os.WriteFile(filePath, []byte(content), 0o644); writeError != nil {
		testingHandle.Fatalf("failed to write %s: %v", filePath, writeError)
	}
}

// TestLoadRecursiveIgnorePatterns verifies that ignore and binary content patterns are correctly aggregated across nested directories.
func TestLoadRecursiveIgnorePatterns(testingHandle *testing.T) {
	const (
		nestedIgnoreSubtestName    = "nested ignore files"
		nestedGitignoreSubtestName = "nested gitignore files"
		binaryContentSubtestName   = "binary content patterns"

		rootIgnorePatternName   = "root.txt"
		nestedIgnorePatternName = "nested.txt"
		nestedIgnoreDirName     = "nested"

		rootGitPatternName   = "root.md"
		nestedGitPatternName = "nested.md"
		nestedGitDirName     = "deep"

		binaryPatternName   = "data.bin"
		nestedBinaryDirName = "binary"
	)

	testCases := []struct {
		name                   string
		useGitignore           bool
		useIgnoreFile          bool
		createTestFiles        func(*testing.T, string)
		expectedPatterns       []string
		expectedBinaryPatterns []string
	}{
		{
			name:          nestedIgnoreSubtestName,
			useIgnoreFile: true,
			createTestFiles: func(testingHandle *testing.T, rootDirectory string) {
				writeTestFile(testingHandle, filepath.Join(rootDirectory, utils.IgnoreFileName), rootIgnorePatternName+"\n")
				nestedDirectoryPath := filepath.Join(rootDirectory, nestedIgnoreDirName)
				if makeDirError := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirError != nil {
					testingHandle.Fatalf("failed to create nested directory: %v", makeDirError)
				}
				writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, utils.IgnoreFileName), nestedIgnorePatternName+"\n")
			},
			expectedPatterns:       []string{rootIgnorePatternName, nestedIgnoreDirName + "/" + nestedIgnorePatternName, gitDirectoryPattern},
			expectedBinaryPatterns: []string{},
		},
		{
			name:         nestedGitignoreSubtestName,
			useGitignore: true,
			createTestFiles: func(testingHandle *testing.T, rootDirectory string) {
				writeTestFile(testingHandle, filepath.Join(rootDirectory, utils.GitIgnoreFileName), rootGitPatternName+"\n")
				nestedDirectoryPath := filepath.Join(rootDirectory, nestedGitDirName)
				if makeDirError := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirError != nil {
					testingHandle.Fatalf("failed to create nested directory: %v", makeDirError)
				}
				writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, utils.GitIgnoreFileName), nestedGitPatternName+"\n")
			},
			expectedPatterns:       []string{rootGitPatternName, nestedGitDirName + "/" + nestedGitPatternName, gitDirectoryPattern},
			expectedBinaryPatterns: []string{},
		},
		{
			name:          binaryContentSubtestName,
			useIgnoreFile: true,
			createTestFiles: func(testingHandle *testing.T, rootDirectory string) {
				writeTestFile(testingHandle, filepath.Join(rootDirectory, utils.IgnoreFileName), showBinaryContentDirective+binaryPatternName+"\n")
				nestedDirectoryPath := filepath.Join(rootDirectory, nestedBinaryDirName)
				if makeDirError := os.MkdirAll(nestedDirectoryPath, 0o755); makeDirError != nil {
					testingHandle.Fatalf("failed to create nested directory: %v", makeDirError)
				}
				writeTestFile(testingHandle, filepath.Join(nestedDirectoryPath, utils.IgnoreFileName), showBinaryContentDirective+binaryPatternName+"\n")
			},
			expectedPatterns:       []string{gitDirectoryPattern},
			expectedBinaryPatterns: []string{binaryPatternName, nestedBinaryDirName + "/" + binaryPatternName},
		},
	}

	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			rootDirectory := testingHandle.TempDir()
			testCase.createTestFiles(testingHandle, rootDirectory)

			patternList, binaryPatternList, loadError := config.LoadRecursiveIgnorePatterns(rootDirectory, "", testCase.useGitignore, testCase.useIgnoreFile, false)
			if loadError != nil {
				testingHandle.Fatalf("LoadRecursiveIgnorePatterns failed: %v", loadError)
			}

			sort.Strings(patternList)
			sort.Strings(testCase.expectedPatterns)
			if !reflect.DeepEqual(patternList, testCase.expectedPatterns) {
				testingHandle.Fatalf("unexpected patterns: got %v want %v", patternList, testCase.expectedPatterns)
			}

			sort.Strings(binaryPatternList)
			sort.Strings(testCase.expectedBinaryPatterns)
			if !reflect.DeepEqual(binaryPatternList, testCase.expectedBinaryPatterns) {
				testingHandle.Fatalf("unexpected binary content patterns: got %v want %v", binaryPatternList, testCase.expectedBinaryPatterns)
			}
		})
	}
}
