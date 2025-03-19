package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/content/utils"
)

// ContentCommand traverses the directory tree starting from rootDirectory and prints the contents
// of files that are not ignored (using full Git‑ignore semantics and the -e flag logic).
func ContentCommand(rootDirectory string, ignorePatterns []string) error {
	absoluteRoot, errorValue := filepath.Abs(rootDirectory)
	if errorValue != nil {
		return errorValue
	}
	cleanRoot := filepath.Clean(absoluteRoot)

	return filepath.WalkDir(cleanRoot, func(currentPath string, directoryEntry os.DirEntry, errorValue error) error {
		if errorValue != nil {
			return errorValue
		}
		cleanPath := filepath.Clean(currentPath)
		relativePath, relativeError := filepath.Rel(cleanRoot, cleanPath)
		if relativeError != nil {
			relativePath = cleanPath
		}

		// Skip the root directory itself.
		if relativePath == "." {
			return nil
		}

		// Determine if the entry is a direct child of the root by checking if its parent is ".".
		if directoryEntry.IsDir() && filepath.Dir(relativePath) == "." {
			for _, pattern := range ignorePatterns {
				if strings.HasPrefix(pattern, "EXCL:") {
					exclusionValue := strings.TrimPrefix(pattern, "EXCL:")
					if relativePath == exclusionValue {
						// Skip this directory and its entire subtree.
						return filepath.SkipDir
					}
				} else if strings.HasSuffix(pattern, "/") {
					patternDirectory := strings.TrimSuffix(pattern, "/")
					if relativePath == patternDirectory {
						return filepath.SkipDir
					}
				}
			}
		}

		// For files (or directories not caught above), apply the standard ignore check.
		if utils.ShouldIgnoreByPath(relativePath, directoryEntry, ignorePatterns) {
			if directoryEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// If it's a file, print its path and contents.
		if !directoryEntry.IsDir() {
			fmt.Printf("File: %s\n", currentPath)
			fileData, fileError := os.ReadFile(currentPath)
			if fileError != nil {
				return fileError
			}
			fmt.Println(string(fileData))
			fmt.Printf("End of file: %s\n", currentPath)
			fmt.Println("----------------------------------------")
		}
		return nil
	})
}
