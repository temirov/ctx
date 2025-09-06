// Package commands contains the core logic for data collection for each command.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

const (
	// warningSkipSubdirFormat is used when a subdirectory cannot be processed.
	warningSkipSubdirFormat = "Warning: Skipping subdirectory %s due to error: %v\n"

	// errorAbsolutePathFormat is used when the absolute path cannot be determined.
	errorAbsolutePathFormat = "getting absolute path for %s: %w"

	// errorBuildTreeFormat is used when building the tree fails.
	errorBuildTreeFormat = "building tree for %s: %w"

	// errorReadDirectoryFormat is used when a directory cannot be read.
	errorReadDirectoryFormat = "reading directory %s: %w"
)

// GetTreeData generates the tree structure data for a given directory.
// It returns a slice containing a single root node representing the directory.
// Warnings for skipped subdirectories are printed to stderr.
func (treeBuilder *TreeBuilder) GetTreeData(rootDirectoryPath string) ([]*types.TreeOutputNode, error) {
	absoluteRootDirPath, absolutePathError := filepath.Abs(rootDirectoryPath)
	if absolutePathError != nil {
		return nil, fmt.Errorf(errorAbsolutePathFormat, rootDirectoryPath, absolutePathError)
	}

	rootNode := &types.TreeOutputNode{
		Path: absoluteRootDirPath,
		Name: filepath.Base(absoluteRootDirPath),
		Type: types.NodeTypeDirectory,
	}

	children, buildError := treeBuilder.buildTreeNodes(absoluteRootDirPath, absoluteRootDirPath)
	if buildError != nil {
		return nil, fmt.Errorf(errorBuildTreeFormat, rootDirectoryPath, buildError)
	}
	rootNode.Children = children

	return []*types.TreeOutputNode{rootNode}, nil
}

// buildTreeNodes recursively builds child nodes for the directory tree.
func (treeBuilder *TreeBuilder) buildTreeNodes(currentDirectoryPath string, rootDirectoryPath string) ([]*types.TreeOutputNode, error) {
	var nodes []*types.TreeOutputNode

	directoryEntries, readDirectoryError := os.ReadDir(currentDirectoryPath)
	if readDirectoryError != nil {
		return nil, fmt.Errorf(errorReadDirectoryFormat, currentDirectoryPath, readDirectoryError)
	}

	for _, directoryEntry := range directoryEntries {
		childPath := filepath.Join(currentDirectoryPath, directoryEntry.Name())
		relativeChildPath := utils.RelativePathOrSelf(childPath, rootDirectoryPath)
		if utils.ShouldIgnoreByPath(relativeChildPath, treeBuilder.IgnorePatterns) {
			continue
		}

		node := &types.TreeOutputNode{
			Path: childPath,
			Name: directoryEntry.Name(),
		}

		if directoryEntry.IsDir() {
			node.Type = types.NodeTypeDirectory
			childNodes, buildError := treeBuilder.buildTreeNodes(childPath, rootDirectoryPath)
			if buildError != nil {
				fmt.Fprintf(os.Stderr, warningSkipSubdirFormat, childPath, buildError)
				node.Children = nil
			} else {
				node.Children = childNodes
			}
		} else {
			detectedMimeType := utils.DetectMimeType(childPath)
			if utils.IsFileBinary(childPath) {
				node.Type = types.NodeTypeBinary
			} else {
				node.Type = types.NodeTypeFile
			}
			node.MimeType = detectedMimeType
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
