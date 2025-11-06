package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tyemirov/ctx/internal/tokenizer"
	"github.com/tyemirov/ctx/internal/types"
	"github.com/tyemirov/ctx/internal/utils"
)

type ContentStreamOptions struct {
	Root                  string
	IgnorePatterns        []string
	BinaryContentPatterns []string
	TokenCounter          tokenizer.Counter
	TokenModel            string
	Warn                  func(string)
}

type ContentVisitor func(types.FileOutput) error

func StreamContent(options ContentStreamOptions, visitor ContentVisitor) error {
	absoluteRootPath, absolutePathError := filepath.Abs(options.Root)
	if absolutePathError != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", options.Root, absolutePathError)
	}

	warn := options.Warn
	if warn == nil {
		warn = func(string) {}
	}

	cleanedRootPath := filepath.Clean(absoluteRootPath)

	walkErr := filepath.WalkDir(cleanedRootPath, func(walkedPath string, directoryEntry os.DirEntry, accessError error) error {
		if accessError != nil {
			warn(fmt.Sprintf(WarningAccessPathFormat, walkedPath, accessError))
			if directoryEntry != nil && directoryEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath := utils.RelativePathOrSelf(walkedPath, cleanedRootPath)
		if relativePath == "." {
			return nil
		}
		if utils.ShouldIgnoreByPath(relativePath, options.IgnorePatterns) {
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
			warn(fmt.Sprintf(WarningAccessPathFormat, walkedPath, infoError))
			return nil
		}

		result, ok := inspectFile(walkedPath, fileInfo, fileInspectionConfig{
			RelativePath:             relativePath,
			IncludeContent:           true,
			BinaryPatterns:           options.BinaryContentPatterns,
			TokenCounter:             options.TokenCounter,
			TokenModel:               options.TokenModel,
			Warn:                     warn,
			AllowBinaryTokenCounting: false,
		})
		if !ok {
			return nil
		}

		nodeType := types.NodeTypeFile
		content := result.Content
		if result.IsBinary {
			nodeType = types.NodeTypeBinary
			if result.ContentEncoding == "" {
				content = utils.EmptyString
			}
		}

		output := types.FileOutput{
			Path:         walkedPath,
			Type:         nodeType,
			Content:      content,
			Size:         utils.FormatFileSize(fileInfo.Size()),
			SizeBytes:    fileInfo.Size(),
			LastModified: utils.FormatTimestamp(fileInfo.ModTime()),
			MimeType:     result.MimeType,
			Tokens:       result.Tokens,
			Model:        result.Model,
		}
		if output.Tokens > 0 && output.Model == "" && options.TokenModel != "" {
			output.Model = options.TokenModel
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
