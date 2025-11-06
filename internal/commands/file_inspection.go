package commands

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/tyemirov/ctx/internal/tokenizer"
	"github.com/tyemirov/ctx/internal/utils"
)

type fileInspectionConfig struct {
	RelativePath             string
	IncludeContent           bool
	BinaryPatterns           []string
	TokenCounter             tokenizer.Counter
	TokenModel               string
	Warn                     func(string)
	AllowBinaryTokenCounting bool
}

type fileInspectionResult struct {
	IsBinary        bool
	MimeType        string
	Content         string
	ContentEncoding string
	Tokens          int
	Model           string
}

func inspectFile(path string, info os.FileInfo, config fileInspectionConfig) (fileInspectionResult, bool) {
	warn := config.Warn
	if warn == nil {
		warn = func(string) {}
	}

	result := fileInspectionResult{
		MimeType: utils.DetectMimeType(path),
	}

	var data []byte
	if config.IncludeContent {
		fileBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			warn(fmt.Sprintf(WarningFileReadFormat, path, readErr))
			return fileInspectionResult{}, false
		}
		data = fileBytes
		if utils.IsBinary(fileBytes) {
			result.IsBinary = true
			if utils.ShouldDisplayBinaryContentByPath(config.RelativePath, config.BinaryPatterns) {
				result.Content = base64.StdEncoding.EncodeToString(fileBytes)
				result.ContentEncoding = "base64"
			}
		} else {
			result.Content = string(fileBytes)
			result.ContentEncoding = "utf-8"
		}
	} else {
		result.IsBinary = utils.IsFileBinary(path)
	}

	if result.IsBinary && result.ContentEncoding == "" {
		result.Content = ""
	}

	if config.TokenCounter != nil {
		if result.IsBinary && !config.AllowBinaryTokenCounting {
			return result, true
		}
		var countResult tokenizer.CountResult
		var tokenErr error
		if len(data) > 0 {
			countResult, tokenErr = tokenizer.CountBytes(config.TokenCounter, data)
		} else {
			countResult, tokenErr = tokenizer.CountFile(config.TokenCounter, path)
		}
		if tokenErr != nil {
			warn(fmt.Sprintf(WarningTokenCountFormat, path, tokenErr))
		} else if countResult.Counted {
			result.Tokens = countResult.Tokens
			if result.Tokens > 0 && config.TokenModel != "" {
				result.Model = config.TokenModel
			}
		}
	}

	return result, true
}
