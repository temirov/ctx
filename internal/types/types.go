// Package types defines every cross‑package data structure used by the ctx CLI.
package types

const (
	NodeTypeFile      = "file"
	NodeTypeDirectory = "directory"
	NodeTypeBinary    = "binary"

	CommandTree      = "tree"
	CommandContent   = "content"
	CommandCallChain = "callchain"

	FormatRaw  = "raw"
	FormatJSON = "json"
)

// ValidatedPath is an absolute input path that already passed existence checks.
type ValidatedPath struct {
	AbsolutePath string
	IsDir        bool
}

// DocumentationEntry is a single piece of documentation attached to output.
type DocumentationEntry struct {
	Kind string `json:"type"`
	Name string `json:"name"`
	Doc  string `json:"documentation"`
}

// FileOutput represents one file returned by the content command.
type FileOutput struct {
	Path string `json:"path"`
	Type string `json:"type"`
	// MimeType is the MIME type of the file when the type is NodeTypeBinary.
	MimeType      string               `json:"mimeType,omitempty"`
	Content       string               `json:"content"`
	Documentation []DocumentationEntry `json:"documentation,omitempty"`
}

// TreeOutputNode represents a node of a directory tree returned by the tree command.
type TreeOutputNode struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
	// MimeType is the MIME type when the node represents binary content.
	MimeType string            `json:"mimeType,omitempty"`
	Children []*TreeOutputNode `json:"children,omitempty"`
}

// CallChainOutput is the result of the callchain command.
type CallChainOutput struct {
	TargetFunction string               `json:"targetFunction"`
	Callers        []string             `json:"callers"`
	Callees        *[]string            `json:"callees,omitempty"`
	Functions      map[string]string    `json:"functions"`
	Documentation  []DocumentationEntry `json:"documentation,omitempty"`
}
