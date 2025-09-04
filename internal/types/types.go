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
	FormatXML  = "xml"
)

// ValidatedPath is an absolute input path that already passed existence checks.
type ValidatedPath struct {
	AbsolutePath string
	IsDir        bool
}

// DocumentationEntry is a single piece of documentation attached to output.
type DocumentationEntry struct {
	Kind string `json:"type" xml:"type"`
	Name string `json:"name" xml:"name"`
	Doc  string `json:"documentation" xml:"documentation"`
}

// FileOutput represents one file returned by the content command.
type FileOutput struct {
	Path          string               `json:"path" xml:"path"`
	Type          string               `json:"type" xml:"type"`
	Content       string               `json:"content" xml:"content"`
	MimeType      string               `json:"mimeType,omitempty" xml:"mimeType,omitempty"`
	Documentation []DocumentationEntry `json:"documentation,omitempty" xml:"documentation>entry,omitempty"`
}

// TreeOutputNode represents a node of a directory tree returned by the tree command.
type TreeOutputNode struct {
	Path     string            `json:"path" xml:"path"`
	Name     string            `json:"name" xml:"name"`
	Type     string            `json:"type" xml:"type"`
	MimeType string            `json:"mimeType,omitempty" xml:"mimeType,omitempty"`
	Children []*TreeOutputNode `json:"children,omitempty" xml:"children>node,omitempty"`
}

// CallChainOutput is the result of the callchain command.
type CallChainOutput struct {
	TargetFunction string               `json:"targetFunction" xml:"targetFunction"`
	Callers        []string             `json:"callers" xml:"callers>caller"`
	Callees        *[]string            `json:"callees,omitempty" xml:"callees>callee,omitempty"`
	Functions      map[string]string    `json:"functions" xml:"-"`
	Documentation  []DocumentationEntry `json:"documentation,omitempty" xml:"documentation>entry,omitempty"`
}
