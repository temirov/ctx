package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

// GetContentData returns FileOutput slices for the specified directory.
func GetContentData(rootPath string, ignorePatterns []string, binaryContentPatterns []string) ([]types.FileOutput, error) {
	absoluteRootPath, absolutePathError := filepath.Abs(rootPath)
	if absolutePathError != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootPath, absolutePathError)
	}
	cleanedRootPath := filepath.Clean(absoluteRootPath)

	var fileOutputs []types.FileOutput

	directoryWalkError := filepath.WalkDir(cleanedRootPath, func(walkedPath string, directoryEntry os.DirEntry, accessError error) error {
		if accessError != nil {
			fmt.Fprintf(os.Stderr, "Warning: error accessing path %s: %v\n", walkedPath, accessError)
			if directoryEntry != nil && directoryEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath := utils.RelativePathOrSelf(walkedPath, cleanedRootPath)
		if relativePath == "." {
			return nil
		}
		if utils.ShouldIgnoreByPath(relativePath, ignorePatterns) {
			if directoryEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if directoryEntry.IsDir() {
			return nil
		}

		fileBytes, fileReadError := os.ReadFile(walkedPath)
		if fileReadError != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read file %s: %v\n", walkedPath, fileReadError)
			return nil
		}

		fileType := types.NodeTypeFile
		fileContent := string(fileBytes)
		mimeType := ""
		if utils.IsBinary(fileBytes) {
			fileType = types.NodeTypeBinary
			mimeType = utils.DetectMimeType(walkedPath)
			if utils.ShouldDisplayBinaryContentByPath(relativePath, binaryContentPatterns) {
				fileContent = base64.StdEncoding.EncodeToString(fileBytes)
			} else {
				fileContent = ""
			}
		}

		fileOutputs = append(fileOutputs, types.FileOutput{
			Path:     walkedPath,
			Type:     fileType,
			Content:  fileContent,
			MimeType: mimeType,
		})
		return nil
	})
	if directoryWalkError != nil {
		return nil, directoryWalkError
	}

	return fileOutputs, nil
}
