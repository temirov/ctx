package commands_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/tyemirov/ctx/internal/commands"
	"github.com/tyemirov/ctx/internal/types"
)

type streamStubCounter struct{}

func (streamStubCounter) Name() string { return "stub" }

func (streamStubCounter) CountString(input string) (int, error) {
	return len([]rune(input)), nil
}

func TestStreamContentMatchesGetContentData(t *testing.T) {
	tempDir := t.TempDir()

	textPath := filepath.Join(tempDir, "a.txt")
	otherPath := filepath.Join(tempDir, "b.txt")

	if err := os.WriteFile(textPath, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write text file: %v", err)
	}
	if err := os.WriteFile(otherPath, []byte("stream"), 0o600); err != nil {
		t.Fatalf("write other file: %v", err)
	}

	var streamed []types.FileOutput
	err := commands.StreamContent(tempDir, nil, nil, streamStubCounter{}, "stub-model", func(output types.FileOutput) error {
		streamed = append(streamed, output)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamContent error: %v", err)
	}

	collected, err := commands.GetContentData(tempDir, nil, nil, streamStubCounter{}, "stub-model")
	if err != nil {
		t.Fatalf("GetContentData error: %v", err)
	}

	if len(streamed) != len(collected) {
		t.Fatalf("expected %d streamed items, got %d", len(collected), len(streamed))
	}

	sort.Slice(streamed, func(i, j int) bool { return streamed[i].Path < streamed[j].Path })
	sort.Slice(collected, func(i, j int) bool { return collected[i].Path < collected[j].Path })

	for i := range streamed {
		if streamed[i].Path != collected[i].Path {
			t.Fatalf("path mismatch at %d: %s vs %s", i, streamed[i].Path, collected[i].Path)
		}
		if streamed[i].Tokens != collected[i].Tokens {
			t.Fatalf("token mismatch at %d: %d vs %d", i, streamed[i].Tokens, collected[i].Tokens)
		}
		if streamed[i].SizeBytes != collected[i].SizeBytes {
			t.Fatalf("size mismatch at %d: %d vs %d", i, streamed[i].SizeBytes, collected[i].SizeBytes)
		}
		if streamed[i].Model != collected[i].Model {
			t.Fatalf("model mismatch at %d: %q vs %q", i, streamed[i].Model, collected[i].Model)
		}
	}
}
