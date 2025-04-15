package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/types"
	"github.com/temirov/ctx/utils"
)

// GetContentData traverses the directory tree starting from rootDirectory
// and collects FileOutput data for files that are not excluded.
// It prints warnings to stderr for unreadable files/dirs but attempts to continue.
// Returns a slice of successfully read file data and the first fatal error encountered during walk.
func GetContentData(rootDirectory string, ignorePatterns []string) ([]types.FileOutput, error) {
	var results []types.FileOutput
	var firstFatalError error

	absoluteRootDirectory, absErr := filepath.Abs(rootDirectory)
	if absErr != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootDirectory, absErr)
	}
	cleanRootDirectory := filepath.Clean(absoluteRootDirectory)

	walkErr := filepath.WalkDir(cleanRootDirectory, func(currentPath string, directoryEntry os.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error accessing path %s: %v\n", currentPath, err)
			if directoryEntry != nil && directoryEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath := utils.RelativePathOrSelf(currentPath, cleanRootDirectory)

		if relativePath == "." {
			return nil
		}

		if utils.ShouldIgnoreByPath(relativePath, ignorePatterns) {
			if directoryEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !directoryEntry.IsDir() {
			fileData, readErr := os.ReadFile(currentPath)
			if readErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", currentPath, readErr)
				return nil
			}

			absoluteEntryPath, _ := filepath.Abs(currentPath)
			results = append(results, types.FileOutput{
				Path:    absoluteEntryPath,
				Type:    types.NodeTypeFile,
				Content: string(fileData),
			})
		}

		return nil
	})

	if walkErr != nil && !errors.Is(walkErr, filepath.SkipDir) {
		firstFatalError = walkErr
	}

	return results, firstFatalError
}
