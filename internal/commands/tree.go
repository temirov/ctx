// Package commands contains the core logic for data collection for each command.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/tokenizer"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

const (
	// warningSkipSubdirFormat is used when a subdirectory cannot be processed.
	warningSkipSubdirFormat = "Warning: Skipping subdirectory %s due to error: %v\n"
	// warningStatPathFormat is used when file information cannot be retrieved.
	warningStatPathFormat = "Warning: unable to stat %s: %v\n"

	// errorAbsolutePathFormat is used when the absolute path cannot be determined.
	errorAbsolutePathFormat = "getting absolute path for %s: %w"

	// errorBuildTreeFormat is used when building the tree fails.
	errorBuildTreeFormat = "building tree for %s: %w"

	// errorReadDirectoryFormat is used when a directory cannot be read.
	errorReadDirectoryFormat = "reading directory %s: %w"

	// warningTokenCountFormat is used when token estimation fails for a file.
	warningTokenCountFormat = "Warning: failed to count tokens for %s: %v\n"
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
	rootInfo, rootStatError := os.Stat(absoluteRootDirPath)
	if rootStatError == nil {
		rootNode.LastModified = utils.FormatTimestamp(rootInfo.ModTime())
	}

	children, buildError := treeBuilder.buildTreeNodes(absoluteRootDirPath, absoluteRootDirPath)
	if buildError != nil {
		return nil, fmt.Errorf(errorBuildTreeFormat, rootDirectoryPath, buildError)
	}
	rootNode.Children = children
	if treeBuilder.IncludeSummary {
		files, bytes, tokens := treeBuilder.collectSummary(rootNode.Children)
		applySummary(rootNode, files, bytes, tokens)
	}

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

		entryInfo, infoError := directoryEntry.Info()
		if infoError != nil {
			fmt.Fprintf(os.Stderr, warningStatPathFormat, childPath, infoError)
		} else {
			node.LastModified = utils.FormatTimestamp(entryInfo.ModTime())
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
			if treeBuilder.IncludeSummary {
				files, bytes, tokens := treeBuilder.collectSummary(node.Children)
				applySummary(node, files, bytes, tokens)
			}
		} else {
			childMimeType := utils.DetectMimeType(childPath)
			isBinaryFile := utils.IsFileBinary(childPath)
			if isBinaryFile {
				node.Type = types.NodeTypeBinary
			} else {
				node.Type = types.NodeTypeFile
			}
			node.MimeType = childMimeType
			if infoError == nil {
				node.Size = utils.FormatFileSize(entryInfo.Size())
				node.SizeBytes = entryInfo.Size()
			}
			if treeBuilder.TokenCounter != nil && node.Type != types.NodeTypeBinary {
				tokenResult, tokenErr := tokenizer.CountFile(treeBuilder.TokenCounter, childPath)
				if tokenErr != nil {
					fmt.Fprintf(os.Stderr, warningTokenCountFormat, childPath, tokenErr)
				} else if tokenResult.Counted {
					node.Tokens = tokenResult.Tokens
				}
			}
			if treeBuilder.IncludeSummary {
				applySummary(node, 1, node.SizeBytes, node.Tokens)
			}
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// collectSummary returns the aggregate file count, size, and tokens for the provided children.
func (treeBuilder *TreeBuilder) collectSummary(children []*types.TreeOutputNode) (int, int64, int) {
	var totalFiles int
	var totalBytes int64
	var totalTokens int
	for _, child := range children {
		if child == nil {
			continue
		}
		files := child.TotalFiles
		bytes := child.SizeBytes
		tokens := child.TotalTokens
		if child.Type == types.NodeTypeFile || child.Type == types.NodeTypeBinary {
			if files == 0 {
				files = 1
			}
			if tokens == 0 {
				tokens = child.Tokens
			}
		}
		totalFiles += files
		totalBytes += bytes
		totalTokens += tokens
	}
	return totalFiles, totalBytes, totalTokens
}

// applySummary stores aggregate counts, bytes, and tokens on the node.
func applySummary(node *types.TreeOutputNode, totalFiles int, totalBytes int64, totalTokens int) {
	node.TotalFiles = totalFiles
	node.SizeBytes = totalBytes
	node.TotalSize = utils.FormatFileSize(totalBytes)
	node.TotalTokens = totalTokens
}
