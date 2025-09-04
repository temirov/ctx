package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

// GetContentData returns FileOutput slices for the specified directory.
func GetContentData(rootPath string, ignorePatterns []string) ([]types.FileOutput, error) {
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
		if entry.IsDir() {
			return nil
		}

		contentBytes, readErr := os.ReadFile(currentPath)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read file %s: %v\n", currentPath, readErr)
			return nil
		}

		fileType := types.NodeTypeFile
		content := string(contentBytes)
		mimeType := ""
		if utils.IsBinary(contentBytes) {
			fileType = types.NodeTypeBinary
			content = ""
			mimeType = utils.DetectMimeType(currentPath)
		}

		fileOutputs = append(fileOutputs, types.FileOutput{
			Path:     currentPath,
			Type:     fileType,
			Content:  content,
			MimeType: mimeType,
		})
		return nil
	})
	if walkError != nil {
		return nil, walkError
	}

	return fileOutputs, nil
}
