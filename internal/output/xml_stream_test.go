package output_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

func TestXMLStreamRendererMatchesBatchOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewXMLStreamRenderer(&stdout, &stderr, types.CommandContent)

	root := "/tmp"
	filePath := root + "/sample.bin"
	events := []stream.Event{
		{Kind: stream.EventKindStart, Path: root},
		{Kind: stream.EventKindFile, File: &stream.FileEvent{
			Path:         filePath,
			Type:         types.NodeTypeBinary,
			SizeBytes:    5,
			LastModified: "now",
			MimeType:     "application/octet-stream",
		}},
		{Kind: stream.EventKindContentChunk, Chunk: &stream.ChunkEvent{Path: filePath, Data: "YWJjZA==", Encoding: "base64", IsFinal: true}},
		{Kind: stream.EventKindTree, Tree: &types.TreeOutputNode{Path: root, Type: types.NodeTypeDirectory}},
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

	decoder := xml.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var eventCount int
	var chunkSeen bool
	var treeSeen bool
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("xml decode error: %v", err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local != "event" {
				continue
			}
			var decoded struct {
				Kind  string `xml:"kind,attr"`
				Chunk *struct {
					Encoding string `xml:"encoding,attr"`
					IsFinal  bool   `xml:"isFinal,attr"`
					Data     string `xml:"data"`
				} `xml:"chunk"`
				Tree *struct {
					Path string `xml:"path"`
				} `xml:"tree"`
			}
			if err := decoder.DecodeElement(&decoded, &element); err != nil {
				t.Fatalf("decode element failed: %v", err)
			}
			eventCount++
			if decoded.Kind == string(stream.EventKindContentChunk) {
				chunkSeen = true
				if decoded.Chunk == nil || decoded.Chunk.Data != "YWJjZA==" || decoded.Chunk.Encoding != "base64" || !decoded.Chunk.IsFinal {
					t.Fatalf("unexpected chunk payload: %+v", decoded.Chunk)
				}
			}
			if decoded.Kind == string(stream.EventKindTree) {
				treeSeen = true
				if decoded.Tree == nil || decoded.Tree.Path != root {
					t.Fatalf("unexpected tree payload: %+v", decoded.Tree)
				}
			}
		}
	}
	if eventCount == 0 {
		t.Fatalf("expected xml events, got none: %s", stdout.String())
	}
	if !chunkSeen {
		t.Fatalf("expected chunk event in xml stream")
	}
	if !treeSeen {
		t.Fatalf("expected tree event in xml stream")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr")
	}
}
