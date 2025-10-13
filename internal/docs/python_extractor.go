package docs

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/temirov/ctx/internal/types"
)

type pythonExtractor struct{}

func newPythonExtractor() documentationExtractor {
	return &pythonExtractor{}
}

func (extractor *pythonExtractor) SupportedExtensions() []string {
	return []string{pythonFileExtension}
}

func (extractor *pythonExtractor) RequiresSource() bool {
	return true
}

func (extractor *pythonExtractor) CollectDocumentation(filePath string, fileContent []byte) ([]types.DocumentationEntry, error) {
	if len(fileContent) == 0 {
		return nil, nil
	}
	normalized := strings.ReplaceAll(string(fileContent), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	moduleName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	entries := make([]types.DocumentationEntry, 0)

	lineIndex := 0
	moduleDoc, nextIndex, moduleFound := extractPythonModuleDocstring(lines)
	if moduleFound {
		entries = append(entries, types.DocumentationEntry{
			Kind: documentationKindModule,
			Name: moduleName,
			Doc:  moduleDoc,
		})
		lineIndex = nextIndex
	}

	blockStack := make([]pythonBlock, 0)
	for ; lineIndex < len(lines); lineIndex++ {
		currentLine := lines[lineIndex]
		trimmedLine := strings.TrimSpace(currentLine)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		indentation := countIndentation(currentLine)
		for len(blockStack) > 0 && indentation <= blockStack[len(blockStack)-1].indentation {
			blockStack = blockStack[:len(blockStack)-1]
		}
		if className, ok := matchPythonClass(trimmedLine); ok {
			documentationText, consumedIndex, foundDoc := extractPythonBlockDocstring(lines, lineIndex+1, indentation)
			if foundDoc {
				qualifiedName := buildPythonQualifiedName(moduleName, blockStack, className)
				entries = append(entries, types.DocumentationEntry{
					Kind: documentationKindClass,
					Name: qualifiedName,
					Doc:  documentationText,
				})
				lineIndex = consumedIndex - 1
			}
			blockStack = append(blockStack, pythonBlock{indentation: indentation, name: className, kind: documentationKindClass})
			continue
		}
		if functionName, ok := matchPythonFunction(trimmedLine); ok {
			documentationText, consumedIndex, foundDoc := extractPythonBlockDocstring(lines, lineIndex+1, indentation)
			if foundDoc {
				entryKind := documentationKindSymbol
				if containsClassContext(blockStack) {
					entryKind = documentationKindMethod
				}
				qualifiedName := buildPythonQualifiedName(moduleName, blockStack, functionName)
				entries = append(entries, types.DocumentationEntry{
					Kind: entryKind,
					Name: qualifiedName,
					Doc:  documentationText,
				})
				lineIndex = consumedIndex - 1
			}
			blockStack = append(blockStack, pythonBlock{indentation: indentation, name: functionName, kind: documentationKindSymbol})
		}
	}

	return entries, nil
}

type pythonBlock struct {
	indentation int
	name        string
	kind        string
}

func extractPythonModuleDocstring(lines []string) (string, int, bool) {
	for index := 0; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "#!") {
			continue
		}
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			return "", index, false
		}
		if documentationText, consumed, found := extractPythonTripleQuotedString(lines, index); found {
			return documentationText, consumed, true
		}
		return "", index, false
	}
	return "", len(lines), false
}

func matchPythonClass(line string) (string, bool) {
	if strings.HasPrefix(line, "class ") {
		name := line[len("class "):]
		name = strings.SplitN(name, "(", 2)[0]
		name = strings.SplitN(name, ":", 2)[0]
		name = strings.TrimSpace(name)
		if name != "" {
			return name, true
		}
	}
	return "", false
}

func matchPythonFunction(line string) (string, bool) {
	trimmed := line
	if strings.HasPrefix(trimmed, "async ") {
		trimmed = strings.TrimSpace(trimmed[len("async "):])
	}
	if strings.HasPrefix(trimmed, "def ") {
		name := trimmed[len("def "):]
		name = strings.SplitN(name, "(", 2)[0]
		name = strings.TrimSpace(name)
		if name != "" {
			return name, true
		}
	}
	return "", false
}

func extractPythonBlockDocstring(lines []string, startIndex int, parentIndentation int) (string, int, bool) {
	currentIndex := startIndex
	for currentIndex < len(lines) {
		trimmed := strings.TrimSpace(lines[currentIndex])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			currentIndex++
			continue
		}
		indentation := countIndentation(lines[currentIndex])
		if indentation <= parentIndentation {
			return "", startIndex, false
		}
		if documentationText, consumed, found := extractPythonTripleQuotedString(lines, currentIndex); found {
			return documentationText, consumed, true
		}
		return "", startIndex, false
	}
	return "", startIndex, false
}

func extractPythonTripleQuotedString(lines []string, startIndex int) (string, int, bool) {
	trimmed := strings.TrimSpace(lines[startIndex])
	quoteToken, prefixLength, ok := detectPythonTripleQuote(trimmed)
	if !ok {
		return "", startIndex, false
	}
	contentStart := prefixLength + len(quoteToken)
	remainder := trimmed[contentStart:]
	if closingIndex := strings.Index(remainder, quoteToken); closingIndex >= 0 {
		raw := remainder[:closingIndex]
		return strings.TrimSpace(raw), startIndex + 1, true
	}
	var builder strings.Builder
	firstLine := trimmed[contentStart:]
	if firstLine != "" {
		builder.WriteString(firstLine)
		builder.WriteString("\n")
	}
	for lineIndex := startIndex + 1; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		if strings.Contains(line, quoteToken) {
			parts := strings.SplitN(line, quoteToken, 2)
			builder.WriteString(strings.TrimRightFunc(parts[0], unicode.IsSpace))
			raw := builder.String()
			return dedentPythonDocstring(raw), lineIndex + 1, true
		}
		builder.WriteString(strings.TrimRightFunc(line, unicode.IsSpace))
		builder.WriteString("\n")
	}
	return "", startIndex, false
}

func detectPythonTripleQuote(line string) (string, int, bool) {
	for _, token := range []string{"\"\"\"", "'''"} {
		index := strings.Index(line, token)
		if index < 0 {
			continue
		}
		prefix := line[:index]
		if isValidPythonStringPrefix(prefix) {
			return token, index, true
		}
	}
	return "", 0, false
}

func isValidPythonStringPrefix(prefix string) bool {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return true
	}
	for _, runeValue := range trimmed {
		lower := unicode.ToLower(runeValue)
		if lower != 'r' && lower != 'u' && lower != 'f' && lower != 'b' {
			return false
		}
	}
	return true
}

func dedentPythonDocstring(raw string) string {
	trimmed := strings.Trim(raw, "\n")
	if trimmed == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	minimalIndentation := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := countIndentation(line)
		if minimalIndentation == -1 || indent < minimalIndentation {
			minimalIndentation = indent
		}
	}
	if minimalIndentation > 0 {
		for index := range lines {
			if len(lines[index]) >= minimalIndentation {
				lines[index] = strings.TrimRightFunc(lines[index][minimalIndentation:], unicode.IsSpace)
			}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func countIndentation(line string) int {
	indentation := 0
	for _, runeValue := range line {
		if runeValue == ' ' {
			indentation++
			continue
		}
		if runeValue == '\t' {
			indentation += 4
			continue
		}
		break
	}
	return indentation
}

func containsClassContext(stack []pythonBlock) bool {
	for _, block := range stack {
		if block.kind == documentationKindClass {
			return true
		}
	}
	return false
}

func buildPythonQualifiedName(moduleName string, stack []pythonBlock, entityName string) string {
	components := []string{moduleName}
	for _, block := range stack {
		if block.kind == documentationKindClass {
			components = append(components, block.name)
		}
	}
	components = append(components, entityName)
	return strings.Join(components, ".")
}
