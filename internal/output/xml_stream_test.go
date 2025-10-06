package output_test

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

const indentSpacerValue = "  "

func TestXMLStreamRendererOutputsSingleRoot(testingInstance *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewXMLStreamRenderer(&stdout, &stderr, types.CommandContent, 1)

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
		if handleError := renderer.Handle(event); handleError != nil {
			testingInstance.Fatalf("handle event failed: %v", handleError)
		}
		if event.Kind == stream.EventKindTree && event.Tree != nil && event.Tree.Path == rootPath {
			if !bytes.HasPrefix(stdout.Bytes(), []byte(xml.Header)) {
				testingInstance.Fatalf("expected xml header at start of streamed output")
			}
		}
	}

	if flushError := renderer.Flush(); flushError != nil {
		testingInstance.Fatalf("flush failed: %v", flushError)
	}

	if !bytes.Contains(stderr.Bytes(), []byte("xml warning")) {
		testingInstance.Fatalf("expected warning on stderr, got %q", stderr.String())
	}

	var decoded types.TreeOutputNode
	if decodeError := xml.Unmarshal(stdout.Bytes(), &decoded); decodeError != nil {
		testingInstance.Fatalf("failed to decode xml output: %v\noutput: %s", decodeError, stdout.String())
	}

	if decoded.Path != tree.Path {
		testingInstance.Fatalf("expected root path %q, got %q", tree.Path, decoded.Path)
	}
	if len(decoded.Children) != 1 {
		testingInstance.Fatalf("expected one child, got %d", len(decoded.Children))
	}
	if decoded.Children[0] == nil || decoded.Children[0].Path != filePath {
		testingInstance.Fatalf("unexpected child node: %+v", decoded.Children[0])
	}
}

func TestXMLStreamRendererOutputsMultipleRootsAsResults(testingInstance *testing.T) {
	var stdout bytes.Buffer
	renderer := output.NewXMLStreamRenderer(&stdout, nil, types.CommandTree, 2)

	firstPath := "/tmp/first"
	secondPath := "/tmp/second"

	firstNode := &types.TreeOutputNode{Path: firstPath, Name: "first", Type: types.NodeTypeDirectory}
	secondNode := &types.TreeOutputNode{Path: secondPath, Name: "second", Type: types.NodeTypeDirectory}

	firstEncoded, firstEncodeError := xml.MarshalIndent(firstNode, indentSpacerValue, indentSpacerValue)
	if firstEncodeError != nil {
		testingInstance.Fatalf("marshal first node: %v", firstEncodeError)
	}
	secondEncoded, secondEncodeError := xml.MarshalIndent(secondNode, indentSpacerValue, indentSpacerValue)
	if secondEncodeError != nil {
		testingInstance.Fatalf("marshal second node: %v", secondEncodeError)
	}

	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: firstPath},
		{Kind: stream.EventKindTree, Path: firstPath, Tree: firstNode},
		{Kind: stream.EventKindDone},
		{Kind: stream.EventKindStart, Path: secondPath},
		{Kind: stream.EventKindTree, Path: secondPath, Tree: secondNode},
		{Kind: stream.EventKindDone},
	}

	header := xml.Header + "<results>\n"

	for _, event := range events {
		if handleError := renderer.Handle(event); handleError != nil {
			testingInstance.Fatalf("handle event failed: %v", handleError)
		}
		if event.Kind == stream.EventKindTree && event.Tree != nil && event.Tree.Path == firstPath {
			expectedPrefix := header + string(firstEncoded) + "\n"
			if stdout.String() != expectedPrefix {
				testingInstance.Fatalf("unexpected streamed xml prefix: got %q, want %q", stdout.String(), expectedPrefix)
			}
		}
	}

	if flushError := renderer.Flush(); flushError != nil {
		testingInstance.Fatalf("flush failed: %v", flushError)
	}

	expectedOutput := header + string(firstEncoded) + "\n" + string(secondEncoded) + "\n</results>\n"
	if stdout.String() != expectedOutput {
		testingInstance.Fatalf("unexpected final xml: got %q, want %q", stdout.String(), expectedOutput)
	}

	var wrapper struct {
		Nodes []types.TreeOutputNode `xml:"node"`
	}

	if decodeError := xml.Unmarshal(stdout.Bytes(), &wrapper); decodeError != nil {
		testingInstance.Fatalf("failed to decode xml wrapper: %v\noutput: %s", decodeError, stdout.String())
	}

	if len(wrapper.Nodes) != 2 {
		testingInstance.Fatalf("expected two nodes, got %d", len(wrapper.Nodes))
	}
	if wrapper.Nodes[0].Path != firstPath {
		testingInstance.Fatalf("expected first path %q, got %q", firstPath, wrapper.Nodes[0].Path)
	}
	if wrapper.Nodes[1].Path != secondPath {
		testingInstance.Fatalf("expected second path %q, got %q", secondPath, wrapper.Nodes[1].Path)
	}
}
