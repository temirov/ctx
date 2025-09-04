// Package commands contains the core logic for data collection for each command.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

// GetTreeData generates the tree structure data for a given directory.
// It returns a slice containing a single root node representing the directory.
// Warnings for skipped subdirectories are printed to stderr.
func GetTreeData(rootDirPath string, ignorePatterns []string) ([]*types.TreeOutputNode, error) {
	absoluteRootDirPath, err := filepath.Abs(rootDirPath)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path for %s: %w", rootDirPath, err)
	}

	rootNode := &types.TreeOutputNode{
		Path: absoluteRootDirPath,
		Name: filepath.Base(absoluteRootDirPath),
		Type: types.NodeTypeDirectory,
	}

	children, err := buildTreeNodes(absoluteRootDirPath, absoluteRootDirPath, ignorePatterns)
	if err != nil {
		return nil, fmt.Errorf("building tree for %s: %w", rootDirPath, err)
	}
	rootNode.Children = children

	return []*types.TreeOutputNode{rootNode}, nil
}

// buildTreeNodes recursively builds child nodes for the directory tree.
func buildTreeNodes(currentDirectoryPath string, rootDirectoryPath string, ignorePatterns []string) ([]*types.TreeOutputNode, error) {
	var nodes []*types.TreeOutputNode

	directoryEntries, readDirError := os.ReadDir(currentDirectoryPath)
	if readDirError != nil {
		return nil, fmt.Errorf("reading directory %s: %w", currentDirectoryPath, readDirError)
	}

	for _, entry := range directoryEntries {
		childPath := filepath.Join(currentDirectoryPath, entry.Name())
		relativeChildPath := utils.RelativePathOrSelf(childPath, rootDirectoryPath)
		if utils.ShouldIgnoreByPath(relativeChildPath, ignorePatterns) {
			continue
		}

		node := &types.TreeOutputNode{
			Path: childPath,
			Name: entry.Name(),
		}

		if entry.IsDir() {
			node.Type = types.NodeTypeDirectory
			childNodes, err := buildTreeNodes(childPath, rootDirectoryPath, ignorePatterns)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Skipping subdirectory %s due to error: %v\n", childPath, err)
				node.Children = nil
			} else {
				node.Children = childNodes
			}
		} else {
			if utils.IsFileBinary(childPath) {
				node.Type = types.NodeTypeBinary
			} else {
				node.Type = types.NodeTypeFile
			}
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
