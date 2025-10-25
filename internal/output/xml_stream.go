package output

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"

	"github.com/tyemirov/ctx/internal/services/stream"
	"github.com/tyemirov/ctx/internal/types"
	"github.com/tyemirov/ctx/internal/utils"
)

type xmlStreamRenderer struct {
	stdout         io.Writer
	stderr         io.Writer
	command        string
	totalRoots     int
	includeSummary bool
	expectContent  bool

	encoder       *xml.Encoder
	headerWritten bool
	wrapperOpened bool
	rootsClosed   int

	dirStack     []*xmlDirectoryFrame
	pendingFiles map[string]*xmlFileBuilder
}

type xmlDirectoryFrame struct {
	depth          int
	childCount     int
	childrenOpened bool
}

type xmlFileBuilder struct {
	path            string
	name            string
	depth           int
	nodeType        string
	sizeBytes       int64
	sizeDisplay     string
	lastModified    string
	mimeType        string
	tokens          int
	model           string
	documentation   []types.DocumentationEntry
	parentDepth     int
	awaitingContent bool
}

func NewXMLStreamRenderer(stdout, stderr io.Writer, command string, totalRoots int, includeSummary bool, expectContent bool) StreamRenderer {
	if totalRoots < 1 {
		totalRoots = 1
	}
	return &xmlStreamRenderer{
		stdout:         stdout,
		stderr:         stderr,
		command:        command,
		totalRoots:     totalRoots,
		includeSummary: includeSummary,
		expectContent:  expectContent,
		pendingFiles:   map[string]*xmlFileBuilder{},
	}
}

func (renderer *xmlStreamRenderer) Handle(event stream.Event) error {
	switch event.Kind {
	case stream.EventKindWarning:
		if event.Message != nil && renderer.stderr != nil {
			_, err := fmt.Fprintln(renderer.stderr, event.Message.Message)
			return err
		}
		return nil
	case stream.EventKindError:
		if event.Err != nil && renderer.stderr != nil {
			_, err := fmt.Fprintln(renderer.stderr, event.Err.Message)
			return err
		}
		return nil
	case stream.EventKindDirectory:
		if event.Directory == nil {
			return nil
		}
		if event.Directory.Phase == stream.DirectoryEnter {
			return renderer.handleDirectoryEnter(event.Directory)
		}
		return renderer.handleDirectoryLeave(event.Directory)
	case stream.EventKindFile:
		return renderer.handleFile(event.File)
	case stream.EventKindContentChunk:
		return renderer.handleChunk(event.Chunk)
	case stream.EventKindDone:
		renderer.dirStack = renderer.dirStack[:0]
		renderer.pendingFiles = map[string]*xmlFileBuilder{}
		return nil
	default:
		return nil
	}
}

func (renderer *xmlStreamRenderer) Flush() error {
	if renderer.stdout == nil {
		return nil
	}
	if err := renderer.ensureEncoder(); err != nil {
		return err
	}
	if err := renderer.closeWrapperIfNeeded(); err != nil {
		return err
	}
	if err := renderer.encoder.Flush(); err != nil {
		return err
	}
	_, err := renderer.stdout.Write([]byte("\n"))
	return err
}

func (renderer *xmlStreamRenderer) handleDirectoryEnter(directory *stream.DirectoryEvent) error {
	if renderer.stdout == nil || directory == nil {
		return nil
	}
	if err := renderer.ensureEncoder(); err != nil {
		return err
	}
	if err := renderer.ensureWrapperOpened(); err != nil {
		return err
	}
	if directory.Depth > 0 {
		parent := renderer.frameForDepth(directory.Depth - 1)
		if parent == nil {
			return fmt.Errorf("xml stream: missing parent frame for %s", directory.Path)
		}
		if err := renderer.ensureChildrenOpened(parent); err != nil {
			return err
		}
		parent.childCount++
	}

	start := xml.StartElement{Name: xml.Name{Local: "node"}}
	if err := renderer.encoder.EncodeToken(start); err != nil {
		return err
	}
	if err := renderer.writeSimpleElement("path", directory.Path); err != nil {
		return err
	}
	if directory.Name != "" {
		if err := renderer.writeSimpleElement("name", directory.Name); err != nil {
			return err
		}
	}
	if err := renderer.writeSimpleElement("type", types.NodeTypeDirectory); err != nil {
		return err
	}
	if directory.LastModified != "" {
		if err := renderer.writeSimpleElement("lastModified", directory.LastModified); err != nil {
			return err
		}
	}

	frame := &xmlDirectoryFrame{depth: directory.Depth}
	renderer.dirStack = append(renderer.dirStack, frame)
	return nil
}

func (renderer *xmlStreamRenderer) handleDirectoryLeave(directory *stream.DirectoryEvent) error {
	if renderer.stdout == nil || directory == nil {
		return nil
	}
	if err := renderer.ensureEncoder(); err != nil {
		return err
	}
	frame := renderer.popFrame()
	if frame == nil {
		return fmt.Errorf("xml stream: directory stack underflow for %s", directory.Path)
	}

	if frame.childrenOpened {
		if err := renderer.encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "children"}}); err != nil {
			return err
		}
	}

	if renderer.includeSummary && directory.Summary != nil {
		if directory.Summary.Model != "" {
			if err := renderer.writeSimpleElement("model", directory.Summary.Model); err != nil {
				return err
			}
		}
		if err := renderer.writeSimpleElement("totalFiles", strconv.Itoa(directory.Summary.Files)); err != nil {
			return err
		}
		if err := renderer.writeSimpleElement("totalSize", utils.FormatFileSize(directory.Summary.Bytes)); err != nil {
			return err
		}
		if directory.Summary.Tokens > 0 {
			if err := renderer.writeSimpleElement("totalTokens", strconv.Itoa(directory.Summary.Tokens)); err != nil {
				return err
			}
		}
	}

	if err := renderer.encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "node"}}); err != nil {
		return err
	}

	parentDepth := frame.depth - 1
	if parentDepth < 0 {
		renderer.rootsClosed++
	}
	return nil
}

func (renderer *xmlStreamRenderer) handleFile(file *stream.FileEvent) error {
	if renderer.stdout == nil || file == nil {
		return nil
	}
	if err := renderer.ensureEncoder(); err != nil {
		return err
	}
	if err := renderer.ensureWrapperOpened(); err != nil {
		return err
	}

	parentDepth := file.Depth - 1
	if parentDepth >= 0 {
		parent := renderer.frameForDepth(parentDepth)
		if parent == nil {
			return fmt.Errorf("xml stream: missing parent frame for file %s", file.Path)
		}
		if err := renderer.ensureChildrenOpened(parent); err != nil {
			return err
		}
		parent.childCount++
	} else {
		renderer.rootsClosed++
	}

	start := xml.StartElement{Name: xml.Name{Local: "node"}}
	if err := renderer.encoder.EncodeToken(start); err != nil {
		return err
	}
	if err := renderer.writeSimpleElement("path", file.Path); err != nil {
		return err
	}
	if err := renderer.writeSimpleElement("name", file.Name); err != nil {
		return err
	}
	if err := renderer.writeSimpleElement("type", file.Type); err != nil {
		return err
	}
	if display := utils.FormatFileSize(file.SizeBytes); display != "" {
		if err := renderer.writeSimpleElement("size", display); err != nil {
			return err
		}
	}
	if file.LastModified != "" {
		if err := renderer.writeSimpleElement("lastModified", file.LastModified); err != nil {
			return err
		}
	}
	if file.MimeType != "" {
		if err := renderer.writeSimpleElement("mimeType", file.MimeType); err != nil {
			return err
		}
	}
	if file.Tokens > 0 {
		if err := renderer.writeSimpleElement("tokens", strconv.Itoa(file.Tokens)); err != nil {
			return err
		}
	}
	if file.Model != "" {
		if err := renderer.writeSimpleElement("model", file.Model); err != nil {
			return err
		}
	}

	builder := &xmlFileBuilder{
		path:            file.Path,
		name:            file.Name,
		depth:           file.Depth,
		nodeType:        file.Type,
		sizeBytes:       file.SizeBytes,
		sizeDisplay:     utils.FormatFileSize(file.SizeBytes),
		lastModified:    file.LastModified,
		mimeType:        file.MimeType,
		tokens:          file.Tokens,
		model:           file.Model,
		documentation:   file.Documentation,
		parentDepth:     parentDepth,
		awaitingContent: renderer.expectContent,
	}

	if builder.awaitingContent {
		renderer.pendingFiles[file.Path] = builder
		return nil
	}

	return renderer.finalizeFile(builder, "")
}

func (renderer *xmlStreamRenderer) handleChunk(chunk *stream.ChunkEvent) error {
	if renderer.stdout == nil || chunk == nil {
		return nil
	}
	builder, ok := renderer.pendingFiles[chunk.Path]
	if !ok {
		return nil
	}
	delete(renderer.pendingFiles, chunk.Path)
	return renderer.finalizeFile(builder, chunk.Data)
}

func (renderer *xmlStreamRenderer) finalizeFile(builder *xmlFileBuilder, content string) error {
	if content != "" {
		if err := renderer.writeSimpleElement("content", content); err != nil {
			return err
		}
	}
	if len(builder.documentation) > 0 {
		if err := renderer.encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "documentation"}}); err != nil {
			return err
		}
		for _, entry := range builder.documentation {
			if err := renderer.encoder.EncodeElement(entry, xml.StartElement{Name: xml.Name{Local: "entry"}}); err != nil {
				return err
			}
		}
		if err := renderer.encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "documentation"}}); err != nil {
			return err
		}
	}
	if err := renderer.encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "node"}}); err != nil {
		return err
	}
	return nil
}

func (renderer *xmlStreamRenderer) ensureEncoder() error {
	if renderer.encoder != nil {
		return nil
	}
	renderer.encoder = xml.NewEncoder(renderer.stdout)
	renderer.encoder.Indent("", indentSpacer)
	return nil
}

func (renderer *xmlStreamRenderer) ensureWrapperOpened() error {
	if renderer.totalRoots <= 1 {
		if !renderer.headerWritten {
			if _, err := renderer.stdout.Write([]byte(xml.Header)); err != nil {
				return err
			}
			renderer.headerWritten = true
		}
		return nil
	}
	if !renderer.headerWritten {
		if _, err := renderer.stdout.Write([]byte(xml.Header)); err != nil {
			return err
		}
		renderer.headerWritten = true
	}
	if renderer.wrapperOpened {
		return nil
	}
	if err := renderer.encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "results"}}); err != nil {
		return err
	}
	renderer.wrapperOpened = true
	return nil
}

func (renderer *xmlStreamRenderer) closeWrapperIfNeeded() error {
	if renderer.totalRoots <= 1 {
		return nil
	}
	if renderer.wrapperOpened {
		if err := renderer.encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "results"}}); err != nil {
			return err
		}
		renderer.wrapperOpened = false
	}
	return nil
}

func (renderer *xmlStreamRenderer) ensureChildrenOpened(frame *xmlDirectoryFrame) error {
	if frame.childrenOpened {
		return nil
	}
	if err := renderer.encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "children"}}); err != nil {
		return err
	}
	frame.childrenOpened = true
	return nil
}

func (renderer *xmlStreamRenderer) writeSimpleElement(name string, value string) error {
	if value == "" {
		return nil
	}
	return renderer.encoder.EncodeElement(value, xml.StartElement{Name: xml.Name{Local: name}})
}

func (renderer *xmlStreamRenderer) frameForDepth(depth int) *xmlDirectoryFrame {
	for index := len(renderer.dirStack) - 1; index >= 0; index-- {
		frame := renderer.dirStack[index]
		if frame.depth == depth {
			return frame
		}
	}
	return nil
}

func (renderer *xmlStreamRenderer) popFrame() *xmlDirectoryFrame {
	if len(renderer.dirStack) == 0 {
		return nil
	}
	frame := renderer.dirStack[len(renderer.dirStack)-1]
	renderer.dirStack = renderer.dirStack[:len(renderer.dirStack)-1]
	return frame
}
