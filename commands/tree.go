// Package commands contains the core logic for data collection for each command.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/types"
	"github.com/temirov/ctx/utils"
)

// GetTreeData generates the tree structure data for a given directory.
// It returns a slice containing a single root node representing the directory.
// Warnings for skipped subdirectories are printed to stderr.
func GetTreeData(rootDirPath string, ignorePatterns, binaryPatterns []string) ([]*types.TreeOutputNode, error) {
	absoluteRootDirPath, err := filepath.Abs(rootDirPath)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path for %s: %w", rootDirPath, err)
	}

	rootNode := &types.TreeOutputNode{
		Path: absoluteRootDirPath,
		Name: filepath.Base(absoluteRootDirPath),
		Type: types.NodeTypeDirectory,
	}

	children, err := buildTreeNodes(absoluteRootDirPath, ignorePatterns, binaryPatterns, true)
	if err != nil {
		return nil, fmt.Errorf("building tree for %s: %w", rootDirPath, err)
	}
	rootNode.Children = children

	return []*types.TreeOutputNode{rootNode}, nil
}

func buildTreeNodes(currentDirectoryPath string, ignorePatterns, binaryPatterns []string, isRootLevel bool) ([]*types.TreeOutputNode, error) {
	var nodes []*types.TreeOutputNode

	directoryEntries, readDirError := os.ReadDir(currentDirectoryPath)
	if readDirError != nil {
		return nil, fmt.Errorf("reading directory %s: %w", currentDirectoryPath, readDirError)
	}

	for _, entry := range directoryEntries {
		if utils.ShouldIgnore(entry, ignorePatterns, isRootLevel) {
			continue
		}

		childPath := filepath.Join(currentDirectoryPath, entry.Name())
		node := &types.TreeOutputNode{
			Path: childPath,
			Name: entry.Name(),
		}

		if utils.ShouldTreatAsBinary(entry, binaryPatterns, isRootLevel) {
			node.Type = types.NodeTypeBinary
		} else if entry.IsDir() {
			node.Type = types.NodeTypeDirectory
			childNodes, err := buildTreeNodes(childPath, ignorePatterns, binaryPatterns, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Skipping subdirectory %s due to error: %v\n", childPath, err)
				node.Children = nil
			} else {
				node.Children = childNodes
			}
		} else {
			node.Type = types.NodeTypeFile
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
