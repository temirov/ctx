// Package commands contains the CLI commands for the 'content' tool,
// including logic for printing a directory tree with .ignore exclusions.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	//nolint:depguard
	"github.com/temirov/content/utils"
)

// TreeCommand prints a directory tree for the given rootDir, applying ignore patterns.
func TreeCommand(rootDir string, ignorePatterns []string) error {
	fmt.Printf("Directory tree for: %s\n", rootDir)
	return printTree(rootDir, "", ignorePatterns, true)
}

// printTree recursively prints the directory tree with proper indentation.
// The isRoot flag indicates whether the current level is the root (affecting -e flag logic).
func printTree(currentPath, prefix string, ignorePatterns []string, isRoot bool) error {
	directoryEntries, readError := os.ReadDir(currentPath)
	if readError != nil {
		return readError
	}

	// Preallocate with capacity=length for performance & linter
	filteredEntries := make([]os.DirEntry, 0, len(directoryEntries))

	for _, directoryEntry := range directoryEntries {
		if utils.ShouldIgnore(directoryEntry, ignorePatterns, isRoot) {
			continue
		}
		filteredEntries = append(filteredEntries, directoryEntry)
	}

	for entryIndex, directoryEntry := range filteredEntries {
		isLast := entryIndex == len(filteredEntries)-1
		if isLast {
			fmt.Printf("%s└── %s\n", prefix, directoryEntry.Name())
		} else {
			fmt.Printf("%s├── %s\n", prefix, directoryEntry.Name())
		}

		var newPrefix string
		if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}

		if directoryEntry.IsDir() {
			childPath := filepath.Join(currentPath, directoryEntry.Name())
			callError := printTree(childPath, newPrefix, ignorePatterns, false)
			if callError != nil {
				return callError
			}
		}
	}
	return nil
}
