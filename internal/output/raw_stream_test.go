package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

func TestRawStreamRendererStreamsTreeEvents(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewRawStreamRenderer(&stdout, &stderr, types.CommandTree, true)

	rootPath := "/tmp/root"
	filePath := rootPath + "/file.txt"

	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: rootPath},
		{Kind: stream.EventKindDirectory, Directory: &stream.DirectoryEvent{Phase: stream.DirectoryEnter, Path: rootPath, Depth: 0}},
		{Kind: stream.EventKindFile, File: &stream.FileEvent{Path: filePath, Depth: 1, Tokens: 2}},
		{Kind: stream.EventKindDirectory, Directory: &stream.DirectoryEvent{Phase: stream.DirectoryLeave, Path: rootPath, Depth: 0, Summary: &stream.SummaryEvent{Files: 1, Bytes: 4, Tokens: 2}}},
		{Kind: stream.EventKindSummary, Summary: &stream.SummaryEvent{Files: 1, Bytes: 4, Tokens: 2, Model: "stub"}},
		{Kind: stream.EventKindDone},
	}

	for index, event := range events {
		if err := renderer.Handle(event); err != nil {
			t.Fatalf("handle event %d failed: %v", index, err)
		}
	}

	if !strings.Contains(stdout.String(), rootPath) {
		t.Fatalf("expected directory path in output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[File] "+filePath) {
		t.Fatalf("expected file entry in output")
	}

	if err := renderer.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "Summary: 1 file") {
		t.Fatalf("expected summary line in output: %s", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output")
	}
}

func TestRawStreamRendererStreamsContentEvents(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewRawStreamRenderer(&stdout, &stderr, types.CommandContent, true)

	filePath := "/tmp/file.txt"
	treeNode := &types.TreeOutputNode{Path: "/tmp", Name: "tmp", Type: types.NodeTypeDirectory}
	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: "/tmp"},
		{Kind: stream.EventKindWarning, Message: &stream.LogEvent{Level: "warning", Message: "alert"}},
		{Kind: stream.EventKindFile, File: &stream.FileEvent{Path: filePath, Type: types.NodeTypeFile}},
		{Kind: stream.EventKindContentChunk, Chunk: &stream.ChunkEvent{Path: filePath, Data: "content", IsFinal: true}},
		{Kind: stream.EventKindSummary, Summary: &stream.SummaryEvent{Files: 1, Bytes: 7}},
		{Kind: stream.EventKindTree, Tree: treeNode},
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

	outputText := stdout.String()
	if !strings.Contains(outputText, "File: "+filePath) {
		t.Fatalf("expected file header in output")
	}
	if !strings.Contains(outputText, "content") {
		t.Fatalf("expected file content in output")
	}
	if !strings.Contains(outputText, "End of file: "+filePath) {
		t.Fatalf("expected end marker in output")
	}
	if !strings.Contains(outputText, "--- Directory Tree: /tmp ---") {
		t.Fatalf("expected tree header in output")
	}
	if !strings.Contains(outputText, "Summary: 1 file") {
		t.Fatalf("expected summary line in output")
	}
	if !strings.Contains(stderr.String(), "alert") {
		t.Fatalf("expected warning on stderr")
	}
}
