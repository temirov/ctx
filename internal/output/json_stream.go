package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/temirov/ctx/internal/services/stream"
)

type jsonStreamRenderer struct {
	stdout  io.Writer
	stderr  io.Writer
	command string
}

func NewJSONStreamRenderer(stdout, stderr io.Writer, command string) StreamRenderer {
	return &jsonStreamRenderer{stdout: stdout, stderr: stderr, command: command}
}

func (renderer *jsonStreamRenderer) Handle(event stream.Event) error {
	if event.Kind == stream.EventKindWarning && event.Message != nil && renderer.stderr != nil {
		fmt.Fprintln(renderer.stderr, event.Message.Message)
	}
	if event.Kind == stream.EventKindError && event.Err != nil && renderer.stderr != nil {
		fmt.Fprintln(renderer.stderr, event.Err.Message)
	}
	return renderer.writeEvent(event)
}

func (renderer *jsonStreamRenderer) Flush() error {
	return nil
}

func (renderer *jsonStreamRenderer) writeEvent(event stream.Event) error {
	if renderer.stdout == nil {
		return nil
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := renderer.stdout.Write(encoded); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}
