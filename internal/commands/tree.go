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
	// errorAbsolutePathFormat is used when the absolute path cannot be determined.
	errorAbsolutePathFormat = "getting absolute path for %s: %w"

	// errorBuildTreeFormat is used when building the tree fails.
	errorBuildTreeFormat = "building tree for %s: %w"
)

// GetTreeData rewrites the directory traversal leveraging StreamTree to build structured nodes.
func (treeBuilder *TreeBuilder) GetTreeData(rootDirectoryPath string) ([]*types.TreeOutputNode, error) {
	absoluteRootDirPath, absolutePathError := filepath.Abs(rootDirectoryPath)
	if absolutePathError != nil {
		return nil, fmt.Errorf(errorAbsolutePathFormat, rootDirectoryPath, absolutePathError)
	}

	collector := &treeCollector{
		includeSummary: treeBuilder.IncludeSummary,
		tokenModel:     treeBuilder.TokenModel,
	}

	options := TreeStreamOptions{
		Root:           absoluteRootDirPath,
		IgnorePatterns: treeBuilder.IgnorePatterns,
		TokenCounter:   treeBuilder.TokenCounter,
		TokenModel:     treeBuilder.TokenModel,
		Warn: func(message string) {
			fmt.Fprint(os.Stderr, message)
		},
	}

	if err := StreamTree(options, collector.handleEvent); err != nil {
		return nil, fmt.Errorf(errorBuildTreeFormat, rootDirectoryPath, err)
	}

	if len(collector.roots) == 0 {
		return nil, nil
	}

	return collector.roots, nil
}

type treeCollector struct {
	includeSummary bool
	tokenModel     string
	stack          []*types.TreeOutputNode
	roots          []*types.TreeOutputNode
}

func (collector *treeCollector) handleEvent(event TreeEvent) error {
	switch event.Kind {
	case TreeEventEnterDir:
		node := &types.TreeOutputNode{
			Path:         event.Directory.Path,
			Name:         event.Directory.Name,
			Type:         types.NodeTypeDirectory,
			LastModified: event.Directory.LastModified,
		}
		if len(collector.stack) > 0 {
			parent := collector.stack[len(collector.stack)-1]
			parent.Children = append(parent.Children, node)
		}
		collector.stack = append(collector.stack, node)
	case TreeEventFile:
		node := collector.makeFileNode(event.File)
		collector.attachNode(node)
	case TreeEventLeaveDir:
		if len(collector.stack) == 0 {
			return fmt.Errorf("tree collector stack underflow")
		}
		node := collector.stack[len(collector.stack)-1]
		collector.stack = collector.stack[:len(collector.stack)-1]
		if collector.includeSummary {
			summary := event.Directory.Summary
			applySummary(node, summary.Files, summary.Bytes, summary.Tokens, collector.tokenModel)
		}
		if len(collector.stack) == 0 {
			collector.roots = append(collector.roots, node)
		}
	}
	return nil
}

func (collector *treeCollector) attachNode(node *types.TreeOutputNode) {
	if len(collector.stack) == 0 {
		collector.roots = append(collector.roots, node)
		return
	}
	parent := collector.stack[len(collector.stack)-1]
	parent.Children = append(parent.Children, node)
}

func (collector *treeCollector) makeFileNode(event *TreeFileEvent) *types.TreeOutputNode {
	node := &types.TreeOutputNode{
		Path:         event.Path,
		Name:         event.Name,
		LastModified: event.LastModified,
		SizeBytes:    event.SizeBytes,
		Size:         utils.FormatFileSize(event.SizeBytes),
		MimeType:     event.MimeType,
		Tokens:       event.Tokens,
		Model:        event.Model,
	}
	if event.IsBinary {
		node.Type = types.NodeTypeBinary
	} else {
		node.Type = types.NodeTypeFile
	}
	if event.Tokens == 0 {
		node.Model = ""
	} else if node.Model == "" && collector.tokenModel != "" {
		node.Model = collector.tokenModel
	}
	return node
}

// applySummary stores aggregate counts, bytes, and tokens on the node.
func applySummary(node *types.TreeOutputNode, totalFiles int, totalBytes int64, totalTokens int, tokenModel string) {
	node.TotalFiles = totalFiles
	node.SizeBytes = totalBytes
	node.TotalSize = utils.FormatFileSize(totalBytes)
	node.TotalTokens = totalTokens
	if totalTokens > 0 && tokenModel != "" {
		node.Model = tokenModel
	} else if totalTokens == 0 {
		node.Model = ""
	}
}
