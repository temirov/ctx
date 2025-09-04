package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/temirov/ctx/config"
)

const (
	patternAlpha        = "alpha"
	patternBeta         = "beta"
	commentLine         = "# comment"
	exclusionFolderName = "excluded"
	ignoreFileName      = ".ignore"
	gitIgnoreFileName   = ".gitignore"
	characterA          = "a"
)

// createFile creates a file with the specified content.
func createFile(testingHandle *testing.T, filePath string, content string) {
	writeError := os.WriteFile(filePath, []byte(content), 0o644)
	if writeError != nil {
		testingHandle.Fatalf("failed to write file %s: %v", filePath, writeError)
	}
}

// createUnreadablePath creates a directory to trigger read errors.
func createUnreadablePath(testingHandle *testing.T, path string) {
	makeError := os.Mkdir(path, 0o755)
	if makeError != nil {
		testingHandle.Fatalf("failed to create unreadable path %s: %v", path, makeError)
	}
}

// TestLoadIgnoreFilePatterns verifies ignore file parsing and error handling.
func TestLoadIgnoreFilePatterns(testingHandle *testing.T) {
	testCases := []struct {
		name             string
		setup            func(*testing.T) string
		expectedPatterns []string
		expectError      bool
	}{
		{
			name: "MissingFile",
			setup: func(t *testing.T) string {
				temporaryDirectory := t.TempDir()
				return filepath.Join(temporaryDirectory, ignoreFileName)
			},
			expectedPatterns: nil,
			expectError:      false,
		},
		{
			name: "ValidFile",
			setup: func(t *testing.T) string {
				temporaryDirectory := t.TempDir()
				filePath := filepath.Join(temporaryDirectory, ignoreFileName)
				fileContent := strings.Join([]string{commentLine, patternAlpha, "", patternBeta}, "\n")
				createFile(t, filePath, fileContent)
				return filePath
			},
			expectedPatterns: []string{patternAlpha, patternBeta},
			expectError:      false,
		},
		{
			name: "UnreadableFile",
			setup: func(t *testing.T) string {
				temporaryDirectory := t.TempDir()
				filePath := filepath.Join(temporaryDirectory, ignoreFileName)
				createUnreadablePath(t, filePath)
				return filePath
			},
			expectedPatterns: nil,
			expectError:      true,
		},
		{
			name: "MalformedFile",
			setup: func(t *testing.T) string {
				temporaryDirectory := t.TempDir()
				filePath := filepath.Join(temporaryDirectory, ignoreFileName)
				longLine := strings.Repeat(characterA, 70000)
				createFile(t, filePath, longLine)
				return filePath
			},
			expectedPatterns: nil,
			expectError:      true,
		},
	}

	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			filePath := testCase.setup(testingHandle)
			patterns, loadError := config.LoadIgnoreFilePatterns(filePath)
			if testCase.expectError {
				if loadError == nil {
					testingHandle.Fatalf("expected error for %s", testCase.name)
				}
				return
			}
			if loadError != nil {
				testingHandle.Fatalf("unexpected error for %s: %v", testCase.name, loadError)
			}
			if !reflect.DeepEqual(patterns, testCase.expectedPatterns) {
				testingHandle.Fatalf("expected %v, got %v", testCase.expectedPatterns, patterns)
			}
		})
	}
}

// TestLoadCombinedIgnorePatterns verifies combined loading, deduplication, exclusion addition, and error paths.
func TestLoadCombinedIgnorePatterns(testingHandle *testing.T) {
	exclusionPattern := "EXCL:" + exclusionFolderName
	testCases := []struct {
		name             string
		setup            func(*testing.T) string
		exclusion        string
		useGitignore     bool
		useIgnoreFile    bool
		expectedPatterns []string
		expectError      bool
	}{
		{
			name: "MissingFiles",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			exclusion:        "",
			useGitignore:     true,
			useIgnoreFile:    true,
			expectedPatterns: []string{},
			expectError:      false,
		},
		{
			name: "CombinedWithDedupAndExclusion",
			setup: func(t *testing.T) string {
				temporaryDirectory := t.TempDir()
				ignorePath := filepath.Join(temporaryDirectory, ignoreFileName)
				gitignorePath := filepath.Join(temporaryDirectory, gitIgnoreFileName)
				createFile(t, ignorePath, strings.Join([]string{patternAlpha, patternBeta}, "\n"))
				createFile(t, gitignorePath, strings.Join([]string{patternBeta, patternAlpha}, "\n"))
				return temporaryDirectory
			},
			exclusion:        exclusionFolderName,
			useGitignore:     true,
			useIgnoreFile:    true,
			expectedPatterns: []string{patternAlpha, patternBeta, exclusionPattern},
			expectError:      false,
		},
		{
			name: "IgnoreFileError",
			setup: func(t *testing.T) string {
				temporaryDirectory := t.TempDir()
				ignorePath := filepath.Join(temporaryDirectory, ignoreFileName)
				createUnreadablePath(t, ignorePath)
				return temporaryDirectory
			},
			exclusion:        "",
			useGitignore:     false,
			useIgnoreFile:    true,
			expectedPatterns: nil,
			expectError:      true,
		},
		{
			name: "GitIgnoreError",
			setup: func(t *testing.T) string {
				temporaryDirectory := t.TempDir()
				gitignorePath := filepath.Join(temporaryDirectory, gitIgnoreFileName)
				createUnreadablePath(t, gitignorePath)
				return temporaryDirectory
			},
			exclusion:        "",
			useGitignore:     true,
			useIgnoreFile:    false,
			expectedPatterns: nil,
			expectError:      true,
		},
	}

	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			directoryPath := testCase.setup(testingHandle)
			patterns, loadError := config.LoadCombinedIgnorePatterns(directoryPath, testCase.exclusion, testCase.useGitignore, testCase.useIgnoreFile)
			if testCase.expectError {
				if loadError == nil {
					testingHandle.Fatalf("expected error for %s", testCase.name)
				}
				return
			}
			if loadError != nil {
				testingHandle.Fatalf("unexpected error for %s: %v", testCase.name, loadError)
			}
			if !reflect.DeepEqual(patterns, testCase.expectedPatterns) {
				testingHandle.Fatalf("expected %v, got %v", testCase.expectedPatterns, patterns)
			}
		})
	}
}
