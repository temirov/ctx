package output_test

import (
	"bytes"
	"testing"

	"github.com/tyemirov/ctx/internal/output"
	"github.com/tyemirov/ctx/internal/services/stream"
	"github.com/tyemirov/ctx/internal/types"
)

func TestXMLStreamRendererOutputsDirectoryTree(testingInstance *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	renderer := output.NewXMLStreamRenderer(&stdout, &stderr, types.CommandContent, 1, true, true)

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

	expected := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<node>\n" +
		"  <path>/tmp/root</path>\n" +
		"  <name>root</name>\n" +
		"  <type>directory</type>\n" +
		"  <lastModified>2025-01-01 10:00</lastModified>\n" +
		"  <children>\n" +
		"    <node>\n" +
		"      <path>/tmp/root/file.txt</path>\n" +
		"      <name>file.txt</name>\n" +
		"      <type>file</type>\n" +
		"      <size>4b</size>\n" +
		"      <lastModified>2025-01-01</lastModified>\n" +
		"      <mimeType>text/plain; charset=utf-8</mimeType>\n" +
		"      <tokens>10</tokens>\n" +
		"      <model>claude</model>\n" +
		"      <content>text</content>\n" +
		"    </node>\n" +
		"  </children>\n" +
		"  <model>claude</model>\n" +
		"  <totalFiles>1</totalFiles>\n" +
		"  <totalSize>4b</totalSize>\n" +
		"  <totalTokens>10</totalTokens>\n" +
		"</node>\n"

	if stdout.String() != expected {
		testingInstance.Fatalf("unexpected XML output\nexpected: %s\nactual: %s", expected, stdout.String())
	}
}

func TestXMLStreamRendererOutputsMultipleRoots(testingInstance *testing.T) {
	var stdout bytes.Buffer

	renderer := output.NewXMLStreamRenderer(&stdout, nil, types.CommandTree, 2, true, false)

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
				Summary:      &stream.SummaryEvent{},
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
				Summary:      &stream.SummaryEvent{},
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

	expected := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<results>\n" +
		"  <node>\n" +
		"    <path>/tmp/first</path>\n" +
		"    <name>first</name>\n" +
		"    <type>directory</type>\n" +
		"    <lastModified>2025-01-01</lastModified>\n" +
		"    <totalFiles>0</totalFiles>\n" +
		"    <totalSize>0b</totalSize>\n" +
		"  </node>\n" +
		"  <node>\n" +
		"    <path>/tmp/second</path>\n" +
		"    <name>second</name>\n" +
		"    <type>directory</type>\n" +
		"    <lastModified>2025-01-02</lastModified>\n" +
		"    <totalFiles>0</totalFiles>\n" +
		"    <totalSize>0b</totalSize>\n" +
		"  </node>\n" +
		"</results>\n"

	if stdout.String() != expected {
		testingInstance.Fatalf("unexpected XML output\nexpected: %s\nactual: %s", expected, stdout.String())
	}
}
