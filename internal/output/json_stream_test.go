package output_test

import (
	"bytes"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

func TestJSONStreamRendererOutputsDirectoryTree(testingInstance *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	renderer := output.NewJSONStreamRenderer(&stdout, &stderr, types.CommandContent, 1, true, true)

	rootPath := "/tmp/root"
	filePath := rootPath + "/file.txt"

	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: rootPath},
		{
			Kind: stream.EventKindDirectory,
			Directory: &stream.DirectoryEvent{
				Phase:        stream.DirectoryEnter,
				Path:         rootPath,
				Name:         "root",
				Depth:        0,
				LastModified: "2025-01-01 10:00",
			},
		},
		{
			Kind: stream.EventKindFile,
			File: &stream.FileEvent{
				Path:         filePath,
				Name:         "file.txt",
				Depth:        1,
				SizeBytes:    4,
				LastModified: "2025-01-01",
				MimeType:     "text/plain; charset=utf-8",
				Tokens:       10,
				Model:        "claude",
				Type:         types.NodeTypeFile,
			},
		},
		{
			Kind: stream.EventKindContentChunk,
			Chunk: &stream.ChunkEvent{
				Path:     filePath,
				Index:    0,
				Data:     "text",
				Encoding: "utf-8",
				IsFinal:  true,
			},
		},
		{
			Kind: stream.EventKindDirectory,
			Directory: &stream.DirectoryEvent{
				Phase:        stream.DirectoryLeave,
				Path:         rootPath,
				Name:         "root",
				Depth:        0,
				LastModified: "2025-01-01 10:00",
				Summary: &stream.SummaryEvent{
					Files:  1,
					Bytes:  4,
					Tokens: 10,
					Model:  "claude",
				},
			},
		},
		{Kind: stream.EventKindSummary, Summary: &stream.SummaryEvent{Files: 1, Bytes: 4, Tokens: 10, Model: "claude"}},
		{Kind: stream.EventKindDone},
	}

	for _, event := range events {
		if handleError := renderer.Handle(event); handleError != nil {
			testingInstance.Fatalf("handle failed: %v", handleError)
		}
	}

	if flushError := renderer.Flush(); flushError != nil {
		testingInstance.Fatalf("flush failed: %v", flushError)
	}

	if stderr.Len() != 0 {
		testingInstance.Fatalf("unexpected stderr output: %s", stderr.String())
	}

	expected := "{\n" +
		"  \"path\": \"/tmp/root\",\n" +
		"  \"name\": \"root\",\n" +
		"  \"type\": \"directory\",\n" +
		"  \"lastModified\": \"2025-01-01 10:00\",\n" +
		"  \"children\": [\n" +
		"    {\n" +
		"      \"path\": \"/tmp/root/file.txt\",\n" +
		"      \"name\": \"file.txt\",\n" +
		"      \"type\": \"file\",\n" +
		"      \"size\": \"4b\",\n" +
		"      \"lastModified\": \"2025-01-01\",\n" +
		"      \"mimeType\": \"text/plain; charset=utf-8\",\n" +
		"      \"tokens\": 10,\n" +
		"      \"model\": \"claude\",\n" +
		"      \"content\": \"text\"\n" +
		"    }\n" +
		"  ],\n" +
		"  \"model\": \"claude\",\n" +
		"  \"totalFiles\": 1,\n" +
		"  \"totalSize\": \"4b\",\n" +
		"  \"totalTokens\": 10\n" +
		"}\n"

	if stdout.String() != expected {
		testingInstance.Fatalf("unexpected JSON output\nexpected: %s\nactual: %s", expected, stdout.String())
	}
}

func TestJSONStreamRendererOutputsMultipleRoots(testingInstance *testing.T) {
	var stdout bytes.Buffer
	renderer := output.NewJSONStreamRenderer(&stdout, nil, types.CommandTree, 2, true, false)

	firstPath := "/tmp/first"
	secondPath := "/tmp/second"

	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: firstPath},
		{
			Kind: stream.EventKindDirectory,
			Directory: &stream.DirectoryEvent{
				Phase:        stream.DirectoryEnter,
				Path:         firstPath,
				Name:         "first",
				Depth:        0,
				LastModified: "2025-01-01",
			},
		},
		{
			Kind: stream.EventKindDirectory,
			Directory: &stream.DirectoryEvent{
				Phase:        stream.DirectoryLeave,
				Path:         firstPath,
				Name:         "first",
				Depth:        0,
				LastModified: "2025-01-01",
				Summary:      &stream.SummaryEvent{Files: 0, Bytes: 0, Tokens: 0},
			},
		},
		{Kind: stream.EventKindDone},
		{Kind: stream.EventKindStart, Path: secondPath},
		{
			Kind: stream.EventKindDirectory,
			Directory: &stream.DirectoryEvent{
				Phase:        stream.DirectoryEnter,
				Path:         secondPath,
				Name:         "second",
				Depth:        0,
				LastModified: "2025-01-02",
			},
		},
		{
			Kind: stream.EventKindDirectory,
			Directory: &stream.DirectoryEvent{
				Phase:        stream.DirectoryLeave,
				Path:         secondPath,
				Name:         "second",
				Depth:        0,
				LastModified: "2025-01-02",
				Summary:      &stream.SummaryEvent{Files: 0, Bytes: 0, Tokens: 0},
			},
		},
		{Kind: stream.EventKindDone},
	}

	for _, event := range events {
		if handleError := renderer.Handle(event); handleError != nil {
			testingInstance.Fatalf("handle failed: %v", handleError)
		}
	}

	if flushError := renderer.Flush(); flushError != nil {
		testingInstance.Fatalf("flush failed: %v", flushError)
	}

	expected := "[\n" +
		"  {\n" +
		"    \"path\": \"/tmp/first\",\n" +
		"    \"name\": \"first\",\n" +
		"    \"type\": \"directory\",\n" +
		"    \"lastModified\": \"2025-01-01\",\n" +
		"    \"children\": [],\n" +
		"    \"totalFiles\": 0,\n" +
		"    \"totalSize\": \"0b\"\n" +
		"  },\n" +
		"  {\n" +
		"    \"path\": \"/tmp/second\",\n" +
		"    \"name\": \"second\",\n" +
		"    \"type\": \"directory\",\n" +
		"    \"lastModified\": \"2025-01-02\",\n" +
		"    \"children\": [],\n" +
		"    \"totalFiles\": 0,\n" +
		"    \"totalSize\": \"0b\"\n" +
		"  }\n" +
		"]\n"

	if stdout.String() != expected {
		testingInstance.Fatalf("unexpected JSON array output\nexpected: %s\nactual: %s", expected, stdout.String())
	}
}
