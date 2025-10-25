package docs

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/tyemirov/ctx/internal/types"
)

type javaScriptExtractor struct{}

func newJavaScriptExtractor() documentationExtractor {
	return &javaScriptExtractor{}
}

func (extractor *javaScriptExtractor) SupportedExtensions() []string {
	return []string{javaScriptFileExtension}
}

func (extractor *javaScriptExtractor) RequiresSource() bool {
	return true
}

func (extractor *javaScriptExtractor) CollectDocumentation(filePath string, fileContent []byte) ([]types.DocumentationEntry, error) {
	if len(fileContent) == 0 {
		return nil, nil
	}
	normalized := strings.ReplaceAll(string(fileContent), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	moduleName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	commentBuffer := make([]string, 0)
	entries := make([]types.DocumentationEntry, 0)

	lineIndex := 0
	for lineIndex < len(lines) {
		currentLine := strings.TrimSpace(lines[lineIndex])
		if strings.HasPrefix(currentLine, "/**") {
			collected, nextIndex := collectJSDoc(lines, lineIndex)
			commentBuffer = collected
			lineIndex = nextIndex
			continue
		}
		if len(commentBuffer) == 0 {
			lineIndex++
			continue
		}
		if currentLine == "" || strings.HasPrefix(currentLine, "//") {
			lineIndex++
			continue
		}
		declarationName, declarationKind, matched := matchJavaScriptDeclaration(currentLine)
		if matched {
			qualifiedName := moduleName + "." + declarationName
			documentationText := formatJSDoc(commentBuffer)
			if documentationText != "" {
				entries = append(entries, types.DocumentationEntry{
					Kind: declarationKind,
					Name: qualifiedName,
					Doc:  documentationText,
				})
			}
		}
		commentBuffer = nil
		lineIndex++
	}

	return entries, nil
}

func collectJSDoc(lines []string, startIndex int) ([]string, int) {
	buffer := make([]string, 0)
	currentIndex := startIndex
	for currentIndex < len(lines) {
		trimmed := strings.TrimSpace(lines[currentIndex])
		if currentIndex == startIndex {
			trimmed = strings.TrimPrefix(trimmed, "/**")
			trimmed = strings.TrimLeftFunc(trimmed, unicode.IsSpace)
			if strings.Contains(trimmed, "*/") {
				parts := strings.SplitN(trimmed, "*/", 2)
				content := strings.TrimSpace(parts[0])
				if content != "" {
					buffer = append(buffer, content)
				}
				return buffer, currentIndex + 1
			}
			if trimmed != "" {
				buffer = append(buffer, trimmed)
			}
			currentIndex++
			continue
		}
		if strings.Contains(trimmed, "*/") {
			parts := strings.SplitN(trimmed, "*/", 2)
			content := strings.TrimSpace(parts[0])
			if content != "" {
				buffer = append(buffer, content)
			}
			return buffer, currentIndex + 1
		}
		buffer = append(buffer, trimmed)
		currentIndex++
	}
	return buffer, currentIndex
}

func formatJSDoc(lines []string) string {
	processed := make([]string, 0, len(lines))
	for _, line := range lines {
		cleaned := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == "" {
			continue
		}
		if strings.HasPrefix(cleaned, "@") {
			continue
		}
		processed = append(processed, cleaned)
	}
	return strings.Join(processed, "\n")
}

func matchJavaScriptDeclaration(line string) (string, string, bool) {
	trimmed := line
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(trimmed[len("export "):])
	}
	if strings.HasPrefix(trimmed, "default ") {
		trimmed = strings.TrimSpace(trimmed[len("default "):])
	}
	if strings.HasPrefix(trimmed, "async ") {
		trimmed = strings.TrimSpace(trimmed[len("async "):])
	}
	if strings.HasPrefix(trimmed, "function ") {
		remainder := strings.TrimSpace(trimmed[len("function "):])
		name := readJavaScriptIdentifier(remainder)
		if name != "" {
			return name, documentationKindSymbol, true
		}
		return "", "", false
	}
	if strings.HasPrefix(trimmed, "class ") {
		remainder := strings.TrimSpace(trimmed[len("class "):])
		name := readJavaScriptIdentifier(remainder)
		if name != "" {
			return name, documentationKindClass, true
		}
		return "", "", false
	}
	for _, prefix := range []string{"const ", "let ", "var "} {
		if strings.HasPrefix(trimmed, prefix) {
			remainder := strings.TrimSpace(trimmed[len(prefix):])
			name := readJavaScriptIdentifier(remainder)
			remainder = strings.TrimSpace(remainder[len(name):])
			if name == "" || !strings.HasPrefix(remainder, "=") {
				return "", "", false
			}
			initializer := strings.TrimSpace(remainder[1:])
			if strings.HasPrefix(initializer, "function") || strings.Contains(initializer, "=>") {
				return name, documentationKindSymbol, true
			}
			if strings.HasPrefix(initializer, "class") {
				return name, documentationKindClass, true
			}
			return "", "", false
		}
	}
	return "", "", false
}

func readJavaScriptIdentifier(input string) string {
	builder := strings.Builder{}
	for _, runeValue := range input {
		if unicode.IsLetter(runeValue) || unicode.IsDigit(runeValue) || runeValue == '_' || runeValue == '$' {
			builder.WriteRune(runeValue)
			continue
		}
		break
	}
	return builder.String()
}
