package output

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/temirov/ctx/internal/services/stream"
)

type xmlStreamRenderer struct {
	stdout  io.Writer
	stderr  io.Writer
	command string
	encoder *xml.Encoder
	started bool
}

func NewXMLStreamRenderer(stdout, stderr io.Writer, command string) StreamRenderer {
	return &xmlStreamRenderer{stdout: stdout, stderr: stderr, command: command}
}

func (renderer *xmlStreamRenderer) Handle(event stream.Event) error {
	if event.Kind == stream.EventKindWarning && event.Message != nil && renderer.stderr != nil {
		fmt.Fprintln(renderer.stderr, event.Message.Message)
	}
	if event.Kind == stream.EventKindError && event.Err != nil && renderer.stderr != nil {
		fmt.Fprintln(renderer.stderr, event.Err.Message)
	}
	return renderer.writeEvent(event)
}

func (renderer *xmlStreamRenderer) Flush() error {
	if renderer.encoder != nil {
		if err := renderer.encoder.Flush(); err != nil {
			return err
		}
	}
	if renderer.started && renderer.stdout != nil {
		if _, err := io.WriteString(renderer.stdout, "</events>\n"); err != nil {
			return err
		}
	}
	return nil
}

func (renderer *xmlStreamRenderer) ensureEncoder() error {
	if renderer.stdout == nil || renderer.started {
		return nil
	}
	if _, err := io.WriteString(renderer.stdout, xml.Header); err != nil {
		return err
	}
	if _, err := io.WriteString(renderer.stdout, "<events>\n"); err != nil {
		return err
	}
	renderer.encoder = xml.NewEncoder(renderer.stdout)
	renderer.encoder.Indent("", "  ")
	renderer.started = true
	return nil
}

func (renderer *xmlStreamRenderer) writeEvent(event stream.Event) error {
	if renderer.stdout == nil {
		return nil
	}
	if err := renderer.ensureEncoder(); err != nil {
		return err
	}
	start := xml.StartElement{Name: xml.Name{Local: "event"}}
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "version"}, Value: strconv.Itoa(event.Version)})
	if event.Kind != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "kind"}, Value: string(event.Kind)})
	}
	if event.Command != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "command"}, Value: event.Command})
	}
	if event.Path != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "path"}, Value: event.Path})
	}
	if !event.EmittedAt.IsZero() {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "emittedAt"}, Value: event.EmittedAt.Format(time.RFC3339Nano)})
	}
	if err := renderer.encoder.EncodeToken(start); err != nil {
		return err
	}
	encodeElement := func(name string, value interface{}) error {
		if value == nil {
			return nil
		}
		return renderer.encoder.EncodeElement(value, xml.StartElement{Name: xml.Name{Local: name}})
	}
	if err := encodeElement("directory", event.Directory); err != nil {
		return err
	}
	if err := encodeElement("file", event.File); err != nil {
		return err
	}
	if err := encodeElement("chunk", event.Chunk); err != nil {
		return err
	}
	if err := encodeElement("summary", event.Summary); err != nil {
		return err
	}
	if err := encodeElement("message", event.Message); err != nil {
		return err
	}
	if err := encodeElement("error", event.Err); err != nil {
		return err
	}
	if event.Tree != nil {
		if err := renderer.encoder.EncodeElement(event.Tree, xml.StartElement{Name: xml.Name{Local: "tree"}}); err != nil {
			return err
		}
	}
	if err := renderer.encoder.EncodeToken(start.End()); err != nil {
		return err
	}
	if err := renderer.encoder.Flush(); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}
