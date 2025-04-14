// Package types defines shared data structures and constants used across the content tool.
package types

// Constants for node types used in output structures.
const (
	NodeTypeFile      = "file"
	NodeTypeDirectory = "directory"
)

// Constants for command names.
const (
	CommandTree      = "tree"
	CommandContent   = "content"
	CommandCallChain = "callchain"
)

// Constants for output formats.
const (
	FormatRaw  = "raw"
	FormatJSON = "json"
)

// ValidatedPath stores information about a resolved and validated input path.
type ValidatedPath struct {
	AbsolutePath string
	IsDir        bool
}

// FileOutput represents the data for a single file's content, used for JSON output.
type FileOutput struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// TreeOutputNode represents a node in the directory tree structure for JSON output.
type TreeOutputNode struct {
	Path     string            `json:"path"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Children []*TreeOutputNode `json:"children,omitempty"`
}

// CallChainOutput represents the call chain details of a target function.
type CallChainOutput struct {
	TargetFunction string            `json:"targetFunction"`
	Callers        []string          `json:"callers"`
	Callees        *[]string         `json:"callees,omitempty"`
	Functions      map[string]string `json:"functions,omitempty"`
}
