package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	//nolint:depguard
	"github.com/temirov/content/types"
	//nolint:depguard
	"github.com/temirov/content/utils"
)

// GetContentData traverses the directory tree starting from rootDirectory
// and collects FileOutput data for files that are not excluded.
// It prints warnings to stderr for unreadable files/dirs but attempts to continue.
// Returns a slice of successfully read file data and the first fatal error encountered during walk.
func GetContentData(rootDirectory string, ignorePatterns []string) ([]types.FileOutput, error) {
	var results []types.FileOutput
	var firstFatalError error // To capture the first actual error stopping the walk

	absoluteRootDirectory, absErr := filepath.Abs(rootDirectory)
	if absErr != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootDirectory, absErr)
	}
	cleanRootDirectory := filepath.Clean(absoluteRootDirectory)

	walkErr := filepath.WalkDir(cleanRootDirectory, func(path string, entry os.DirEntry, err error) error {
		// Handle errors accessing the entry itself (permissions, etc.)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error accessing path %s: %v\n", path, err)
			// Try to skip the problematic entry if possible
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			// If it's a file or WalkDir had a more general error, just continue if possible
			return nil
		}

		// Skip the root directory itself
		relativePath := relativeOrSelf(path, cleanRootDirectory)
		if relativePath == "." {
			return nil
		}

		// Handle -e folder exclusion
		// Note: utils.ShouldIgnoreByPath also handles EXCL patterns for nested checks
		if entry.IsDir() && filepath.Dir(relativePath) == "." { // Check if top-level item
			for _, pattern := range ignorePatterns {
				if strings.HasPrefix(pattern, "EXCL:") {
					exclusionName := strings.TrimPrefix(pattern, "EXCL:")
					if relativePath == exclusionName {
						return filepath.SkipDir
					}
				}
			}
		}

		// Handle .ignore/.gitignore patterns
		if utils.ShouldIgnoreByPath(relativePath, ignorePatterns) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil // Skip ignored file
		}

		// Process files: read content and add to results
		if !entry.IsDir() {
			fileData, readErr := os.ReadFile(path)
			if readErr != nil {
				// Warn about read errors but continue the walk
				fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", path, readErr)
				return nil // Don't stop the walk for one unreadable file
			}

			// Ensure absolute path in the result
			absPath, _ := filepath.Abs(path)
			results = append(results, types.FileOutput{
				Path:    absPath,
				Type:    "file",
				Content: string(fileData),
			})
		}

		return nil // Continue walking
	})

	// Capture the first fatal error from WalkDir itself
	if walkErr != nil {
		firstFatalError = walkErr
	}

	return results, firstFatalError
}

// relativeOrSelf calculates the relative path from root to fullPath.
// Returns fullPath if relative calculation fails.
func relativeOrSelf(fullPath, root string) string {
	cleanPath := filepath.Clean(fullPath)
	// Ensure root is absolute for reliable relative path calculation
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return cleanPath // Fallback if root path is invalid
	}
	relativePath, relErr := filepath.Rel(absRoot, cleanPath)
	if relErr != nil {
		return cleanPath // Fallback if relative fails
	}
	return relativePath
}
