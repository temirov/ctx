package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

type jsonStreamRenderer struct {
	stdout         io.Writer
	stderr         io.Writer
	command        string
	totalRoots     int
	includeSummary bool
	expectContent  bool
	rootIndent     int
	arrayOpened    bool
	arrayClosed    bool
	rootsClosed    int
	dirStack       []*jsonDirectoryFrame
	pendingFiles   map[string]*jsonFileBuilder
}

type jsonDirectoryFrame struct {
	depth          int
	childCount     int
	childrenOpened bool
}

type jsonFileBuilder struct {
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

type jsonFilePayload struct {
	Path          string                     `json:"path"`
	Name          string                     `json:"name"`
	Type          string                     `json:"type"`
	Size          string                     `json:"size,omitempty"`
	LastModified  string                     `json:"lastModified,omitempty"`
	MimeType      string                     `json:"mimeType,omitempty"`
	Tokens        int                        `json:"tokens,omitempty"`
	Model         string                     `json:"model,omitempty"`
	Content       string                     `json:"content,omitempty"`
	Documentation []types.DocumentationEntry `json:"documentation,omitempty"`
}

type jsonField struct {
	key   string
	value string
}

func NewJSONStreamRenderer(stdout, stderr io.Writer, command string, totalRoots int, includeSummary bool, expectContent bool) StreamRenderer {
	if totalRoots < 1 {
		totalRoots = 1
	}
	renderer := &jsonStreamRenderer{
		stdout:         stdout,
		stderr:         stderr,
		command:        command,
		totalRoots:     totalRoots,
		includeSummary: includeSummary,
		expectContent:  expectContent,
		pendingFiles:   map[string]*jsonFileBuilder{},
	}
	if totalRoots > 1 {
		renderer.rootIndent = 1
	}
	return renderer
}

func (renderer *jsonStreamRenderer) Handle(event stream.Event) error {
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
		renderer.pendingFiles = map[string]*jsonFileBuilder{}
		renderer.dirStack = renderer.dirStack[:0]
		return nil
	default:
		return nil
	}
}

func (renderer *jsonStreamRenderer) Flush() error {
	if renderer.stdout == nil {
		return nil
	}
	if err := renderer.closeArrayIfNeeded(); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func (renderer *jsonStreamRenderer) handleDirectoryEnter(directory *stream.DirectoryEvent) error {
	if renderer.stdout == nil || directory == nil {
		return nil
	}

	if directory.Depth == 0 {
		if err := renderer.beginRootObject(); err != nil {
			return err
		}
	} else {
		parent := renderer.frameForDepth(directory.Depth - 1)
		if parent == nil {
			return fmt.Errorf("json stream: missing parent frame for directory %s", directory.Path)
		}
		if err := renderer.beginChild(parent); err != nil {
			return err
		}
	}

	if err := renderer.startObject(renderer.baseIndent(directory.Depth)); err != nil {
		return err
	}

	fields := renderer.directoryFields(directory)
	fieldHasMore := true
	if err := renderer.writeFields(renderer.baseIndent(directory.Depth)+1, fields, fieldHasMore); err != nil {
		return err
	}
	if err := renderer.writeIndent(renderer.baseIndent(directory.Depth) + 1); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte("\"children\": [")); err != nil {
		return err
	}

	frame := &jsonDirectoryFrame{depth: directory.Depth}
	renderer.dirStack = append(renderer.dirStack, frame)
	return nil
}

func (renderer *jsonStreamRenderer) handleDirectoryLeave(directory *stream.DirectoryEvent) error {
	if renderer.stdout == nil || directory == nil {
		return nil
	}
	frame := renderer.popFrame()
	if frame == nil {
		return fmt.Errorf("json stream: directory stack underflow for %s", directory.Path)
	}
	childIndent := renderer.baseIndent(frame.depth) + 1
	if frame.childCount > 0 {
		if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
			return err
		}
		if err := renderer.writeIndent(childIndent); err != nil {
			return err
		}
	}
	if _, err := renderer.stdout.Write([]byte("]")); err != nil {
		return err
	}

	newlineNeeded := true
	if renderer.includeSummary && directory.Summary != nil {
		summaryFields := renderer.summaryFields(directory.Summary)
		if len(summaryFields) > 0 {
			if _, err := renderer.stdout.Write([]byte(",\n")); err != nil {
				return err
			}
			if err := renderer.writeFields(childIndent, summaryFields, false); err != nil {
				return err
			}
			newlineNeeded = false
		}
	}

	if newlineNeeded {
		if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if err := renderer.writeIndent(renderer.baseIndent(frame.depth)); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte("}")); err != nil {
		return err
	}

	parentDepth := frame.depth - 1
	totalFiles := 0
	totalBytes := int64(0)
	totalTokens := 0
	model := ""
	if directory.Summary != nil {
		totalFiles = directory.Summary.Files
		totalBytes = directory.Summary.Bytes
		totalTokens = directory.Summary.Tokens
		model = directory.Summary.Model
	}
	renderer.onChildClosed(parentDepth, totalFiles, totalBytes, totalTokens, model)
	return nil
}

func (renderer *jsonStreamRenderer) handleFile(file *stream.FileEvent) error {
	if renderer.stdout == nil || file == nil {
		return nil
	}

	parentDepth := file.Depth - 1
	builder := &jsonFileBuilder{
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

	return renderer.emitFile(builder, "")
}

func (renderer *jsonStreamRenderer) handleChunk(chunk *stream.ChunkEvent) error {
	if renderer.stdout == nil || chunk == nil {
		return nil
	}
	builder, ok := renderer.pendingFiles[chunk.Path]
	if !ok {
		return nil
	}
	delete(renderer.pendingFiles, chunk.Path)
	return renderer.emitFile(builder, chunk.Data)
}

func (renderer *jsonStreamRenderer) emitFile(builder *jsonFileBuilder, content string) error {
	parentDepth := builder.parentDepth
	if parentDepth >= 0 {
		parent := renderer.frameForDepth(parentDepth)
		if parent == nil {
			return fmt.Errorf("json stream: missing parent frame for file %s", builder.path)
		}
		if err := renderer.beginChild(parent); err != nil {
			return err
		}
	} else {
		if err := renderer.beginRootObject(); err != nil {
			return err
		}
	}

	objectIndent := renderer.baseIndent(builder.depth)
	if parentDepth >= 0 {
		objectIndent++
	}

	payload := renderer.buildFilePayload(builder, content)
	encoded, err := json.MarshalIndent(payload, indentPrefix, indentSpacer)
	if err != nil {
		return err
	}
	if err := renderer.writeIndentedBlock(objectIndent, encoded); err != nil {
		return err
	}
	renderer.onChildClosed(parentDepth, 1, builder.sizeBytes, builder.tokens, builder.model)
	return nil
}

func (renderer *jsonStreamRenderer) beginRootObject() error {
	if renderer.totalRoots > 1 {
		if !renderer.arrayOpened {
			if _, err := renderer.stdout.Write([]byte("[\n")); err != nil {
				return err
			}
			renderer.arrayOpened = true
		} else {
			if _, err := renderer.stdout.Write([]byte(",\n")); err != nil {
				return err
			}
		}
	} else if renderer.rootsClosed > 0 {
		if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (renderer *jsonStreamRenderer) beginChild(frame *jsonDirectoryFrame) error {
	if frame.childCount > 0 {
		if _, err := renderer.stdout.Write([]byte(",\n")); err != nil {
			return err
		}
	} else {
		if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
			return err
		}
		frame.childrenOpened = true
	}
	return nil
}

func (renderer *jsonStreamRenderer) startObject(indentLevel int) error {
	if err := renderer.writeIndent(indentLevel); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte("{\n")); err != nil {
		return err
	}
	return nil
}

func (renderer *jsonStreamRenderer) writeFields(indent int, fields []jsonField, hasMore bool) error {
	if len(fields) == 0 {
		return nil
	}
	for index, field := range fields {
		suffix := ""
		if index < len(fields)-1 || hasMore {
			suffix = ","
		}
		line := fmt.Sprintf("\"%s\": %s%s", field.key, field.value, suffix)
		if err := renderer.writeLine(indent, line); err != nil {
			return err
		}
	}
	return nil
}

func (renderer *jsonStreamRenderer) directoryFields(directory *stream.DirectoryEvent) []jsonField {
	fields := []jsonField{
		{key: "path", value: encodeJSONString(directory.Path)},
	}
	if directory.Name != "" {
		fields = append(fields, jsonField{key: "name", value: encodeJSONString(directory.Name)})
	}
	fields = append(fields, jsonField{key: "type", value: encodeJSONString(types.NodeTypeDirectory)})
	if directory.LastModified != "" {
		fields = append(fields, jsonField{key: "lastModified", value: encodeJSONString(directory.LastModified)})
	}
	return fields
}

func (renderer *jsonStreamRenderer) summaryFields(summary *stream.SummaryEvent) []jsonField {
	if summary == nil {
		return nil
	}
	var fields []jsonField
	if summary.Model != "" {
		fields = append(fields, jsonField{key: "model", value: encodeJSONString(summary.Model)})
	}
	fields = append(fields, jsonField{key: "totalFiles", value: strconv.Itoa(summary.Files)})
	fields = append(fields, jsonField{key: "totalSize", value: encodeJSONString(utils.FormatFileSize(summary.Bytes))})
	if summary.Tokens > 0 {
		fields = append(fields, jsonField{key: "totalTokens", value: strconv.Itoa(summary.Tokens)})
	}
	return fields
}

func (renderer *jsonStreamRenderer) buildFilePayload(builder *jsonFileBuilder, content string) jsonFilePayload {
	payload := jsonFilePayload{
		Path: builder.path,
		Name: builder.name,
		Type: builder.nodeType,
	}
	if builder.sizeDisplay != "" {
		payload.Size = builder.sizeDisplay
	}
	if builder.lastModified != "" {
		payload.LastModified = builder.lastModified
	}
	if builder.mimeType != "" {
		payload.MimeType = builder.mimeType
	}
	if builder.tokens > 0 {
		payload.Tokens = builder.tokens
	}
	if builder.model != "" {
		payload.Model = builder.model
	}
	if content != "" {
		payload.Content = content
	}
	if len(builder.documentation) > 0 {
		payload.Documentation = builder.documentation
	}
	return payload
}

func (renderer *jsonStreamRenderer) onChildClosed(parentDepth int, totalFiles int, totalBytes int64, totalTokens int, model string) {
	if parentDepth >= 0 {
		parent := renderer.frameForDepth(parentDepth)
		if parent != nil {
			parent.childCount++
		}
		return
	}
	renderer.rootsClosed++
}

func (renderer *jsonStreamRenderer) frameForDepth(depth int) *jsonDirectoryFrame {
	for index := len(renderer.dirStack) - 1; index >= 0; index-- {
		frame := renderer.dirStack[index]
		if frame.depth == depth {
			return frame
		}
	}
	return nil
}

func (renderer *jsonStreamRenderer) popFrame() *jsonDirectoryFrame {
	if len(renderer.dirStack) == 0 {
		return nil
	}
	last := renderer.dirStack[len(renderer.dirStack)-1]
	renderer.dirStack = renderer.dirStack[:len(renderer.dirStack)-1]
	return last
}

func (renderer *jsonStreamRenderer) writeIndent(level int) error {
	if renderer.stdout == nil {
		return nil
	}
	for count := 0; count < level; count++ {
		if _, err := renderer.stdout.Write([]byte(indentSpacer)); err != nil {
			return err
		}
	}
	return nil
}

func (renderer *jsonStreamRenderer) writeLine(level int, content string) error {
	if err := renderer.writeIndent(level); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte(content)); err != nil {
		return err
	}
	if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func (renderer *jsonStreamRenderer) baseIndent(depth int) int {
	return depth + renderer.rootIndent
}

func (renderer *jsonStreamRenderer) closeArrayIfNeeded() error {
	if !renderer.arrayOpened || renderer.arrayClosed {
		return nil
	}
	if renderer.rootsClosed > 0 {
		if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if _, err := renderer.stdout.Write([]byte("]")); err != nil {
		return err
	}
	renderer.arrayClosed = true
	return nil
}

func (renderer *jsonStreamRenderer) writeIndentedBlock(indentLevel int, block []byte) error {
	lines := bytes.Split(block, []byte("\n"))
	for index, line := range lines {
		if err := renderer.writeIndent(indentLevel); err != nil {
			return err
		}
		if _, err := renderer.stdout.Write(line); err != nil {
			return err
		}
		if index < len(lines)-1 {
			if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
				return err
			}
		}
	}
	return nil
}

func encodeJSONString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "\"\""
	}
	return string(encoded)
}
