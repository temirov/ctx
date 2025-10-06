package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

type treeStubCounter struct{}

func (treeStubCounter) Name() string { return "stub" }

func (treeStubCounter) CountString(input string) (int, error) {
	return len([]rune(input)), nil
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writePipe

	var buffer bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buffer, readPipe)
		close(done)
	}()

	fn()

	writePipe.Close()
	os.Stdout = original
	<-done
	return buffer.String()
}

func TestRunTreeRawStreamingOutputsSummaryAfterFiles(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	rootFile := filepath.Join(tempDir, "root.txt")
	nestedFile := filepath.Join(nestedDir, "inner.txt")

	if err := os.WriteFile(rootFile, []byte("token"), 0o600); err != nil {
		t.Fatalf("write root file: %v", err)
	}
	if err := os.WriteFile(nestedFile, []byte("abc"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	validated := []types.ValidatedPath{{AbsolutePath: tempDir, IsDir: true}}

	outputText := captureStdout(t, func() {
		if err := runTreeRawStreaming(validated, nil, true, true, false, true, treeStubCounter{}, "stub-model"); err != nil {
			t.Fatalf("runTreeRawStreaming error: %v", err)
		}
	})

	if !strings.Contains(outputText, tempDir) {
		t.Fatalf("expected directory path in output")
	}
	if !strings.Contains(outputText, nestedDir) {
		t.Fatalf("expected nested directory path in output")
	}
	if !strings.Contains(outputText, rootFile) {
		t.Fatalf("expected root file in output")
	}
	if !strings.Contains(outputText, nestedFile) {
		t.Fatalf("expected nested file in output")
	}

	nestedFileIndex := strings.Index(outputText, nestedFile)
	nestedSummaryKey := fmt.Sprintf("  Summary: 1 file, %s", utils.FormatFileSize(3))
	nestedSummaryIndex := strings.Index(outputText, nestedSummaryKey)
	if nestedSummaryIndex == -1 {
		t.Fatalf("expected nested summary line in output")
	}
	if nestedSummaryIndex < nestedFileIndex {
		t.Fatalf("nested summary appeared before nested file")
	}

	globalSummary := "Summary: 2 files"
	globalIndex := strings.LastIndex(outputText, globalSummary)
	if globalIndex == -1 {
		t.Fatalf("expected global summary in output")
	}
}
