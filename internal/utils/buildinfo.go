// Package utils provides helper functions, including version retrieval.
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
)

const (
	unknownVersion = "unknown"
)

// GetApplicationVersion attempts to determine the application version using various methods.
// It checks Go build info first, then falls back to git describe commands if available.
func GetApplicationVersion() string {
	buildInfo, buildInfoAvailable := debug.ReadBuildInfo()
	if buildInfoAvailable && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return buildInfo.Main.Version
	}

	gitDirectoryPath, gitDirectoryError := findGitDirectory(".")
	if gitDirectoryError == nil && gitDirectoryPath != "" {
		// #nosec G204
		gitExactCommand := exec.Command("git", "describe", "--tags", "--exact-match")
		gitExactCommand.Dir = gitDirectoryPath
		gitExactOutput, errorGitExact := gitExactCommand.Output()
		if errorGitExact == nil && len(gitExactOutput) > 0 {
			return strings.TrimSpace(string(gitExactOutput))
		}

		// #nosec G204
		gitLongCommand := exec.Command("git", "describe", "--tags", "--long", "--dirty")
		gitLongCommand.Dir = gitDirectoryPath
		gitLongOutput, errorGitLong := gitLongCommand.Output()
		if errorGitLong == nil && len(gitLongOutput) > 0 {
			return strings.TrimSpace(string(gitLongOutput))
		}
	}

	return unknownVersion
}

// findGitDirectory searches upward from the provided starting directory
// until it locates a directory containing the .git folder and returns
// the path to that directory.
func findGitDirectory(startDirectory string) (string, error) {
	absoluteStartDirectory, errorAbsolute := filepath.Abs(startDirectory)
	if errorAbsolute != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", startDirectory, errorAbsolute)
	}

	currentDirectory := absoluteStartDirectory
	for {
		gitPath := filepath.Join(currentDirectory, GitDirectoryName)
		fileInformation, errorStat := os.Stat(gitPath)
		if errorStat == nil && fileInformation.IsDir() {
			return currentDirectory, nil
		}

		parentDirectory := filepath.Dir(currentDirectory)
		if parentDirectory == currentDirectory {
			break
		}
		currentDirectory = parentDirectory
	}

	return "", fmt.Errorf(".git directory not found in or above %s", absoluteStartDirectory)
}
