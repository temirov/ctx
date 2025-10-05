package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

// BuildContentTree constructs a directory tree representation for content command outputs.
func BuildContentTree(rootPath string, files []types.FileOutput, includeSummary bool) (*types.TreeOutputNode, error) {
	if len(files) == 0 {
		return buildEmptyRoot(rootPath, includeSummary)
	}
	absRoot, absError := filepath.Abs(rootPath)
	if absError != nil {
		return nil, fmt.Errorf("abs failed for %s: %w", rootPath, absError)
	}
	rootInfo, rootStatError := os.Stat(absRoot)
	if rootStatError == nil && !rootInfo.IsDir() {
		for _, file := range files {
			if filepath.Clean(file.Path) == absRoot {
				node := &types.TreeOutputNode{
					Path:          file.Path,
					Name:          filepath.Base(file.Path),
					Type:          file.Type,
					Size:          file.Size,
					SizeBytes:     file.SizeBytes,
					LastModified:  file.LastModified,
					MimeType:      file.MimeType,
					Content:       file.Content,
					Documentation: file.Documentation,
					Tokens:        file.Tokens,
				}
				if includeSummary {
					applySummary(node, 1, file.SizeBytes, file.Tokens)
				}
				return node, nil
			}
		}
	}
	rootNode := &types.TreeOutputNode{
		Path: absRoot,
		Name: filepath.Base(absRoot),
		Type: types.NodeTypeDirectory,
	}
	if rootStatError == nil {
		rootNode.LastModified = utils.FormatTimestamp(rootInfo.ModTime())
	}
	nodeByPath := map[string]*types.TreeOutputNode{absRoot: rootNode}
	for _, file := range files {
		ensureDirectories(nodeByPath, absRoot, filepath.Dir(file.Path))
		parentPath := filepath.Dir(file.Path)
		parentNode := nodeByPath[parentPath]
		if parentNode == nil {
			continue
		}
		fileNode := &types.TreeOutputNode{
			Path:          file.Path,
			Name:          filepath.Base(file.Path),
			Type:          file.Type,
			Size:          file.Size,
			SizeBytes:     file.SizeBytes,
			LastModified:  file.LastModified,
			MimeType:      file.MimeType,
			Content:       file.Content,
			Documentation: file.Documentation,
			Tokens:        file.Tokens,
		}
		if includeSummary {
			applySummary(fileNode, 1, file.SizeBytes, file.Tokens)
		}
		parentNode.Children = append(parentNode.Children, fileNode)
	}
	sortTreeChildren(rootNode)
	if includeSummary {
		populateDirectorySummaries(rootNode)
	}
	return rootNode, nil
}

func buildEmptyRoot(rootPath string, includeSummary bool) (*types.TreeOutputNode, error) {
	absRoot, absError := filepath.Abs(rootPath)
	if absError != nil {
		return nil, fmt.Errorf("abs failed for %s: %w", rootPath, absError)
	}
	info, statError := os.Stat(absRoot)
	node := &types.TreeOutputNode{
		Path: absRoot,
		Name: filepath.Base(absRoot),
	}
	if statError == nil {
		if info.IsDir() {
			node.Type = types.NodeTypeDirectory
		} else {
			node.Type = types.NodeTypeFile
		}
		node.LastModified = utils.FormatTimestamp(info.ModTime())
	} else {
		node.Type = types.NodeTypeDirectory
	}
	if includeSummary {
		applySummary(node, 0, 0, 0)
	}
	return node, nil
}

func ensureDirectories(nodeByPath map[string]*types.TreeOutputNode, rootPath string, currentPath string) {
	if currentPath == "" || currentPath == rootPath {
		return
	}
	if _, exists := nodeByPath[currentPath]; exists {
		return
	}
	ensureDirectories(nodeByPath, rootPath, filepath.Dir(currentPath))
	info, statError := os.Stat(currentPath)
	node := &types.TreeOutputNode{
		Path: currentPath,
		Name: filepath.Base(currentPath),
		Type: types.NodeTypeDirectory,
	}
	if statError == nil {
		node.LastModified = utils.FormatTimestamp(info.ModTime())
	}
	parent := nodeByPath[filepath.Dir(currentPath)]
	if parent != nil {
		parent.Children = append(parent.Children, node)
	}
	nodeByPath[currentPath] = node
}

func sortTreeChildren(node *types.TreeOutputNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Name < node.Children[j].Name
	})
	for _, child := range node.Children {
		sortTreeChildren(child)
	}
}

func populateDirectorySummaries(node *types.TreeOutputNode) (int, int64, int) {
	if node.Type == types.NodeTypeFile || node.Type == types.NodeTypeBinary {
		if node.TotalFiles == 0 {
			applySummary(node, 1, node.SizeBytes, node.Tokens)
		}
		return node.TotalFiles, node.SizeBytes, node.TotalTokens
	}
	var totalFiles int
	var totalBytes int64
	var totalTokens int
	for _, child := range node.Children {
		childFiles, childBytes, childTokens := populateDirectorySummaries(child)
		totalFiles += childFiles
		totalBytes += childBytes
		totalTokens += childTokens
	}
	applySummary(node, totalFiles, totalBytes, totalTokens)
	return totalFiles, totalBytes, totalTokens
}
