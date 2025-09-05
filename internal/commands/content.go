package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

const (
	// WarningAccessPathFormat is used when a path cannot be accessed during traversal.
	WarningAccessPathFormat = "Warning: error accessing path %s: %v\n"
	// WarningFileReadFormat is used when a file cannot be read.
	WarningFileReadFormat = "Warning: failed to read file %s: %v\n"
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
			fmt.Fprintf(os.Stderr, WarningAccessPathFormat, walkedPath, accessError)
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
			fmt.Fprintf(os.Stderr, WarningFileReadFormat, walkedPath, fileReadError)
			return nil
		}

		fileType := types.NodeTypeFile
		fileContent := string(fileBytes)
		mimeType := utils.DetectMimeType(walkedPath)
		if utils.IsBinary(fileBytes) {
			fileType = types.NodeTypeBinary
			if utils.ShouldDisplayBinaryContentByPath(relativePath, binaryContentPatterns) {
				fileContent = base64.StdEncoding.EncodeToString(fileBytes)
			} else {
				fileContent = utils.EmptyString
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
