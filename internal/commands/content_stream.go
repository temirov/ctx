package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/tokenizer"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

// ContentVisitor receives each FileOutput discovered during traversal.
type ContentVisitor func(types.FileOutput) error

// StreamContent walks rootPath and invokes visitor for every file that survives
// ignore filtering. It mirrors GetContentData but yields results incrementally.
func StreamContent(rootPath string, ignorePatterns []string, binaryContentPatterns []string, tokenCounter tokenizer.Counter, tokenModel string, visitor ContentVisitor) error {
	absoluteRootPath, absolutePathError := filepath.Abs(rootPath)
	if absolutePathError != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", rootPath, absolutePathError)
	}
	cleanedRootPath := filepath.Clean(absoluteRootPath)

	walkErr := filepath.WalkDir(cleanedRootPath, func(walkedPath string, directoryEntry os.DirEntry, accessError error) error {
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

		fileInfo, infoError := directoryEntry.Info()
		if infoError != nil {
			fmt.Fprintf(os.Stderr, WarningAccessPathFormat, walkedPath, infoError)
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

		var tokenCount int
		if tokenCounter != nil && fileType != types.NodeTypeBinary {
			countResult, tokenErr := tokenizer.CountBytes(tokenCounter, fileBytes)
			if tokenErr != nil {
				fmt.Fprintf(os.Stderr, WarningTokenCountFormat, walkedPath, tokenErr)
			} else if countResult.Counted {
				tokenCount = countResult.Tokens
			}
		}

		output := types.FileOutput{
			Path:         walkedPath,
			Type:         fileType,
			Content:      fileContent,
			Size:         utils.FormatFileSize(fileInfo.Size()),
			SizeBytes:    fileInfo.Size(),
			LastModified: utils.FormatTimestamp(fileInfo.ModTime()),
			MimeType:     mimeType,
			Tokens:       tokenCount,
		}
		if tokenCount > 0 {
			output.Model = tokenModel
		}

		if visitor != nil {
			if visitErr := visitor(output); visitErr != nil {
				return visitErr
			}
		}
		return nil
	})

	return walkErr
}
