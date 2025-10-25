package output

import (
	"path/filepath"

	"github.com/tyemirov/ctx/internal/types"
)

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func pathsEqual(a, b string) bool {
	return normalizePath(a) == normalizePath(b)
}

func cloneTreeNode(node *types.TreeOutputNode) *types.TreeOutputNode {
	if node == nil {
		return nil
	}

	cloned := *node

	if len(node.Children) > 0 {
		cloned.Children = make([]*types.TreeOutputNode, len(node.Children))
		for index, child := range node.Children {
			if child == nil {
				continue
			}
			cloned.Children[index] = cloneTreeNode(child)
		}
	} else {
		cloned.Children = nil
	}

	if len(node.Documentation) > 0 {
		cloned.Documentation = append([]types.DocumentationEntry(nil), node.Documentation...)
	} else {
		cloned.Documentation = nil
	}

	return &cloned
}
