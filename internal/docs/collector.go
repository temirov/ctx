// Package docs provides documentation extraction for source files referenced by ctx when the --doc flag is used.
package docs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/ctx/internal/types"
)

type documentationExtractor interface {
	SupportedExtensions() []string
	RequiresSource() bool
	CollectDocumentation(filePath string, fileContent []byte) ([]types.DocumentationEntry, error)
}

// Collector routes documentation lookups to language-specific extractors.
type Collector struct {
	extensionToExtractor map[string]documentationExtractor
}

// NewCollector creates a Collector using the repository root that contains go.mod.
func NewCollector(repositoryRoot string) (*Collector, error) {
	extensionToExtractor := map[string]documentationExtractor{}
	moduleRoot, moduleLocateError := findModuleRoot(repositoryRoot)
	if moduleLocateError != nil {
		return nil, moduleLocateError
	}
	if moduleRoot != "" {
		goExtractor, goExtractorError := newGoExtractor(moduleRoot)
		if goExtractorError != nil {
			return nil, goExtractorError
		}
		registerExtractor(extensionToExtractor, goExtractor)
	}
	registerExtractor(extensionToExtractor, newPythonExtractor())
	registerExtractor(extensionToExtractor, newJavaScriptExtractor())
	return &Collector{extensionToExtractor: extensionToExtractor}, nil
}

func registerExtractor(extensionToExtractor map[string]documentationExtractor, extractor documentationExtractor) {
	for _, extension := range extractor.SupportedExtensions() {
		normalizedExtension := strings.ToLower(extension)
		extensionToExtractor[normalizedExtension] = extractor
	}
}

// CollectFromFile returns documentation entries for the provided source file.
func (collector *Collector) CollectFromFile(filePath string) ([]types.DocumentationEntry, error) {
	if collector == nil {
		return nil, nil
	}
	extension := strings.ToLower(filepath.Ext(filePath))
	extractor, found := collector.extensionToExtractor[extension]
	if !found {
		return nil, nil
	}
	var fileContent []byte
	var readError error
	if extractor.RequiresSource() {
		fileContent, readError = os.ReadFile(filePath)
		if readError != nil {
			return nil, readError
		}
	}
	documentationEntries, extractionError := extractor.CollectDocumentation(filePath, fileContent)
	if extractionError != nil {
		return nil, extractionError
	}
	return documentationEntries, nil
}

func findModuleRoot(startingPath string) (string, error) {
	currentPath := startingPath
	for {
		goModPath := filepath.Join(currentPath, "go.mod")
		info, statError := os.Stat(goModPath)
		if statError == nil {
			if info.IsDir() {
				return "", errors.New("go.mod path refers to a directory")
			}
			return currentPath, nil
		}
		if !errors.Is(statError, os.ErrNotExist) {
			return "", statError
		}
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			return "", nil
		}
		currentPath = parentPath
	}
}
