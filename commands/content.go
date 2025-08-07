package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/types"
	"github.com/temirov/ctx/utils"
)

// GetContentData returns FileOutput slices for the specified directory.
func GetContentData(rootPath string, ignorePatterns, binaryPatterns []string) ([]types.FileOutput, error) {
	absoluteRootPath, pathErr := filepath.Abs(rootPath)
	if pathErr != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootPath, pathErr)
	}
	cleanRootPath := filepath.Clean(absoluteRootPath)

	var fileOutputs []types.FileOutput

	walkError := filepath.WalkDir(cleanRootPath, func(currentPath string, entry os.DirEntry, accessErr error) error {
		if accessErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: error accessing path %s: %v\n", currentPath, accessErr)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath := utils.RelativePathOrSelf(currentPath, cleanRootPath)
		if relativePath == "." {
			return nil
		}
		if utils.ShouldIgnoreByPath(relativePath, ignorePatterns) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if utils.ShouldTreatAsBinaryByPath(relativePath, binaryPatterns) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			fileOutputs = append(fileOutputs, types.FileOutput{
				Path: currentPath,
				Type: types.NodeTypeBinary,
			})
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		info, statErr := os.Stat(currentPath)
		if statErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read file %s: %v\n", currentPath, statErr)
			return nil
		}
		if info.Mode().Perm()&0o444 == 0 {
			fmt.Fprintf(os.Stderr, "Warning: failed to read file %s: permission denied\n", currentPath)
			return nil
		}

		contentBytes, readErr := os.ReadFile(currentPath)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read file %s: %v\n", currentPath, readErr)
			return nil
		}

		fileOutputs = append(fileOutputs, types.FileOutput{
			Path:    currentPath,
			Type:    types.NodeTypeFile,
			Content: string(contentBytes),
		})
		return nil
	})
	if walkError != nil {
		return nil, walkError
	}

	return fileOutputs, nil
}
