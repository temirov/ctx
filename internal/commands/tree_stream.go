package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tyemirov/ctx/internal/tokenizer"
	"github.com/tyemirov/ctx/internal/utils"
)

type TreeEventKind int

const (
	TreeEventEnterDir TreeEventKind = iota
	TreeEventFile
	TreeEventLeaveDir
)

type TreeSummary struct {
	Files  int
	Bytes  int64
	Tokens int
}

type TreeDirectoryEvent struct {
	Path         string
	Name         string
	Depth        int
	LastModified string
	Summary      TreeSummary
}

type TreeFileEvent struct {
	Path            string
	Name            string
	Depth           int
	SizeBytes       int64
	LastModified    string
	MimeType        string
	IsBinary        bool
	Tokens          int
	Model           string
	Content         string
	ContentEncoding string
}

type TreeEvent struct {
	Kind      TreeEventKind
	Directory *TreeDirectoryEvent
	File      *TreeFileEvent
}

type TreeStreamOptions struct {
	Root                  string
	IgnorePatterns        []string
	TokenCounter          tokenizer.Counter
	TokenModel            string
	Warn                  func(message string)
	BinaryContentPatterns []string
	IncludeContent        bool
}

type treeStreamContext struct {
	options TreeStreamOptions
	handler func(TreeEvent) error
}

func StreamTree(options TreeStreamOptions, handler func(TreeEvent) error) error {
	if handler == nil {
		return fmt.Errorf("tree stream handler is nil")
	}

	ctx := treeStreamContext{options: options, handler: handler}
	if ctx.options.Warn == nil {
		ctx.options.Warn = func(string) {}
	}

	info, statErr := os.Stat(options.Root)
	if statErr != nil {
		return statErr
	}

	if info.IsDir() {
		_, err := ctx.walkDirectory(options.Root, options.Root, 0)
		return err
	}

	relative := filepath.Base(options.Root)
	_, err := ctx.emitFile(options.Root, relative, info, 0)
	return err
}

func (ctx *treeStreamContext) walkDirectory(path string, root string, depth int) (TreeSummary, error) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		return TreeSummary{}, statErr
	}

	enterEvent := TreeDirectoryEvent{
		Path:         path,
		Name:         filepath.Base(path),
		Depth:        depth,
		LastModified: utils.FormatTimestamp(info.ModTime()),
	}
	if err := ctx.handler(TreeEvent{Kind: TreeEventEnterDir, Directory: &enterEvent}); err != nil {
		return TreeSummary{}, err
	}

	entries, readErr := os.ReadDir(path)
	if readErr != nil {
		ctx.options.Warn(fmt.Sprintf("reading directory %s: %v\n", path, readErr))
		return TreeSummary{}, readErr
	}

	summary := TreeSummary{}

	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		relativePath := utils.RelativePathOrSelf(childPath, root)
		if utils.ShouldIgnoreByPath(relativePath, ctx.options.IgnorePatterns) {
			continue
		}

		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			ctx.options.Warn(fmt.Sprintf("Warning: unable to stat %s: %v\n", childPath, infoErr))
			continue
		}

		if entry.IsDir() {
			childSummary, err := ctx.walkDirectory(childPath, root, depth+1)
			if err != nil {
				ctx.options.Warn(fmt.Sprintf("Warning: Skipping subdirectory %s due to error: %v\n", childPath, err))
				continue
			}
			summary.Files += childSummary.Files
			summary.Bytes += childSummary.Bytes
			summary.Tokens += childSummary.Tokens
			continue
		}

		fileSummary, err := ctx.emitFile(childPath, relativePath, entryInfo, depth+1)
		if err != nil {
			return TreeSummary{}, err
		}
		summary.Files += fileSummary.Files
		summary.Bytes += fileSummary.Bytes
		summary.Tokens += fileSummary.Tokens
	}

	leaveEvent := enterEvent
	leaveEvent.Summary = summary
	if err := ctx.handler(TreeEvent{Kind: TreeEventLeaveDir, Directory: &leaveEvent}); err != nil {
		return TreeSummary{}, err
	}

	return summary, nil
}

func (ctx *treeStreamContext) emitFile(path string, relativePath string, info os.FileInfo, depth int) (TreeSummary, error) {
	mimeType := utils.DetectMimeType(path)
	isBinary := false
	var fileBytes []byte
	var fileContent string
	var contentEncoding string

	if ctx.options.IncludeContent {
		bytesRead, readError := os.ReadFile(path)
		if readError != nil {
			ctx.options.Warn(fmt.Sprintf(WarningFileReadFormat, path, readError))
		} else {
			fileBytes = bytesRead
			isBinary = utils.IsBinary(fileBytes)
		}
	}
	if !ctx.options.IncludeContent || fileBytes == nil {
		isBinary = utils.IsFileBinary(path)
	}
	if ctx.options.IncludeContent && fileBytes == nil {
		return TreeSummary{}, nil
	}

	if ctx.options.IncludeContent && fileBytes != nil {
		if isBinary {
			if utils.ShouldDisplayBinaryContentByPath(relativePath, ctx.options.BinaryContentPatterns) {
				fileContent = base64.StdEncoding.EncodeToString(fileBytes)
				contentEncoding = "base64"
			} else {
				fileContent = ""
				contentEncoding = ""
			}
		} else {
			fileContent = string(fileBytes)
			contentEncoding = "utf-8"
		}
	}

	tokens := 0
	if ctx.options.TokenCounter != nil && (!isBinary || ctx.options.IncludeContent) {
		var result tokenizer.CountResult
		var tokenErr error
		if ctx.options.IncludeContent && fileBytes != nil {
			result, tokenErr = tokenizer.CountBytes(ctx.options.TokenCounter, fileBytes)
		} else {
			result, tokenErr = tokenizer.CountFile(ctx.options.TokenCounter, path)
		}
		if tokenErr != nil {
			ctx.options.Warn(fmt.Sprintf("Warning: failed to count tokens for %s: %v\n", path, tokenErr))
		} else if result.Counted {
			tokens = result.Tokens
		}
	}

	fileEvent := TreeFileEvent{
		Path:            path,
		Name:            filepath.Base(path),
		Depth:           depth,
		SizeBytes:       info.Size(),
		LastModified:    utils.FormatTimestamp(info.ModTime()),
		MimeType:        mimeType,
		IsBinary:        isBinary,
		Tokens:          tokens,
		Content:         fileContent,
		ContentEncoding: contentEncoding,
	}
	if tokens > 0 {
		fileEvent.Model = ctx.options.TokenModel
	}

	if err := ctx.handler(TreeEvent{Kind: TreeEventFile, File: &fileEvent}); err != nil {
		return TreeSummary{}, err
	}

	return TreeSummary{Files: 1, Bytes: info.Size(), Tokens: tokens}, nil
}
