package output

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/tyemirov/ctx/internal/types"
	"github.com/tyemirov/ctx/internal/utils"
)

type toonBuilder struct {
	buffer bytes.Buffer
}

func (builder *toonBuilder) String() string {
	if builder.buffer.Len() == 0 {
		return ""
	}
	if builder.buffer.Bytes()[builder.buffer.Len()-1] != '\n' {
		builder.buffer.WriteByte('\n')
	}
	return builder.buffer.String()
}

func (builder *toonBuilder) writeTreeNodes(name string, nodes []*types.TreeOutputNode) {
	builder.writeArrayHeader(0, name, len(nodes))
	for _, node := range nodes {
		builder.writeTreeNode(1, node)
	}
}

func (builder *toonBuilder) writeTreeNode(indent int, node *types.TreeOutputNode) {
	if node == nil {
		return
	}
	fields := assembleTreeFields(node)
	builder.writeObjectInArray(indent, fields)
}

func assembleTreeFields(node *types.TreeOutputNode) []toonField {
	fields := []toonField{
		{key: "path", value: node.Path},
	}
	if node.Name != "" {
		fields = append(fields, toonField{key: "name", value: node.Name})
	}
	if node.Type != "" {
		fields = append(fields, toonField{key: "type", value: node.Type})
	}
	if node.Size != "" {
		fields = append(fields, toonField{key: "size", value: node.Size})
	}
	if node.TotalFiles > 0 {
		fields = append(fields, toonField{key: "totalFiles", value: node.TotalFiles})
	}
	if node.TotalSize != "" {
		fields = append(fields, toonField{key: "totalSize", value: node.TotalSize})
	}
	if node.TotalTokens > 0 {
		fields = append(fields, toonField{key: "totalTokens", value: node.TotalTokens})
	}
	if node.LastModified != "" {
		fields = append(fields, toonField{key: "lastModified", value: node.LastModified})
	}
	if node.MimeType != "" {
		fields = append(fields, toonField{key: "mimeType", value: node.MimeType})
	}
	if node.Tokens > 0 {
		fields = append(fields, toonField{key: "tokens", value: node.Tokens})
	}
	if node.Model != "" {
		fields = append(fields, toonField{key: "model", value: node.Model})
	}
	if node.Content != "" {
		fields = append(fields, toonField{key: "content", value: node.Content})
	}
	if len(node.Children) > 0 {
		fields = append(fields, toonField{key: "children", value: node.Children})
	}
	if len(node.Documentation) > 0 {
		fields = append(fields, toonField{key: "documentation", value: node.Documentation})
	}
	return fields
}

type toonField struct {
	key   string
	value interface{}
}

func (builder *toonBuilder) writeObjectInArray(indent int, fields []toonField) {
	if len(fields) == 0 {
		builder.writeIndent(indent)
		builder.buffer.WriteString("-\n")
		return
	}
	builder.writeFieldWithPrefix(indent, "- ", fields[0])
	for _, field := range fields[1:] {
		builder.writeField(indent+1, field)
	}
}

func (builder *toonBuilder) writeField(indent int, field toonField) {
	builder.writeFieldWithPrefix(indent, "", field)
}

func (builder *toonBuilder) writeFieldWithPrefix(indent int, prefix string, field toonField) {
	switch value := field.value.(type) {
	case []*types.TreeOutputNode:
		builder.writeArray(indent, prefix, field.key, len(value))
		for _, child := range value {
			builder.writeTreeNode(indent+1, child)
		}
	case []types.DocumentationEntry:
		builder.writeArray(indent, prefix, field.key, len(value))
		for _, entry := range value {
			builder.writeDocumentationEntry(indent+1, entry)
		}
	default:
		if scalar, ok := formatScalar(value); ok {
			builder.writeIndent(indent)
			builder.buffer.WriteString(prefix)
			builder.buffer.WriteString(field.key)
			builder.buffer.WriteString(": ")
			builder.buffer.WriteString(scalar)
			builder.buffer.WriteByte('\n')
			return
		}
		builder.writeIndent(indent)
		builder.buffer.WriteString(prefix)
		builder.buffer.WriteString(field.key)
		builder.buffer.WriteString(":\n")
		builder.writeIndent(indent + 1)
		builder.buffer.WriteString(formatToonString(fmt.Sprint(value)))
		builder.buffer.WriteByte('\n')
	}
}

func (builder *toonBuilder) writeDocumentationEntry(indent int, entry types.DocumentationEntry) {
	fields := []toonField{
		{key: "type", value: entry.Kind},
		{key: "name", value: entry.Name},
	}
	if entry.Doc != "" {
		fields = append(fields, toonField{key: "documentation", value: entry.Doc})
	}
	builder.writeObjectInArray(indent, fields)
}

func (builder *toonBuilder) writeSummary(summary toonSummary) {
	builder.ensureLineBreak()
	builder.writeIndent(0)
	builder.buffer.WriteString("summary:\n")
	builder.writeIndent(1)
	builder.buffer.WriteString("totalFiles: ")
	builder.buffer.WriteString(strconv.Itoa(summary.files))
	builder.buffer.WriteByte('\n')
	builder.writeIndent(1)
	builder.buffer.WriteString("totalSize: ")
	builder.buffer.WriteString(utils.FormatFileSize(summary.bytes))
	builder.buffer.WriteByte('\n')
	if summary.tokens > 0 {
		builder.writeIndent(1)
		builder.buffer.WriteString("totalTokens: ")
		builder.buffer.WriteString(strconv.Itoa(summary.tokens))
		builder.buffer.WriteByte('\n')
	}
	if summary.model != "" {
		builder.writeIndent(1)
		builder.buffer.WriteString("model: ")
		builder.buffer.WriteString(formatToonString(summary.model))
		builder.buffer.WriteByte('\n')
	}
}

func (builder *toonBuilder) writeScalarArray(indent int, key string, values []string) {
	builder.writeIndent(indent)
	builder.buffer.WriteString(key)
	builder.buffer.WriteString("[")
	builder.buffer.WriteString(strconv.Itoa(len(values)))
	builder.buffer.WriteString("]:")
	if len(values) == 0 {
		builder.buffer.WriteByte('\n')
		return
	}
	formatted := make([]string, len(values))
	for index, value := range values {
		formatted[index] = formatToonString(value)
	}
	builder.buffer.WriteByte(' ')
	builder.buffer.WriteString(strings.Join(formatted, ","))
	builder.buffer.WriteByte('\n')
}

func (builder *toonBuilder) writeArray(indent int, prefix string, name string, length int) {
	builder.writeIndent(indent)
	builder.buffer.WriteString(prefix)
	builder.buffer.WriteString(name)
	builder.buffer.WriteString("[")
	builder.buffer.WriteString(strconv.Itoa(length))
	builder.buffer.WriteString("]:\n")
}

func (builder *toonBuilder) writeArrayHeader(indent int, name string, length int) {
	builder.writeIndent(indent)
	builder.buffer.WriteString(name)
	builder.buffer.WriteString("[")
	builder.buffer.WriteString(strconv.Itoa(length))
	builder.buffer.WriteString("]:\n")
}

func (builder *toonBuilder) writeIndent(level int) {
	for index := 0; index < level; index++ {
		builder.buffer.WriteString("  ")
	}
}

func (builder *toonBuilder) ensureLineBreak() {
	if builder.buffer.Len() == 0 {
		return
	}
	if builder.buffer.Bytes()[builder.buffer.Len()-1] != '\n' {
		builder.buffer.WriteByte('\n')
	}
}

func formatScalar(value interface{}) (string, bool) {
	switch typed := value.(type) {
	case string:
		return formatToonString(typed), true
	case int:
		return strconv.Itoa(typed), true
	case int8:
		return strconv.FormatInt(int64(typed), 10), true
	case int16:
		return strconv.FormatInt(int64(typed), 10), true
	case int32:
		return strconv.FormatInt(int64(typed), 10), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case uint:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint8:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint16:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint32:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint64:
		return strconv.FormatUint(typed, 10), true
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case bool:
		if typed {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func formatToonString(value string) string {
	if needsQuote(value) {
		return strconv.Quote(value)
	}
	return value
}

func needsQuote(value string) bool {
	if value == "" {
		return true
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			continue
		case r >= 'A' && r <= 'Z':
			continue
		case r >= '0' && r <= '9':
			continue
		case r == '-', r == '_', r == '.', r == '/', r == '\\', r == '@', r == '~', r == '+':
			continue
		default:
			return true
		}
	}
	return false
}
