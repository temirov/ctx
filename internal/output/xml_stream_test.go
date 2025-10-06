package output_test

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

func TestXMLStreamRendererOutputsSingleRoot(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewXMLStreamRenderer(&stdout, &stderr, types.CommandContent)

	rootPath := "/tmp/root"
	filePath := rootPath + "/file.txt"

	tree := &types.TreeOutputNode{
		Path:       rootPath,
		Name:       "root",
		Type:       types.NodeTypeDirectory,
		TotalFiles: 1,
		TotalSize:  "4b",
		Children: []*types.TreeOutputNode{
			{
				Path:      filePath,
				Name:      "file.txt",
				Type:      types.NodeTypeFile,
				Size:      "4b",
				SizeBytes: 4,
			},
		},
	}

	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: rootPath},
		{Kind: stream.EventKindWarning, Message: &stream.LogEvent{Level: "warning", Message: "xml warning"}},
		{Kind: stream.EventKindTree, Path: rootPath + "/ignored", Tree: &types.TreeOutputNode{Path: rootPath + "/ignored", Name: "ignored", Type: types.NodeTypeDirectory}},
		{Kind: stream.EventKindTree, Path: rootPath, Tree: tree},
		{Kind: stream.EventKindSummary, Summary: &stream.SummaryEvent{Files: 1, Bytes: 4}},
		{Kind: stream.EventKindDone},
	}

	for _, event := range events {
		if err := renderer.Handle(event); err != nil {
			t.Fatalf("handle event failed: %v", err)
		}
	}

	if err := renderer.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if !bytes.Contains(stderr.Bytes(), []byte("xml warning")) {
		t.Fatalf("expected warning on stderr, got %q", stderr.String())
	}

	var decoded types.TreeOutputNode
	if err := xml.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode xml output: %v\noutput: %s", err, stdout.String())
	}

	if decoded.Path != tree.Path {
		t.Fatalf("expected root path %q, got %q", tree.Path, decoded.Path)
	}
	if len(decoded.Children) != 1 {
		t.Fatalf("expected one child, got %d", len(decoded.Children))
	}
	if decoded.Children[0] == nil || decoded.Children[0].Path != filePath {
		t.Fatalf("unexpected child node: %+v", decoded.Children[0])
	}
}

func TestXMLStreamRendererOutputsMultipleRootsAsResults(t *testing.T) {
	var stdout bytes.Buffer
	renderer := output.NewXMLStreamRenderer(&stdout, nil, types.CommandTree)

	firstPath := "/tmp/first"
	secondPath := "/tmp/second"

	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: firstPath},
		{Kind: stream.EventKindTree, Path: firstPath, Tree: &types.TreeOutputNode{Path: firstPath, Name: "first", Type: types.NodeTypeDirectory}},
		{Kind: stream.EventKindDone},
		{Kind: stream.EventKindStart, Path: secondPath},
		{Kind: stream.EventKindTree, Path: secondPath, Tree: &types.TreeOutputNode{Path: secondPath, Name: "second", Type: types.NodeTypeDirectory}},
		{Kind: stream.EventKindDone},
	}

	for _, event := range events {
		if err := renderer.Handle(event); err != nil {
			t.Fatalf("handle event failed: %v", err)
		}
	}

	if err := renderer.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	var wrapper struct {
		Nodes []types.TreeOutputNode `xml:"node"`
		Items []types.TreeOutputNode `xml:"item"`
	}

	if err := xml.Unmarshal(stdout.Bytes(), &wrapper); err != nil {
		t.Fatalf("failed to decode xml wrapper: %v\noutput: %s", err, stdout.String())
	}

	nodes := wrapper.Nodes
	if len(nodes) == 0 && len(wrapper.Items) > 0 {
		nodes = wrapper.Items
	}

	if len(nodes) != 2 {
		t.Fatalf("expected two nodes, got %d", len(nodes))
	}
	if nodes[0].Path != firstPath {
		t.Fatalf("expected first path %q, got %q", firstPath, nodes[0].Path)
	}
	if nodes[1].Path != secondPath {
		t.Fatalf("expected second path %q, got %q", secondPath, nodes[1].Path)
	}
}
