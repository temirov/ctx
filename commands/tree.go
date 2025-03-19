package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/content/utils"
)

// TreeCommand prints a directory tree for the given rootDir,
// applying ignore patterns (using full Git‑ignore semantics).
func TreeCommand(rootDir string, ignorePatterns []string) error {
	fmt.Printf("Directory tree for: %s\n", rootDir)
	return printTree(rootDir, "", ignorePatterns, true)
}

// printTree recursively prints the directory tree with proper indentation.
// The isRoot flag indicates whether the current level is the root (affecting -e flag logic).
func printTree(currentPath, prefix string, ignorePatterns []string, isRoot bool) error {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	var filteredEntries []os.DirEntry
	for _, entry := range entries {
		fullPath := filepath.Join(currentPath, entry.Name())
		if utils.ShouldIgnore(entry, fullPath, ignorePatterns, isRoot) {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	for i, entry := range filteredEntries {
		isLast := i == len(filteredEntries)-1
		var newPrefix string
		if isLast {
			fmt.Printf("%s└── %s\n", prefix, entry.Name())
			newPrefix = prefix + "    "
		} else {
			fmt.Printf("%s├── %s\n", prefix, entry.Name())
			newPrefix = prefix + "│   "
		}
		if entry.IsDir() {
			if err := printTree(filepath.Join(currentPath, entry.Name()), newPrefix, ignorePatterns, false); err != nil {
				return err
			}
		}
	}
	return nil
}
