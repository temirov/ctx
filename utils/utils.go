package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// IsDirectory returns true if the given path exists and is a directory.
func IsDirectory(path string) bool {
	fileInfo, errorValue := os.Stat(path)
	if errorValue != nil {
		return false
	}
	return fileInfo.IsDir()
}

// ShouldIgnore returns true if the given entry (file or directory) should be ignored,
// based on the ignorePatterns. For tree output, if isRoot is true, directories whose names
// match an ignore pattern (or a marked exclusion) are ignored.
func ShouldIgnore(directoryEntry os.DirEntry, fullPath string, ignorePatterns []string, isRoot bool) bool {
	entryName := directoryEntry.Name()
	for _, pattern := range ignorePatterns {
		if strings.HasPrefix(pattern, "EXCL:") {
			exclusionValue := strings.TrimPrefix(pattern, "EXCL:")
			if isRoot && directoryEntry.IsDir() && entryName == exclusionValue {
				return true
			}
			continue
		}
		if strings.HasSuffix(pattern, "/") {
			patternDirectory := strings.TrimSuffix(pattern, "/")
			if isRoot && directoryEntry.IsDir() && entryName == patternDirectory {
				return true
			}
		} else {
			matched, matchError := filepath.Match(pattern, entryName)
			if matchError == nil && matched {
				return true
			}
		}
	}
	return false
}

// ShouldIgnoreByPath returns true if the entry (file or directory) should be ignored,
// based on its relative path (normalized with forward slashes) and the ignorePatterns.
func ShouldIgnoreByPath(relativePath string, directoryEntry os.DirEntry, ignorePatterns []string) bool {
	normalizedPath := filepath.ToSlash(relativePath)
	for _, pattern := range ignorePatterns {
		if strings.HasPrefix(pattern, "EXCL:") {
			exclusionValue := strings.TrimPrefix(pattern, "EXCL:")
			pathComponents := strings.Split(normalizedPath, "/")
			if len(pathComponents) >= 1 && pathComponents[0] == exclusionValue {
				// For the -e flag exclusion we only ignore if the entry is either the direct folder (1 part)
				// or a file immediately within that folder (2 parts). Nested directories or files beyond that
				// should not be ignored.
				if len(pathComponents) == 1 || len(pathComponents) == 2 {
					return true
				}
			}
		} else if strings.HasSuffix(pattern, "/") {
			patternDirectory := strings.TrimSuffix(pattern, "/")
			pathComponents := strings.Split(normalizedPath, "/")
			if len(pathComponents) > 0 && pathComponents[0] == patternDirectory {
				return true
			}
		} else {
			matched, matchError := filepath.Match(pattern, normalizedPath)
			if matchError == nil && matched {
				return true
			}
		}
	}
	return false
}
