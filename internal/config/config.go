// Package config loads and parses ignore files into slices of patterns.
package config

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tyemirov/ctx/internal/utils"
)

const (
	// gitDirectoryPattern represents the pattern that matches the Git directory.
	gitDirectoryPattern = utils.GitDirectoryName + "/"
	// binarySectionHeader identifies the section listing binary content patterns.
	binarySectionHeader = "[binary]"
	// ignoreSectionHeader identifies the section listing ignore patterns.
	ignoreSectionHeader = "[ignore]"
)

// LoadIgnoreFilePatterns reads a specified ignore file and returns ignore patterns and binary content patterns.
//
// #nosec G304
func LoadIgnoreFilePatterns(ignoreFilePath string) ([]string, []string, error) {
	fileHandle, openFileError := os.Open(ignoreFilePath)
	if openFileError != nil {
		if os.IsNotExist(openFileError) {
			return nil, nil, nil
		}
		return nil, nil, openFileError
	}
	defer func() {
		closeError := fileHandle.Close()
		if closeError != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close %s: %v\n", ignoreFilePath, closeError)
		}
	}()

	var ignorePatterns []string
	var binaryContentPatterns []string
	currentSectionHeader := ignoreSectionHeader
	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		trimmedLine := strings.TrimSpace(scanner.Text())
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		if strings.EqualFold(trimmedLine, binarySectionHeader) {
			currentSectionHeader = binarySectionHeader
			continue
		}
		if strings.EqualFold(trimmedLine, ignoreSectionHeader) {
			currentSectionHeader = ignoreSectionHeader
			continue
		}
		if currentSectionHeader == binarySectionHeader {
			binaryContentPatterns = append(binaryContentPatterns, trimmedLine)
			continue
		}
		ignorePatterns = append(ignorePatterns, trimmedLine)
	}
	if scanError := scanner.Err(); scanError != nil {
		return nil, nil, scanError
	}
	return ignorePatterns, binaryContentPatterns, nil
}

// LoadCombinedIgnorePatterns aggregates patterns from .ignore and/or .gitignore files within a directory.
// The .git directory is excluded by default unless includeGit is true.
// The provided exclusionPatterns are appended to the result.
func LoadCombinedIgnorePatterns(absoluteDirectoryPath string, exclusionPatterns []string, useGitignore bool, useIgnoreFile bool, includeGit bool) ([]string, error) {
	var combinedPatterns []string

	if useIgnoreFile {
		ignoreFilePath := filepath.Join(absoluteDirectoryPath, utils.IgnoreFileName)
		ignoreFilePatterns, _, loadError := LoadIgnoreFilePatterns(ignoreFilePath)
		if loadError != nil {
			return nil, fmt.Errorf("loading %s from %s: %w", utils.IgnoreFileName, absoluteDirectoryPath, loadError)
		}
		combinedPatterns = append(combinedPatterns, ignoreFilePatterns...)
	}

	if useGitignore {
		gitIgnoreFilePath := filepath.Join(absoluteDirectoryPath, utils.GitIgnoreFileName)
		gitIgnoreFilePatterns, _, loadError := LoadIgnoreFilePatterns(gitIgnoreFilePath)
		if loadError != nil {
			return nil, fmt.Errorf("loading %s from %s: %w", utils.GitIgnoreFileName, absoluteDirectoryPath, loadError)
		}
		combinedPatterns = append(combinedPatterns, gitIgnoreFilePatterns...)
	}

	if !includeGit {
		combinedPatterns = append(combinedPatterns, gitDirectoryPattern)
	}

	deduplicatedFilePatterns := utils.DeduplicatePatterns(combinedPatterns)

	for _, pattern := range exclusionPatterns {
		trimmedPattern := strings.TrimSpace(pattern)
		if trimmedPattern == "" {
			continue
		}
		if !utils.ContainsString(deduplicatedFilePatterns, trimmedPattern) {
			deduplicatedFilePatterns = append(deduplicatedFilePatterns, trimmedPattern)
		}
	}

	return deduplicatedFilePatterns, nil
}

// LoadRecursiveIgnorePatterns walks rootDirectoryPath and aggregates ignore patterns and binary content patterns.
// Patterns from utils.IgnoreFileName and utils.GitIgnoreFileName in each nested directory are prefixed with that directory's
// path relative to rootDirectoryPath. For example, a pattern listed in utils.GitIgnoreFileName within a child directory is
// returned with the directory's relative path prepended. Patterns from utils.GitIgnoreFileName are handled the same way as
// those from utils.IgnoreFileName. The directory named utils.GitDirectoryName is ignored by default unless includeGit is
// true. The provided exclusionPatterns are appended to the result.
func LoadRecursiveIgnorePatterns(rootDirectoryPath string, exclusionPatterns []string, useGitignore bool, useIgnoreFile bool, includeGit bool) ([]string, []string, error) {
	var aggregatedPatterns []string
	var aggregatedBinaryContentPatterns []string

	walkFunction := func(currentDirectoryPath string, directoryEntry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if !directoryEntry.IsDir() {
			return nil
		}
		if !includeGit && directoryEntry.Name() == utils.GitDirectoryName {
			return filepath.SkipDir
		}

		relativeDirectory := utils.RelativePathOrSelf(currentDirectoryPath, rootDirectoryPath)
		prefix := ""
		if relativeDirectory != "." {
			prefix = relativeDirectory + "/"
		}

		if useIgnoreFile {
			ignoreFilePath := filepath.Join(currentDirectoryPath, utils.IgnoreFileName)
			ignorePatterns, binaryContentPatterns, loadError := LoadIgnoreFilePatterns(ignoreFilePath)
			if loadError != nil {
				return fmt.Errorf("loading %s from %s: %w", utils.IgnoreFileName, currentDirectoryPath, loadError)
			}
			for _, pattern := range ignorePatterns {
				aggregatedPatterns = append(aggregatedPatterns, prefix+pattern)
			}
			for _, binaryPattern := range binaryContentPatterns {
				aggregatedBinaryContentPatterns = append(aggregatedBinaryContentPatterns, prefix+binaryPattern)
			}
		}

		if useGitignore {
			gitIgnoreFilePath := filepath.Join(currentDirectoryPath, utils.GitIgnoreFileName)
			gitIgnorePatterns, _, loadError := LoadIgnoreFilePatterns(gitIgnoreFilePath)
			if loadError != nil {
				return fmt.Errorf("loading %s from %s: %w", utils.GitIgnoreFileName, currentDirectoryPath, loadError)
			}
			for _, pattern := range gitIgnorePatterns {
				aggregatedPatterns = append(aggregatedPatterns, prefix+pattern)
			}
		}

		return nil
	}

	if walkError := filepath.WalkDir(rootDirectoryPath, walkFunction); walkError != nil {
		return nil, nil, walkError
	}

	if !includeGit {
		aggregatedPatterns = append(aggregatedPatterns, gitDirectoryPattern)
	}

	deduplicatedPatterns := utils.DeduplicatePatterns(aggregatedPatterns)
	deduplicatedBinaryPatterns := utils.DeduplicatePatterns(aggregatedBinaryContentPatterns)

	for _, pattern := range exclusionPatterns {
		trimmedPattern := strings.TrimSpace(pattern)
		if trimmedPattern == "" {
			continue
		}
		if !utils.ContainsString(deduplicatedPatterns, trimmedPattern) {
			deduplicatedPatterns = append(deduplicatedPatterns, trimmedPattern)
		}
	}

	return deduplicatedPatterns, deduplicatedBinaryPatterns, nil
}
