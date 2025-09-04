// Package utils contains general helper functions used across the content tool.
package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// Ignore file constants used across the project.
const (
	// IgnoreFileName is the name of the project's ignore file.
	IgnoreFileName = ".ignore"
	// GitIgnoreFileName is the name of the Git ignore file.
	GitIgnoreFileName = ".gitignore"
	// ExclusionPrefix marks patterns that exclude directories from processing.
	ExclusionPrefix = "EXCL:"
	// GitDirectoryName is the name of the Git repository directory.
	GitDirectoryName = ".git"
)

var serviceFiles = map[string]struct{}{
	IgnoreFileName:    {},
	GitIgnoreFileName: {},
}

// DeduplicatePatterns removes duplicate patterns from a slice while preserving order.
// The first occurrence of each unique pattern is kept.
func DeduplicatePatterns(patterns []string) []string {
	encounteredPatterns := make(map[string]struct{})
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if _, exists := encounteredPatterns[pattern]; !exists {
			encounteredPatterns[pattern] = struct{}{}
			result = append(result, pattern)
		}
	}
	return result
}

// ContainsString checks if a slice of strings contains a specific target string.
func ContainsString(stringSlice []string, targetString string) bool {
	for _, currentString := range stringSlice {
		if currentString == targetString {
			return true
		}
	}
	return false
}

// RelativePathOrSelf calculates the relative path from root to fullPath.
// Returns the cleaned fullPath if relative calculation fails.
// Returns "." if fullPath and root resolve to the same directory.
func RelativePathOrSelf(fullPath, root string) string {
	cleanPath := filepath.Clean(fullPath)
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return cleanPath
	}
	cleanAbsoluteRoot := filepath.Clean(absoluteRoot)

	if cleanPath == cleanAbsoluteRoot {
		return "."
	}

	relativePath, relErr := filepath.Rel(cleanAbsoluteRoot, cleanPath)
	if relErr != nil {
		return cleanPath
	}
	return filepath.ToSlash(relativePath)
}

// ShouldIgnore checks if a directory entry should be ignored based on its name and type,
// relative to a set of ignore patterns and whether it's at the root level of processing.
// Used primarily during tree building (os.ReadDir).
func ShouldIgnore(directoryEntry os.DirEntry, ignorePatterns []string, isRootLevel bool) bool {
	entryName := directoryEntry.Name()

	if _, isServiceFile := serviceFiles[entryName]; isServiceFile {
		return true
	}

	for _, patternValue := range ignorePatterns {
		if strings.HasPrefix(patternValue, ExclusionPrefix) {
			exclusionName := strings.TrimPrefix(patternValue, ExclusionPrefix)
			if isRootLevel && directoryEntry.IsDir() && entryName == exclusionName {
				return true
			}
			continue
		}

		if strings.HasSuffix(patternValue, "/") {
			patternDirectory := strings.TrimSuffix(patternValue, "/")
			if directoryEntry.IsDir() && entryName == patternDirectory {
				return true
			}
		} else {
			isMatched, matchError := filepath.Match(patternValue, entryName)
			if matchError == nil && isMatched {
				return true
			}
		}
	}
	return false
}

// ShouldIgnoreByPath checks if a path relative to its processing root should be ignored.
// It considers exclusion patterns, directory patterns, filename patterns, and service files.
// Used by the content command's walk function (filepath.WalkDir).
func ShouldIgnoreByPath(relativePath string, ignorePatterns []string) bool {
	normalizedPath := filepath.ToSlash(relativePath)
	pathComponents := strings.Split(normalizedPath, "/")
	entryName := ""
	if len(pathComponents) > 0 {
		entryName = pathComponents[len(pathComponents)-1]
	}

	if _, isServiceFile := serviceFiles[entryName]; isServiceFile {
		return true
	}

	for _, pattern := range ignorePatterns {
		if strings.HasPrefix(pattern, ExclusionPrefix) {
			exclusionName := strings.TrimPrefix(pattern, ExclusionPrefix)
			if len(pathComponents) >= 1 && pathComponents[0] == exclusionName {
				return true
			}
			continue
		}

		isDirectoryPattern := strings.HasSuffix(pattern, "/")
		cleanedPattern := strings.TrimSuffix(pattern, "/")

		if !strings.Contains(cleanedPattern, "/") {
			isMatched, _ := filepath.Match(cleanedPattern, entryName)
			if isMatched {
				return true
			}
		} else {
			isMatched, _ := filepath.Match(pattern, normalizedPath)
			if isMatched {
				return true
			}
			if isDirectoryPattern && (normalizedPath == cleanedPattern || strings.HasPrefix(normalizedPath, cleanedPattern+"/")) {
				return true
			}
		}
	}

	return false
}

// ShouldDisplayBinaryContentByPath checks if a path should reveal binary content based on binary content patterns.
func ShouldDisplayBinaryContentByPath(relativePath string, binaryContentPatterns []string) bool {
	normalizedPath := filepath.ToSlash(relativePath)
	for _, patternValue := range binaryContentPatterns {
		trimmedPattern := strings.TrimSuffix(patternValue, "/")
		if strings.HasSuffix(patternValue, "/") {
			if normalizedPath == trimmedPattern || strings.HasPrefix(normalizedPath, trimmedPattern+"/") {
				return true
			}
			continue
		}
		isMatched, _ := filepath.Match(patternValue, normalizedPath)
		if isMatched {
			return true
		}
	}
	return false
}
