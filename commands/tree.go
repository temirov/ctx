// Package commands contains the CLI commands for the 'content' tool,
// including logic for printing a directory tree with .ignore exclusions.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	//nolint:depguard
	"github.com/temirov/content/types"
	//nolint:depguard
	"github.com/temirov/content/utils"
)

// GetTreeData generates the tree structure data for a given directory.
// It returns a slice containing a single root node representing the directory.
// Warnings for skipped subdirectories are printed to stderr.
func GetTreeData(rootDir string, ignorePatterns []string) ([]*types.TreeOutputNode, error) {
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path for %s: %w", rootDir, err)
	}

	// Create the root node representing the input directory itself
	rootNode := &types.TreeOutputNode{
		Path: absRootDir,
		Name: filepath.Base(absRootDir),
		Type: "directory",
	}

	// Build the children structure starting from the root directory
	// The `buildTreeNodes` function now handles the recursive part.
	// `isRoot` for ShouldIgnore context is effectively true for this first level.
	children, err := buildTreeNodes(absRootDir, ignorePatterns, true) // Pass true for isRoot here
	if err != nil {
		// If the root directory itself cannot be read, return the error
		return nil, fmt.Errorf("building tree for %s: %w", rootDir, err)
	}
	rootNode.Children = children

	// Return a slice containing just the single root node
	return []*types.TreeOutputNode{rootNode}, nil
}

// buildTreeNodes recursively builds the node structure for a directory's contents.
// It now takes an isRoot parameter for context with ShouldIgnore.
func buildTreeNodes(currentPath string, ignorePatterns []string, isRoot bool) ([]*types.TreeOutputNode, error) {
	var nodes []*types.TreeOutputNode

	directoryEntries, readError := os.ReadDir(currentPath)
	if readError != nil {
		// Return error if we cannot read the directory contents
		return nil, fmt.Errorf("reading directory %s: %w", currentPath, readError)
	}

	// Filter entries based on ignore patterns first
	filteredEntries := make([]os.DirEntry, 0, len(directoryEntries))
	for _, entry := range directoryEntries {
		// Apply ignore rules using the correct isRoot context
		if !utils.ShouldIgnore(entry, ignorePatterns, isRoot) {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	// Process filtered entries
	for _, entry := range filteredEntries {
		childPath := filepath.Join(currentPath, entry.Name())
		node := &types.TreeOutputNode{
			Path: childPath, // Store absolute path for clarity in JSON
			Name: entry.Name(),
		}

		if entry.IsDir() {
			node.Type = "directory"
			// Recurse for subdirectories, passing `isRoot=false`
			childNodes, err := buildTreeNodes(childPath, ignorePatterns, false)
			if err != nil {
				// Warn about skipping subdirectory but continue processing siblings
				fmt.Fprintf(os.Stderr, "Warning: Skipping subdirectory %s due to error: %v\n", childPath, err)
				// Optionally add the node itself but mark it as incomplete, or just skip adding children
				node.Children = nil // Ensure children is nil if skipped
			} else {
				// Only assign children if there are any to avoid "children": [] in JSON for empty dirs
				if len(childNodes) > 0 {
					node.Children = childNodes
				}
			}
		} else {
			node.Type = "file"
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
