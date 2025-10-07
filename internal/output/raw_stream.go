package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

type rawSummary struct {
	files  int
	bytes  int64
	tokens int
	model  string
}

func (summary *rawSummary) add(data *stream.SummaryEvent) {
	if data == nil {
		return
	}
	summary.files += data.Files
	summary.bytes += data.Bytes
	summary.tokens += data.Tokens
	if summary.model == "" && data.Model != "" && data.Tokens > 0 {
		summary.model = data.Model
	}
}

type pendingContent struct {
	fileType string
	path     string
}

type rawStreamRenderer struct {
	stdout         io.Writer
	stderr         io.Writer
	command        string
	includeSummary bool
	summary        rawSummary
	trees          []*types.TreeOutputNode
	pending        map[string]pendingContent
}

func NewRawStreamRenderer(stdout, stderr io.Writer, command string, includeSummary bool) StreamRenderer {
	return &rawStreamRenderer{
		stdout:         stdout,
		stderr:         stderr,
		command:        command,
		includeSummary: includeSummary,
		pending:        map[string]pendingContent{},
	}
}

func (renderer *rawStreamRenderer) Handle(event stream.Event) error {
	switch event.Kind {
	case stream.EventKindWarning:
		if event.Message != nil && renderer.stderr != nil {
			fmt.Fprintln(renderer.stderr, event.Message.Message)
		}
	case stream.EventKindError:
		if event.Err != nil && renderer.stderr != nil {
			fmt.Fprintln(renderer.stderr, event.Err.Message)
		}
	case stream.EventKindDirectory:
		renderer.handleDirectory(event.Directory)
	case stream.EventKindFile:
		renderer.handleFile(event.File)
	case stream.EventKindContentChunk:
		renderer.handleChunk(event.Chunk)
	case stream.EventKindSummary:
		renderer.summary.add(event.Summary)
	case stream.EventKindTree:
		if renderer.command == types.CommandContent && event.Tree != nil {
			renderer.trees = append(renderer.trees, event.Tree)
		}
	}
	return nil
}

func (renderer *rawStreamRenderer) Flush() error {
	if renderer.includeSummary && renderer.stdout != nil {
		outputSummary := &types.OutputSummary{
			TotalFiles:  renderer.summary.files,
			TotalSize:   utils.FormatFileSize(renderer.summary.bytes),
			TotalTokens: renderer.summary.tokens,
			Model:       renderer.summary.model,
		}
		fmt.Fprintln(renderer.stdout, FormatSummaryLine(outputSummary))
		fmt.Fprintln(renderer.stdout)
	}

	if renderer.command == types.CommandContent && renderer.stdout != nil {
		for _, node := range renderer.trees {
			fmt.Fprintf(renderer.stdout, "\n--- Directory Tree: %s ---\n", node.Path)
			WriteTreeRaw(renderer.stdout, node, renderer.includeSummary)
		}
	}

	return nil
}

func (renderer *rawStreamRenderer) handleDirectory(directory *stream.DirectoryEvent) {
	if renderer.stdout == nil || renderer.command != types.CommandTree || directory == nil {
		return
	}
	prefix := strings.Repeat("  ", directory.Depth)
	if directory.Phase == stream.DirectoryEnter {
		fmt.Fprintf(renderer.stdout, "%s%s\n", prefix, directory.Path)
		return
	}
	if !renderer.includeSummary || directory.Summary == nil {
		return
	}
	label := "files"
	count := directory.Summary.Files
	if count == 1 {
		label = "file"
	}
	size := utils.FormatFileSize(directory.Summary.Bytes)
	extra := ""
	if directory.Summary.Tokens > 0 {
		extra = fmt.Sprintf(", %d tokens", directory.Summary.Tokens)
	}
	fmt.Fprintf(renderer.stdout, "%s  Summary: %d %s, %s%s\n", prefix, count, label, size, extra)
}

func (renderer *rawStreamRenderer) handleFile(file *stream.FileEvent) {
	if renderer.stdout == nil || file == nil {
		return
	}

	switch renderer.command {
	case types.CommandTree:
		renderer.printTreeFile(file)
	case types.CommandContent:
		renderer.printContentHeader(file)
	}
}

func (renderer *rawStreamRenderer) handleChunk(chunk *stream.ChunkEvent) {
	if renderer.stdout == nil || renderer.command != types.CommandContent || chunk == nil {
		return
	}
	pending, exists := renderer.pending[chunk.Path]
	if !exists {
		return
	}
	if pending.fileType == types.NodeTypeBinary {
		if chunk.Encoding == "" {
			fmt.Fprintln(renderer.stdout, binaryContentOmitted)
		} else {
			fmt.Fprintln(renderer.stdout, chunk.Data)
		}
	} else {
		fmt.Fprintln(renderer.stdout, chunk.Data)
	}
	if chunk.IsFinal {
		fmt.Fprintf(renderer.stdout, "End of file: %s\n", pending.path)
		fmt.Fprintln(renderer.stdout, separatorLine)
		delete(renderer.pending, chunk.Path)
	}
}

func (renderer *rawStreamRenderer) printTreeFile(file *stream.FileEvent) {
	prefix := strings.Repeat("  ", file.Depth)
	if file.IsBinary {
		fmt.Fprintf(renderer.stdout, binaryTreeFormat, prefix, file.Path, mimeTypeLabel, file.MimeType)
		return
	}
	if file.Tokens > 0 {
		fmt.Fprintf(renderer.stdout, "%s[File] %s (%d tokens)\n", prefix, file.Path, file.Tokens)
		return
	}
	fmt.Fprintf(renderer.stdout, "%s[File] %s\n", prefix, file.Path)
}

func (renderer *rawStreamRenderer) printContentHeader(file *stream.FileEvent) {
	fmt.Fprintf(renderer.stdout, "File: %s\n", file.Path)
	if file.Type == types.NodeTypeBinary {
		fmt.Fprintf(renderer.stdout, "%s%s\n", mimeTypeLabel, file.MimeType)
	}
	renderer.pending[file.Path] = pendingContent{fileType: file.Type, path: file.Path}
}
