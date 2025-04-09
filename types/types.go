// Package types defines shared data structures used across the content tool.
package types

// ValidatedPath stores information about a resolved and validated input path.
type ValidatedPath struct {
	AbsolutePath string
	IsDir        bool
}

// FileOutput represents the data for a single file's content, used for JSON output.
type FileOutput struct {
	Path    string `json:"path"`
	Type    string `json:"type"` // Always "file" for this struct
	Content string `json:"content"`
}

// TreeOutputNode represents a node in the directory tree structure for JSON output.
type TreeOutputNode struct {
	Path     string            `json:"path"`               // Absolute path
	Name     string            `json:"name"`               // Base name of the file/dir
	Type     string            `json:"type"`               // "file" or "directory"
	Children []*TreeOutputNode `json:"children,omitempty"` // Omit if empty (for files or empty dirs)
}
