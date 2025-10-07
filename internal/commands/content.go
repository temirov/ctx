package commands

import (
	"github.com/temirov/ctx/internal/tokenizer"
	"github.com/temirov/ctx/internal/types"
)

const (
	// WarningAccessPathFormat is used when a path cannot be accessed during traversal.
	WarningAccessPathFormat = "Warning: error accessing path %s: %v\n"
	// WarningFileReadFormat is used when a file cannot be read.
	WarningFileReadFormat = "Warning: failed to read file %s: %v\n"
	// WarningTokenCountFormat is used when token counting fails for a file.
	WarningTokenCountFormat = "Warning: failed to count tokens for %s: %v\n"
)

// GetContentData returns FileOutput slices for the specified directory.
func GetContentData(rootPath string, ignorePatterns []string, binaryContentPatterns []string, tokenCounter tokenizer.Counter, tokenModel string) ([]types.FileOutput, error) {
	var fileOutputs []types.FileOutput
	streamErr := StreamContent(rootPath, ignorePatterns, binaryContentPatterns, tokenCounter, tokenModel, func(output types.FileOutput) error {
		fileOutputs = append(fileOutputs, output)
		return nil
	})
	if streamErr != nil {
		return nil, streamErr
	}
	return fileOutputs, nil
}
