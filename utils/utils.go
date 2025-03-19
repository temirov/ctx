// Package utils contains helper functions for ignoring paths and checking directories.
package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// IsDirectory returns true if the given path exists and is a directory.
func IsDirectory(pathValue string) bool {
	fileInformation, statError := os.Stat(pathValue)
	if statError != nil {
		return false
	}
	return fileInformation.IsDir()
}

// ShouldIgnore is used by the tree command.
func ShouldIgnore(directoryEntry os.DirEntry, ignorePatterns []string, isRoot bool) bool {
	entryName := directoryEntry.Name()
	for _, patternValue := range ignorePatterns {
		// EXCL: means this is the -e special folder exclusion.
		if strings.HasPrefix(patternValue, "EXCL:") {
			exclusionValue := strings.TrimPrefix(patternValue, "EXCL:")
			if isRoot && directoryEntry.IsDir() && entryName == exclusionValue {
				return true
			}
			continue
		}
		if strings.HasSuffix(patternValue, "/") {
			patternDirectory := strings.TrimSuffix(patternValue, "/")
			if isRoot && directoryEntry.IsDir() && entryName == patternDirectory {
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

// ShouldIgnoreByPath is used by the content command.
func ShouldIgnoreByPath(relativePath string, ignorePatterns []string) bool {
	normalizedPath := filepath.ToSlash(relativePath)
	pathComponents := strings.Split(normalizedPath, "/")

	for _, patternValue := range ignorePatterns {
		if strings.HasPrefix(patternValue, "EXCL:") {
			exclusionValue := strings.TrimPrefix(patternValue, "EXCL:")
			if len(pathComponents) >= 1 && pathComponents[0] == exclusionValue {
				return true
			}
			continue
		}
		if strings.HasSuffix(patternValue, "/") {
			patternDirectory := strings.TrimSuffix(patternValue, "/")
			if len(pathComponents) > 0 && pathComponents[0] == patternDirectory {
				return true
			}
		} else {
			// If no slash in the pattern, match only the last component
			if !strings.Contains(patternValue, "/") {
				lastComponent := pathComponents[len(pathComponents)-1]
				isMatched, matchError := filepath.Match(patternValue, lastComponent)
				if matchError == nil && isMatched {
					return true
				}
			} else {
				// If pattern includes slash, match entire path
				isMatched, matchError := filepath.Match(patternValue, normalizedPath)
				if matchError == nil && isMatched {
					return true
				}
			}
		}
	}
	return false
}
