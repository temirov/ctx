// Package types defines every cross‑package data structure used by the ctx CLI.
package types

import "encoding/xml"

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
	Size          string               `json:"size,omitempty" xml:"size,omitempty"`
	SizeBytes     int64                `json:"-" xml:"-"`
	LastModified  string               `json:"lastModified,omitempty" xml:"lastModified,omitempty"`
	MimeType      string               `json:"mimeType,omitempty" xml:"mimeType,omitempty"`
	Tokens        int                  `json:"tokens,omitempty" xml:"tokens,omitempty"`
	Model         string               `json:"model,omitempty" xml:"model,omitempty"`
	Documentation []DocumentationEntry `json:"documentation,omitempty" xml:"documentation>entry,omitempty"`
}

// TreeOutputNode represents a node of a directory tree returned by the tree command.
type TreeOutputNode struct {
	XMLName       xml.Name             `json:"-" xml:"node"`
	Path          string               `json:"path" xml:"path"`
	Name          string               `json:"name" xml:"name"`
	Type          string               `json:"type" xml:"type"`
	Size          string               `json:"size,omitempty" xml:"size,omitempty"`
	SizeBytes     int64                `json:"-" xml:"-"`
	LastModified  string               `json:"lastModified,omitempty" xml:"lastModified,omitempty"`
	MimeType      string               `json:"mimeType,omitempty" xml:"mimeType,omitempty"`
	Tokens        int                  `json:"tokens,omitempty" xml:"tokens,omitempty"`
	Model         string               `json:"model,omitempty" xml:"model,omitempty"`
	Children      []*TreeOutputNode    `json:"children,omitempty" xml:"children>node,omitempty"`
	TotalFiles    int                  `json:"totalFiles,omitempty" xml:"totalFiles,omitempty"`
	TotalSize     string               `json:"totalSize,omitempty" xml:"totalSize,omitempty"`
	TotalTokens   int                  `json:"totalTokens,omitempty" xml:"totalTokens,omitempty"`
	Content       string               `json:"content,omitempty" xml:"content,omitempty"`
	Documentation []DocumentationEntry `json:"documentation,omitempty" xml:"documentation>entry,omitempty"`
}

// CallChainOutput is the result of the callchain command.
type CallChainOutput struct {
	TargetFunction string               `json:"targetFunction" xml:"targetFunction"`
	Callers        []string             `json:"callers" xml:"callers>caller"`
	Callees        *[]string            `json:"callees,omitempty" xml:"callees>callee,omitempty"`
	Functions      map[string]string    `json:"functions" xml:"-"`
	Documentation  []DocumentationEntry `json:"documentation,omitempty" xml:"documentation>entry,omitempty"`
}

// OutputSummary captures aggregate information about rendered files.
type OutputSummary struct {
	TotalFiles  int    `json:"totalFiles" xml:"totalFiles"`
	TotalSize   string `json:"totalSize" xml:"totalSize"`
	TotalTokens int    `json:"totalTokens,omitempty" xml:"totalTokens,omitempty"`
	Model       string `json:"model,omitempty" xml:"model,omitempty"`
}
