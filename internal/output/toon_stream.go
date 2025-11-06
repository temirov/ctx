package output

import (
	"fmt"
	"io"

	"github.com/tyemirov/ctx/internal/services/stream"
	"github.com/tyemirov/ctx/internal/types"
)

type toonSummary struct {
	files  int
	bytes  int64
	tokens int
	model  string
}

func (summary *toonSummary) add(data *stream.SummaryEvent) {
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

type toonStreamRenderer struct {
	stdout         io.Writer
	stderr         io.Writer
	command        string
	includeSummary bool
	trees          []*types.TreeOutputNode
	summary        toonSummary
}

func NewToonStreamRenderer(stdout, stderr io.Writer, command string, includeSummary bool) StreamRenderer {
	return &toonStreamRenderer{
		stdout:         stdout,
		stderr:         stderr,
		command:        command,
		includeSummary: includeSummary,
	}
}

func (renderer *toonStreamRenderer) Handle(event stream.Event) error {
	switch event.Kind {
	case stream.EventKindWarning:
		if event.Message != nil && renderer.stderr != nil {
			_, err := fmt.Fprintln(renderer.stderr, event.Message.Message)
			return err
		}
	case stream.EventKindError:
		if event.Err != nil && renderer.stderr != nil {
			_, err := fmt.Fprintln(renderer.stderr, event.Err.Message)
			return err
		}
	case stream.EventKindSummary:
		renderer.summary.add(event.Summary)
	case stream.EventKindTree:
		if event.Tree != nil {
			renderer.trees = append(renderer.trees, cloneTreeNode(event.Tree))
		}
	}
	return nil
}

func (renderer *toonStreamRenderer) Flush() error {
	if renderer.stdout == nil {
		return nil
	}
	builder := toonBuilder{}
	builder.writeTreeNodes("roots", renderer.trees)
	if renderer.includeSummary {
		builder.writeSummary(renderer.summary)
	}
	_, err := fmt.Fprint(renderer.stdout, builder.String())
	return err
}
