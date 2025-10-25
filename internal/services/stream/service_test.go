package stream_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tyemirov/ctx/internal/services/stream"
	"github.com/tyemirov/ctx/internal/types"
)

type stubCounter struct{}

func (stubCounter) Name() string { return "stub" }

func (stubCounter) CountString(input string) (int, error) { return len([]rune(input)), nil }

func TestStreamTreeEmitsEventsWithSummary(t *testing.T) {
	t.TempDir()
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	filePath := filepath.Join(nested, "example.txt")
	if err := os.WriteFile(filePath, []byte("tree"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	events := collectEvents(t, func(ch chan<- stream.Event) error {
		options := stream.TreeOptions{Root: root, TokenCounter: stubCounter{}, TokenModel: "stub-model"}
		return stream.StreamTree(context.Background(), options, ch)
	})

	if len(events) == 0 {
		t.Fatalf("expected events, got none")
	}
	if events[0].Kind != stream.EventKindStart {
		t.Fatalf("expected first event to be start, got %v", events[0].Kind)
	}

	var sawFile, sawSummary, sawTree bool
	for _, event := range events {
		switch event.Kind {
		case stream.EventKindFile:
			sawFile = true
			if event.File.Path != filePath {
				t.Fatalf("unexpected file path: %s", event.File.Path)
			}
			if event.File.Model != "stub-model" {
				t.Fatalf("expected model propagated to file event")
			}
		case stream.EventKindSummary:
			sawSummary = true
			if event.Summary.Files != 1 {
				t.Fatalf("expected 1 file in summary, got %d", event.Summary.Files)
			}
			if event.Summary.Bytes != int64(len("tree")) {
				t.Fatalf("unexpected bytes in summary: %d", event.Summary.Bytes)
			}
		case stream.EventKindTree:
			sawTree = true
			if event.Tree == nil {
				t.Fatalf("tree event missing payload")
			}
			if event.Tree.Path != root {
				t.Fatalf("unexpected tree root: %s", event.Tree.Path)
			}
		}
	}

	if !sawFile {
		t.Fatalf("file event not emitted")
	}
	if !sawSummary {
		t.Fatalf("summary event not emitted")
	}
	if !sawTree {
		t.Fatalf("tree event not emitted")
	}

	lastEvents := events[len(events)-2:]
	if lastEvents[0].Kind != stream.EventKindSummary || lastEvents[1].Kind != stream.EventKindDone {
		t.Fatalf("expected summary followed by done at end, got %v %v", lastEvents[0].Kind, lastEvents[1].Kind)
	}
}

func TestStreamContentEmitsChunkAndTree(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "example.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	events := collectEvents(t, func(ch chan<- stream.Event) error {
		options := stream.ContentOptions{Root: root, TokenCounter: stubCounter{}, TokenModel: "stub"}
		return stream.StreamContent(context.Background(), options, ch)
	})

	var sawChunk, sawSummary, sawTree bool
	for _, event := range events {
		switch event.Kind {
		case stream.EventKindContentChunk:
			sawChunk = true
			if event.Chunk.Path != filePath {
				t.Fatalf("chunk for unexpected path: %s", event.Chunk.Path)
			}
			if event.Chunk.Data != "content" {
				t.Fatalf("unexpected chunk data: %s", event.Chunk.Data)
			}
		case stream.EventKindSummary:
			sawSummary = true
			if event.Summary.Files != 1 {
				t.Fatalf("expected summary to include one file, got %d", event.Summary.Files)
			}
		case stream.EventKindTree:
			sawTree = true
			if event.Tree == nil || event.Tree.Type != types.NodeTypeDirectory {
				t.Fatalf("expected directory tree node")
			}
		}
	}

	if !sawChunk {
		t.Fatalf("expected chunk event")
	}
	if !sawSummary {
		t.Fatalf("expected summary event")
	}
	if !sawTree {
		t.Fatalf("expected tree event")
	}
}

func collectEvents(t *testing.T, producer func(chan<- stream.Event) error) []stream.Event {
	t.Helper()
	events := make(chan stream.Event, 32)
	errCh := make(chan error, 1)
	go func() {
		errCh <- producer(events)
		close(events)
	}()

	var out []stream.Event
	for event := range events {
		out = append(out, event)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("producer returned error: %v", err)
	}
	return out
}
