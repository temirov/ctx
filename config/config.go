// Package config loads and parses ignore files into slices of patterns.
package config

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/ctx/utils"
)

const (
	ignoreFileName      = ".ignore"
	gitIgnoreFileName   = ".gitignore"
	exclusionPrefix     = "EXCL:"
	gitDirectoryPattern = utils.GitDirectoryName + "/"
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

// LoadCombinedIgnorePatterns aggregates patterns from .ignore and/or .gitignore files within a directory.
// The .git directory is excluded by default unless includeGit is true.
// The exclusion folder pattern is appended when provided.
func LoadCombinedIgnorePatterns(absoluteDirectoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, includeGit bool) ([]string, error) {
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

	if !includeGit {
		combinedPatterns = append(combinedPatterns, gitDirectoryPattern)
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

// LoadRecursiveIgnorePatterns traverses a directory tree rooted at rootDirectoryPath and aggregates ignore patterns.
// Patterns from .ignore and .gitignore files found in each directory are prefixed with that directory's relative path.
// The .git directory is ignored by default unless includeGit is true. The exclusion folder pattern is appended when provided.
func LoadRecursiveIgnorePatterns(rootDirectoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, includeGit bool) ([]string, error) {
	var aggregatedPatterns []string

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
			ignoreFilePath := filepath.Join(currentDirectoryPath, ignoreFileName)
			ignorePatterns, loadError := LoadIgnoreFilePatterns(ignoreFilePath)
			if loadError != nil {
				return fmt.Errorf("loading %s from %s: %w", ignoreFileName, currentDirectoryPath, loadError)
			}
			for _, pattern := range ignorePatterns {
				aggregatedPatterns = append(aggregatedPatterns, prefix+pattern)
			}
		}

		if useGitignore {
			gitIgnoreFilePath := filepath.Join(currentDirectoryPath, gitIgnoreFileName)
			gitIgnorePatterns, loadError := LoadIgnoreFilePatterns(gitIgnoreFilePath)
			if loadError != nil {
				return fmt.Errorf("loading %s from %s: %w", gitIgnoreFileName, currentDirectoryPath, loadError)
			}
			for _, pattern := range gitIgnorePatterns {
				aggregatedPatterns = append(aggregatedPatterns, prefix+pattern)
			}
		}

		return nil
	}

	if err := filepath.WalkDir(rootDirectoryPath, walkFunction); err != nil {
		return nil, err
	}

	if !includeGit {
		aggregatedPatterns = append(aggregatedPatterns, gitDirectoryPattern)
	}

	deduplicatedPatterns := utils.DeduplicatePatterns(aggregatedPatterns)

	trimmedExclusion := strings.TrimSpace(exclusionFolder)
	if trimmedExclusion != "" {
		normalizedExclusion := strings.TrimSuffix(trimmedExclusion, "/")
		exclusionPattern := exclusionPrefix + normalizedExclusion
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

	return deduplicatedPatterns, nil
}
