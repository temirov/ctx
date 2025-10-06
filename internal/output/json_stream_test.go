package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

func TestJSONStreamRendererMatchesBatchOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewJSONStreamRenderer(&stdout, &stderr, types.CommandContent)

	root := "/tmp"
	filePath := root + "/file.txt"
	doc := []types.DocumentationEntry{{Kind: "pkg", Name: "fmt", Doc: "fmt docs"}}

	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: root},
		{Kind: stream.EventKindWarning, Message: &stream.LogEvent{Level: "warning", Message: "json warning"}},
		{Kind: stream.EventKindFile, File: &stream.FileEvent{
			Path:          filePath,
			Type:          types.NodeTypeFile,
			SizeBytes:     int64(len("chunk")),
			LastModified:  "now",
			MimeType:      "text/plain",
			Documentation: doc,
		}},
		{Kind: stream.EventKindContentChunk, Chunk: &stream.ChunkEvent{Path: filePath, Data: "chunk", IsFinal: true}},
		{Kind: stream.EventKindTree, Tree: &types.TreeOutputNode{Path: root, Type: types.NodeTypeDirectory}},
		{Kind: stream.EventKindSummary, Summary: &stream.SummaryEvent{Files: 1, Bytes: int64(len("chunk"))}},
	}

	for _, event := range events {
		if err := renderer.Handle(event); err != nil {
			t.Fatalf("handle event failed: %v", err)
		}
	}
	if err := renderer.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	lines := strings.FieldsFunc(stdout.String(), func(r rune) bool { return r == '\n' || r == '\r' })
	if len(lines) != len(events) {
		t.Fatalf("expected %d json events, got %d", len(events), len(lines))
	}
	var chunkSeen bool
	var treeSeen bool
	for index, line := range lines {
		var decoded stream.Event
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("failed to decode line %d: %v", index, err)
		}
		if decoded.Kind == stream.EventKindContentChunk {
			chunkSeen = true
			if decoded.Chunk == nil || decoded.Chunk.Data != "chunk" || !decoded.Chunk.IsFinal {
				t.Fatalf("chunk payload mismatch: %+v", decoded.Chunk)
			}
		}
		if decoded.Kind == stream.EventKindTree {
			treeSeen = true
			if decoded.Tree == nil || decoded.Tree.Path != root {
				t.Fatalf("tree payload mismatch: %+v", decoded.Tree)
			}
		}
	}
	if !chunkSeen {
		t.Fatalf("expected chunk event in json stream")
	}
	if !treeSeen {
		t.Fatalf("expected tree event in json stream")
	}

	if !strings.Contains(stderr.String(), "json warning") {
		t.Fatalf("expected warning on stderr")
	}
}
