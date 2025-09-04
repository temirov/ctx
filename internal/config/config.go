// Package config loads and parses ignore files into slices of patterns.
package config

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/ctx/internal/utils"
)

// gitDirectoryPattern represents the pattern that matches the Git directory.
const gitDirectoryPattern = utils.GitDirectoryName + "/"

// showBinaryContentDirective marks patterns whose binary content should be displayed.
const showBinaryContentDirective = "show-binary-content:"

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
	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		trimmedLine := strings.TrimSpace(scanner.Text())
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		if strings.HasPrefix(trimmedLine, showBinaryContentDirective) {
			pattern := strings.TrimSpace(strings.TrimPrefix(trimmedLine, showBinaryContentDirective))
			if pattern != "" {
				binaryContentPatterns = append(binaryContentPatterns, pattern)
			}
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
// The exclusion folder pattern is appended when provided.
func LoadCombinedIgnorePatterns(absoluteDirectoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, includeGit bool) ([]string, error) {
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

	trimmedExclusion := strings.TrimSpace(exclusionFolder)
	if trimmedExclusion != "" {
		normalizedExclusion := strings.TrimSuffix(trimmedExclusion, "/")
		exclusionPattern := utils.ExclusionPrefix + normalizedExclusion
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

// LoadRecursiveIgnorePatterns traverses a directory tree rooted at rootDirectoryPath and aggregates ignore patterns and binary content patterns.
// Patterns from .ignore and .gitignore files found in each directory are prefixed with that directory's relative path.
// The .git directory is ignored by default unless includeGit is true. The exclusion folder pattern is appended when provided.
func LoadRecursiveIgnorePatterns(rootDirectoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, includeGit bool) ([]string, []string, error) {
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

	trimmedExclusion := strings.TrimSpace(exclusionFolder)
	if trimmedExclusion != "" {
		normalizedExclusion := strings.TrimSuffix(trimmedExclusion, "/")
		exclusionPattern := utils.ExclusionPrefix + normalizedExclusion
		isPresent := false
		for _, pattern := range deduplicatedPatterns {
			if pattern == exclusionPattern {
				isPresent = true
				break
			}
		}
		if !isPresent {
			deduplicatedPatterns = append(deduplicatedPatterns, exclusionPattern)
		}
	}

	return deduplicatedPatterns, deduplicatedBinaryPatterns, nil
}
