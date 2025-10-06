package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

const jsonIndentSpacer = "  "

func TestJSONStreamRendererOutputsSingleRoot(testingInstance *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewJSONStreamRenderer(&stdout, &stderr, types.CommandContent, 1)

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
		{Kind: stream.EventKindWarning, Message: &stream.LogEvent{Level: "warning", Message: "json warning"}},
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
			var immediate types.TreeOutputNode
			if decodeError := json.Unmarshal(stdout.Bytes(), &immediate); decodeError != nil {
				testingInstance.Fatalf("failed to decode streamed json: %v\noutput: %s", decodeError, stdout.String())
			}
			if immediate.Path != tree.Path {
				testingInstance.Fatalf("expected streamed path %q, got %q", tree.Path, immediate.Path)
			}
		}
	}

	if flushError := renderer.Flush(); flushError != nil {
		testingInstance.Fatalf("flush failed: %v", flushError)
	}

	if !bytes.Contains(stderr.Bytes(), []byte("json warning")) {
		testingInstance.Fatalf("expected warning on stderr, got %q", stderr.String())
	}

	var decoded types.TreeOutputNode
	if decodeError := json.Unmarshal(stdout.Bytes(), &decoded); decodeError != nil {
		testingInstance.Fatalf("failed to decode json output: %v\noutput: %s", decodeError, stdout.String())
	}

	if decoded.Path != tree.Path {
		testingInstance.Fatalf("expected root path %q, got %q", tree.Path, decoded.Path)
	}
	if decoded.TotalFiles != tree.TotalFiles {
		testingInstance.Fatalf("expected total files %d, got %d", tree.TotalFiles, decoded.TotalFiles)
	}
	if len(decoded.Children) != 1 {
		testingInstance.Fatalf("expected one child, got %d", len(decoded.Children))
	}
	if decoded.Children[0] == nil || decoded.Children[0].Path != filePath {
		testingInstance.Fatalf("unexpected child node: %+v", decoded.Children[0])
	}
}

func TestJSONStreamRendererOutputsMultipleRootsAsArray(testingInstance *testing.T) {
	var stdout bytes.Buffer
	renderer := output.NewJSONStreamRenderer(&stdout, nil, types.CommandTree, 2)

	firstPath := "/tmp/first"
	secondPath := "/tmp/second"

	firstNode := &types.TreeOutputNode{Path: firstPath, Name: "first", Type: types.NodeTypeDirectory}
	secondNode := &types.TreeOutputNode{Path: secondPath, Name: "second", Type: types.NodeTypeDirectory}

	firstEncoded, firstEncodeError := json.MarshalIndent(firstNode, jsonIndentSpacer, jsonIndentSpacer)
	if firstEncodeError != nil {
		testingInstance.Fatalf("marshal first node: %v", firstEncodeError)
	}
	secondEncoded, secondEncodeError := json.MarshalIndent(secondNode, jsonIndentSpacer, jsonIndentSpacer)
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

	expectedPrefix := "[\n" + string(firstEncoded)

	for _, event := range events {
		if handleError := renderer.Handle(event); handleError != nil {
			testingInstance.Fatalf("handle event failed: %v", handleError)
		}
		if event.Kind == stream.EventKindTree && event.Tree != nil && event.Tree.Path == firstPath {
			if stdout.String() != expectedPrefix {
				testingInstance.Fatalf("unexpected streamed prefix: got %q, want %q", stdout.String(), expectedPrefix)
			}
		}
	}

	if flushError := renderer.Flush(); flushError != nil {
		testingInstance.Fatalf("flush failed: %v", flushError)
	}

	expectedOutput := "[\n" + string(firstEncoded) + ",\n" + string(secondEncoded) + "\n]\n"
	if stdout.String() != expectedOutput {
		testingInstance.Fatalf("unexpected final array: got %q, want %q", stdout.String(), expectedOutput)
	}

	var decoded []types.TreeOutputNode
	if decodeError := json.Unmarshal(stdout.Bytes(), &decoded); decodeError != nil {
		testingInstance.Fatalf("failed to decode json array: %v\noutput: %s", decodeError, stdout.String())
	}

	if len(decoded) != 2 {
		testingInstance.Fatalf("expected two roots, got %d", len(decoded))
	}
	if decoded[0].Path != firstPath {
		testingInstance.Fatalf("expected first root path %q, got %q", firstPath, decoded[0].Path)
	}
	if decoded[1].Path != secondPath {
		testingInstance.Fatalf("expected second root path %q, got %q", secondPath, decoded[1].Path)
	}
}
