// Package commands contains the CLI commands for the 'content' tool,
// including logic for printing file contents with .contentignore exclusions.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	//nolint:depguard
	"github.com/temirov/content/utils"
)

// ContentCommand traverses the directory tree starting from rootDirectory
// and prints the contents of files that are not excluded by .contentignore or -e.
func ContentCommand(rootDirectory string, ignorePatterns []string) error {
	absoluteRootDirectory, absoluteError := filepath.Abs(rootDirectory)
	if absoluteError != nil {
		return absoluteError
	}
	cleanRootDirectory := filepath.Clean(absoluteRootDirectory)

	walkError := filepath.WalkDir(cleanRootDirectory, func(path string, entry os.DirEntry, err error) error {
		return handleContentWalkEntry(path, entry, err, cleanRootDirectory, ignorePatterns)
	})

	return walkError
}

// handleContentWalkEntry is broken out into smaller sub‑checks to keep complexity <= 10.
func handleContentWalkEntry(
	currentPath string,
	directoryEntry os.DirEntry,
	entryError error,
	cleanRoot string,
	ignorePatterns []string,
) error {
	if entryError != nil {
		return entryError
	}

	relativePath := relativeOrSelf(currentPath, cleanRoot)
	if relativePath == "." {
		return nil // Skip the root directory itself
	}

	// #1: If it's a top-level directory that matches an EXCL: pattern, skip
	skipDir, skipDirErr := handleExclusionFolder(directoryEntry, relativePath, ignorePatterns)
	if skipDirErr != nil || skipDir {
		return skipDirErr
	}

	// #2: If it matches typical .contentignore rules, skip
	if utils.ShouldIgnoreByPath(relativePath, ignorePatterns) {
		if directoryEntry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}

	// #3: If it's a file, print it
	if !directoryEntry.IsDir() {
		return printFileContents(currentPath)
	}
	return nil
}

// relativeOrSelf returns relative path or the path itself on error.
func relativeOrSelf(fullPath, root string) string {
	cleanPath := filepath.Clean(fullPath)
	relativePath, relErr := filepath.Rel(root, cleanPath)
	if relErr != nil {
		return cleanPath
	}
	return relativePath
}

// handleExclusionFolder checks if the directory is excluded by the "EXCL:" or "log/" logic for direct children.
func handleExclusionFolder(
	directoryEntry os.DirEntry,
	relativePath string,
	ignorePatterns []string,
) (bool, error) {
	// Only apply if it is a directory and a top-level item.
	if !directoryEntry.IsDir() || filepath.Dir(relativePath) != "." {
		return false, nil
	}
	for _, patternValue := range ignorePatterns {
		if strings.HasPrefix(patternValue, "EXCL:") {
			exclusionName := strings.TrimPrefix(patternValue, "EXCL:")
			if relativePath == exclusionName {
				return true, filepath.SkipDir
			}
		} else if strings.HasSuffix(patternValue, "/") {
			patternDirectory := strings.TrimSuffix(patternValue, "/")
			if relativePath == patternDirectory {
				return true, filepath.SkipDir
			}
		}
	}
	return false, nil
}

// printFileContents prints the file's path and data.
func printFileContents(path string) error {
	fmt.Printf("File: %s\n", path)
	fileData, readErr := os.ReadFile(path)
	if readErr != nil {
		return readErr
	}
	fmt.Println(string(fileData))
	fmt.Printf("End of file: %s\n", path)
	fmt.Println("----------------------------------------")
	return nil
}
