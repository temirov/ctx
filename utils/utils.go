// Package utils contains helper functions for ignoring paths based on patterns.
package utils

import (
	"os"
	"path/filepath"
	"strings"
)

var serviceFiles = map[string]struct{}{
	".gitignore": {},
	".ignore":    {},
}

// ShouldIgnore checks if a directory entry should be ignored based on name patterns and defaults.
// It is primarily used during the tree building process.
func ShouldIgnore(directoryEntry os.DirEntry, ignorePatterns []string, isRoot bool) bool {
	entryName := directoryEntry.Name()

	if _, exists := serviceFiles[entryName]; exists {
		return true
	}

	for _, patternValue := range ignorePatterns {
		if strings.HasPrefix(patternValue, "EXCL:") {
			exclusionValue := strings.TrimPrefix(patternValue, "EXCL:")
			if isRoot && directoryEntry.IsDir() && entryName == exclusionValue {
				return true
			}
			continue
		}

		if strings.HasSuffix(patternValue, "/") {
			patternDirectory := strings.TrimSuffix(patternValue, "/")
			if directoryEntry.IsDir() {
				if isRoot && entryName == patternDirectory {
					return true
				}
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
// It considers EXCL: patterns, directory patterns (ending in /), and filename patterns.
// It also includes default ignores for .gitignore and .ignore files.
// Used by the content command's walk function.
func ShouldIgnoreByPath(relativePath string, ignorePatterns []string) bool {
	normalizedPath := filepath.ToSlash(relativePath)
	pathComponents := strings.Split(normalizedPath, "/")
	fileName := ""
	if len(pathComponents) > 0 {
		fileName = pathComponents[len(pathComponents)-1]
	}

	if _, exists := serviceFiles[fileName]; exists {
		return true
	}

	for _, pattern := range ignorePatterns {
		if strings.HasPrefix(pattern, "EXCL:") {
			exclusionValue := strings.TrimPrefix(pattern, "EXCL:")
			if len(pathComponents) >= 1 && pathComponents[0] == exclusionValue {
				return true
			}
			continue
		}

		isDirPattern := strings.HasSuffix(pattern, "/")
		cleanedPattern := strings.TrimSuffix(pattern, "/")

		if !strings.Contains(cleanedPattern, "/") {
			match, _ := filepath.Match(cleanedPattern, fileName)
			if match {
				return true
			}
		} else {
			if isDirPattern {
				if strings.HasPrefix(normalizedPath, cleanedPattern+"/") || normalizedPath == cleanedPattern {
					return true
				}
			} else {
				match, _ := filepath.Match(pattern, normalizedPath)
				if match {
					return true
				}
			}
		}
	}
	return false
}
