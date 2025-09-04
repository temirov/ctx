// Package config loads and parses ignore files into slices of patterns.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/ctx/utils"
)

const (
	ignoreFileName    = ".ignore"
	gitIgnoreFileName = ".gitignore"
	exclusionPrefix   = "EXCL:"
)

// LoadIgnoreFilePatterns reads a specified ignore file (if it exists) and returns a slice of ignore patterns.
// Blank lines and lines beginning with '#' are skipped.
//
// #nosec G304
func LoadIgnoreFilePatterns(ignoreFilePath string) ([]string, error) {
	fileHandle, openError := os.Open(ignoreFilePath)
	if openError != nil {
		if os.IsNotExist(openError) {
			return nil, nil
		}
		return nil, openError
	}
	defer func() {
		closeErr := fileHandle.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close %s: %v\n", ignoreFilePath, closeErr)
		}
	}()

	var patterns []string
	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		lineValue := strings.TrimSpace(scanner.Text())
		if lineValue == "" || strings.HasPrefix(lineValue, "#") {
			continue
		}
		patterns = append(patterns, lineValue)
	}
	if scanError := scanner.Err(); scanError != nil {
		return nil, scanError
	}
	return patterns, nil
}

// LoadCombinedIgnorePatterns loads patterns from .ignore and/or .gitignore files within a directory,
// optionally adds the exclusion folder pattern, and returns the combined, deduplicated list.
func LoadCombinedIgnorePatterns(
	absoluteDirectoryPath string,
	exclusionFolder string,
	useGitignore bool,
	useIgnoreFile bool,
	includeExclusion bool,
) ([]string, error) {
	var combinedPatterns []string

	if useIgnoreFile {
		ignoreFilePath := filepath.Join(absoluteDirectoryPath, ignoreFileName)
		ignoreFilePatterns, loadError := LoadIgnoreFilePatterns(ignoreFilePath)
		if loadError != nil {
			return nil, fmt.Errorf("loading %s from %s: %w", ignoreFileName, absoluteDirectoryPath, loadError)
		}
		combinedPatterns = append(combinedPatterns, ignoreFilePatterns...)
	}

	if useGitignore {
		gitIgnoreFilePath := filepath.Join(absoluteDirectoryPath, gitIgnoreFileName)
		gitignoreFilePatterns, loadError := LoadIgnoreFilePatterns(gitIgnoreFilePath)
		if loadError != nil {
			return nil, fmt.Errorf("loading %s from %s: %w", gitIgnoreFileName, absoluteDirectoryPath, loadError)
		}
		combinedPatterns = append(combinedPatterns, gitignoreFilePatterns...)
	}

	deduplicatedFilePatterns := utils.DeduplicatePatterns(combinedPatterns)

	trimmedExclusion := strings.TrimSpace(exclusionFolder)
	if includeExclusion && trimmedExclusion != "" {
		normalizedExclusion := strings.TrimSuffix(trimmedExclusion, "/")
		exclusionPattern := exclusionPrefix + normalizedExclusion
		isPresent := false
		for _, pattern := range deduplicatedFilePatterns {
			if pattern == exclusionPattern {
				isPresent = true
				break
			}
		}
		if !isPresent {
			deduplicatedFilePatterns = append(deduplicatedFilePatterns, exclusionPattern)
		}
	}

	return deduplicatedFilePatterns, nil
}
