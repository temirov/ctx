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

// IgnoreFileSections represents patterns grouped by category within an ignore file.
type IgnoreFileSections struct {
	Ignore []string
	Binary []string
}

// LoadIgnoreFilePatterns reads a specified ignore file (if it exists) and returns categorized patterns.
// Supported sections are [ignore] and [binary]. Lines before any section header and files without
// headers default to the [ignore] section. Blank lines and lines beginning with '#' are skipped.
//
// #nosec G304
func LoadIgnoreFilePatterns(ignoreFilePath string) (*IgnoreFileSections, error) {
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

	sections := &IgnoreFileSections{}
	currentSection := "ignore"

	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		lineValue := strings.TrimSpace(scanner.Text())
		if lineValue == "" || strings.HasPrefix(lineValue, "#") {
			continue
		}
		if strings.HasPrefix(lineValue, "[") && strings.HasSuffix(lineValue, "]") {
			header := strings.ToLower(strings.Trim(lineValue, "[]"))
			switch header {
			case "ignore":
				currentSection = "ignore"
			case "binary":
				currentSection = "binary"
			default:
				currentSection = "ignore"
			}
			continue
		}

		switch currentSection {
		case "binary":
			sections.Binary = append(sections.Binary, lineValue)
		default:
			sections.Ignore = append(sections.Ignore, lineValue)
		}
	}
	if scanError := scanner.Err(); scanError != nil {
		return nil, scanError
	}
	return sections, nil
}

// LoadCombinedIgnorePatterns loads patterns from .ignore and/or .gitignore files within a directory,
// adds the exclusion folder pattern if specified, and returns the combined, deduplicated list.
func LoadCombinedIgnorePatterns(absoluteDirectoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool) ([]string, error) {
	var combinedPatterns []string

	if useIgnoreFile {
		ignoreFilePath := filepath.Join(absoluteDirectoryPath, ignoreFileName)
		ignoreFilePatterns, loadError := LoadIgnoreFilePatterns(ignoreFilePath)
		if loadError != nil {
			return nil, fmt.Errorf("loading %s from %s: %w", ignoreFileName, absoluteDirectoryPath, loadError)
		}
		if ignoreFilePatterns != nil {
			combinedPatterns = append(combinedPatterns, ignoreFilePatterns.Ignore...)
		}
	}

	if useGitignore {
		gitIgnoreFilePath := filepath.Join(absoluteDirectoryPath, gitIgnoreFileName)
		gitignoreFilePatterns, loadError := LoadIgnoreFilePatterns(gitIgnoreFilePath)
		if loadError != nil {
			return nil, fmt.Errorf("loading %s from %s: %w", gitIgnoreFileName, absoluteDirectoryPath, loadError)
		}
		if gitignoreFilePatterns != nil {
			combinedPatterns = append(combinedPatterns, gitignoreFilePatterns.Ignore...)
		}
	}

	deduplicatedFilePatterns := utils.DeduplicatePatterns(combinedPatterns)

	trimmedExclusion := strings.TrimSpace(exclusionFolder)
	if trimmedExclusion != "" {
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
