package stream

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/temirov/ctx/internal/commands"
	"github.com/temirov/ctx/internal/tokenizer"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

type TreeOptions struct {
	Root           string
	IgnorePatterns []string
	TokenCounter   tokenizer.Counter
	TokenModel     string
}

type ContentOptions struct {
	Root           string
	IgnorePatterns []string
	BinaryContent  []string
	TokenCounter   tokenizer.Counter
	TokenModel     string
	IncludeSummary bool
}

type emitter struct {
	ctx     context.Context
	out     chan<- Event
	command string
}

func newEmitter(ctx context.Context, out chan<- Event, command string) *emitter {
	if ctx == nil {
		ctx = context.Background()
	}
	return &emitter{ctx: ctx, out: out, command: command}
}

func (e *emitter) send(event Event) error {
	if e.out == nil {
		return fmt.Errorf("stream: event channel is nil")
	}
	event.Version = SchemaVersion
	if event.Command == "" {
		event.Command = e.command
	}
	if event.EmittedAt.IsZero() {
		event.EmittedAt = time.Now().UTC()
	}
	select {
	case <-e.ctx.Done():
		return e.ctx.Err()
	case e.out <- event:
		return nil
	}
}

func (e *emitter) warn(path, message string) {
	trimmed := strings.TrimRight(message, "\n")
	if trimmed == "" {
		return
	}
	_ = e.send(Event{
		Kind:    EventKindWarning,
		Path:    path,
		Message: &LogEvent{Level: "warning", Message: trimmed},
	})
}

type summaryTracker struct {
	files  int
	bytes  int64
	tokens int
	model  string
}

func (tracker *summaryTracker) add(size int64, tokens int, model string) {
	tracker.files++
	tracker.bytes += size
	tracker.tokens += tokens
	if tracker.model == "" && model != "" && tokens > 0 {
		tracker.model = model
	}
}

func (tracker *summaryTracker) summary() *SummaryEvent {
	return &SummaryEvent{
		Files:  tracker.files,
		Bytes:  tracker.bytes,
		Tokens: tracker.tokens,
		Model:  tracker.model,
	}
}

type directoryStackEntry struct {
	node  *types.TreeOutputNode
	depth int
}

func StreamTree(ctx context.Context, opts TreeOptions, out chan<- Event) error {
	if opts.Root == "" {
		return fmt.Errorf("stream: tree root path is empty")
	}

	emitter := newEmitter(ctx, out, types.CommandTree)
	if err := emitter.send(Event{Kind: EventKindStart, Path: opts.Root}); err != nil {
		return err
	}

	tracker := &summaryTracker{}
	var stack []*directoryStackEntry
	var lastError error

	streamOptions := commands.TreeStreamOptions{
		Root:           opts.Root,
		IgnorePatterns: opts.IgnorePatterns,
		TokenCounter:   opts.TokenCounter,
		TokenModel:     opts.TokenModel,
		Warn: func(message string) {
			emitter.warn(opts.Root, message)
		},
	}

	handler := func(evt commands.TreeEvent) error {
		switch evt.Kind {
		case commands.TreeEventEnterDir:
			dir := evt.Directory
			if err := emitter.send(Event{
				Kind: EventKindDirectory,
				Path: dir.Path,
				Directory: &DirectoryEvent{
					Phase:        DirectoryEnter,
					Path:         dir.Path,
					Name:         dir.Name,
					Depth:        dir.Depth,
					LastModified: dir.LastModified,
				},
			}); err != nil {
				return err
			}

			node := &types.TreeOutputNode{
				Path:         dir.Path,
				Name:         dir.Name,
				Type:         types.NodeTypeDirectory,
				LastModified: dir.LastModified,
			}
			entry := &directoryStackEntry{node: node, depth: dir.Depth}
			stack = append(stack, entry)
			return nil
		case commands.TreeEventFile:
			file := evt.File
			tracker.add(file.SizeBytes, file.Tokens, file.Model)
			nodeType := types.NodeTypeFile
			if file.IsBinary {
				nodeType = types.NodeTypeBinary
			}
			if err := emitter.send(Event{
				Kind: EventKindFile,
				Path: file.Path,
				File: &FileEvent{
					Path:         file.Path,
					Name:         file.Name,
					Depth:        file.Depth,
					SizeBytes:    file.SizeBytes,
					LastModified: file.LastModified,
					MimeType:     file.MimeType,
					IsBinary:     file.IsBinary,
					Tokens:       file.Tokens,
					Model:        file.Model,
					Type:         nodeType,
				},
			}); err != nil {
				return err
			}

			fileNode := &types.TreeOutputNode{
				Path:         file.Path,
				Name:         file.Name,
				Type:         nodeType,
				Size:         utils.FormatFileSize(file.SizeBytes),
				SizeBytes:    file.SizeBytes,
				LastModified: file.LastModified,
				MimeType:     file.MimeType,
				Tokens:       file.Tokens,
				Model:        file.Model,
			}
			if len(stack) == 0 {
				// standalone file input
				if err := emitter.send(Event{Kind: EventKindTree, Path: file.Path, Tree: fileNode}); err != nil {
					return err
				}
				return nil
			}
			parent := stack[len(stack)-1]
			parent.node.Children = append(parent.node.Children, fileNode)
			parent.node.TotalFiles += 1
			parent.node.TotalTokens += file.Tokens
			parent.node.SizeBytes += file.SizeBytes
			parent.node.TotalSize = utils.FormatFileSize(parent.node.SizeBytes)
			return nil
		case commands.TreeEventLeaveDir:
			dir := evt.Directory
			summary := dir.Summary
			summaryEvent := &SummaryEvent{Files: summary.Files, Bytes: summary.Bytes, Tokens: summary.Tokens}
			if summaryEvent.Tokens > 0 && opts.TokenModel != "" {
				summaryEvent.Model = opts.TokenModel
			}
			if err := emitter.send(Event{
				Kind: EventKindDirectory,
				Path: dir.Path,
				Directory: &DirectoryEvent{
					Phase:        DirectoryLeave,
					Path:         dir.Path,
					Name:         dir.Name,
					Depth:        dir.Depth,
					LastModified: dir.LastModified,
					Summary:      summaryEvent,
				},
			}); err != nil {
				return err
			}

			if len(stack) == 0 {
				return nil
			}
			entry := stack[len(stack)-1]
			if entry.depth != dir.Depth || entry.node.Path != dir.Path {
				return fmt.Errorf("stream: directory stack mismatch for %s", dir.Path)
			}
			entry.node.TotalFiles = summary.Files
			entry.node.TotalTokens = summary.Tokens
			entry.node.TotalSize = utils.FormatFileSize(summary.Bytes)
			entry.node.SizeBytes = summary.Bytes
			if summary.Tokens > 0 && opts.TokenModel != "" {
				entry.node.Model = opts.TokenModel
			}
			stack = stack[:len(stack)-1]

			if len(stack) == 0 {
				if err := emitter.send(Event{Kind: EventKindTree, Path: entry.node.Path, Tree: entry.node}); err != nil {
					return err
				}
			} else {
				parent := stack[len(stack)-1]
				parent.node.Children = append(parent.node.Children, entry.node)
				parent.node.TotalFiles += entry.node.TotalFiles
				parent.node.TotalTokens += entry.node.TotalTokens
				parent.node.SizeBytes += entry.node.SizeBytes
				parent.node.TotalSize = utils.FormatFileSize(parent.node.SizeBytes)
			}
			return nil
		default:
			return nil
		}
	}

	if err := commands.StreamTree(streamOptions, handler); err != nil {
		lastError = err
		emitter.warn(opts.Root, err.Error())
	}

	if lastError != nil {
		_ = emitter.send(Event{Kind: EventKindError, Path: opts.Root, Err: &ErrorEvent{Message: lastError.Error()}})
		return lastError
	}

	if err := emitter.send(Event{Kind: EventKindSummary, Path: opts.Root, Summary: tracker.summary()}); err != nil {
		return err
	}
	return emitter.send(Event{Kind: EventKindDone, Path: opts.Root})
}

func StreamContent(ctx context.Context, opts ContentOptions, out chan<- Event) error {
	if opts.Root == "" {
		return fmt.Errorf("stream: content root path is empty")
	}

	emitter := newEmitter(ctx, out, types.CommandContent)
	if err := emitter.send(Event{Kind: EventKindStart, Path: opts.Root}); err != nil {
		return err
	}

	tracker := &summaryTracker{}
	var collected []types.FileOutput
	var streamErr error

	if info, statErr := os.Stat(opts.Root); statErr == nil && !info.IsDir() {
		if err := emitSingleFileContent(emitter, tracker, opts, info); err != nil {
			emitter.warn(opts.Root, err.Error())
			emitter.send(Event{Kind: EventKindError, Path: opts.Root, Err: &ErrorEvent{Message: err.Error()}})
			return err
		}
		if err := emitter.send(Event{Kind: EventKindSummary, Path: opts.Root, Summary: tracker.summary()}); err != nil {
			return err
		}
		return emitter.send(Event{Kind: EventKindDone, Path: opts.Root})
	}

	visit := func(fileOut types.FileOutput) error {
		copied := fileOut
		collected = append(collected, copied)
		tracker.add(copied.SizeBytes, copied.Tokens, copied.Model)

		depth := directoryDepth(opts.Root, copied.Path)
		isBinary := copied.Type == types.NodeTypeBinary
		if err := emitter.send(Event{
			Kind: EventKindFile,
			Path: copied.Path,
			File: &FileEvent{
				Path:          copied.Path,
				Name:          filepath.Base(copied.Path),
				Depth:         depth,
				SizeBytes:     copied.SizeBytes,
				LastModified:  copied.LastModified,
				MimeType:      copied.MimeType,
				IsBinary:      isBinary,
				Tokens:        copied.Tokens,
				Model:         copied.Model,
				Type:          copied.Type,
				Documentation: copied.Documentation,
			},
		}); err != nil {
			return err
		}

		encoding := "utf-8"
		if isBinary {
			if copied.Content == "" {
				encoding = ""
			} else {
				encoding = "base64"
			}
		}
		if err := emitter.send(Event{
			Kind: EventKindContentChunk,
			Path: copied.Path,
			Chunk: &ChunkEvent{
				Path:     copied.Path,
				Index:    0,
				Data:     copied.Content,
				Encoding: encoding,
				IsFinal:  true,
			},
		}); err != nil {
			return err
		}
		return nil
	}

	streamErr = commands.StreamContent(opts.Root, opts.IgnorePatterns, opts.BinaryContent, opts.TokenCounter, opts.TokenModel, visit)
	if streamErr != nil {
		emitter.warn(opts.Root, streamErr.Error())
		_ = emitter.send(Event{Kind: EventKindError, Path: opts.Root, Err: &ErrorEvent{Message: streamErr.Error()}})
		return streamErr
	}

	if len(collected) > 0 {
		tree, err := commands.BuildContentTree(opts.Root, collected, opts.IncludeSummary, opts.TokenModel)
		if err != nil {
			emitter.warn(opts.Root, err.Error())
		} else if tree != nil {
			if err := emitter.send(Event{Kind: EventKindTree, Path: tree.Path, Tree: tree}); err != nil {
				return err
			}
		}
	}

	if err := emitter.send(Event{Kind: EventKindSummary, Path: opts.Root, Summary: tracker.summary()}); err != nil {
		return err
	}
	return emitter.send(Event{Kind: EventKindDone, Path: opts.Root})
}

func directoryDepth(root, path string) int {
	relative := utils.RelativePathOrSelf(path, root)
	if relative == "." {
		return 0
	}
	separators := string(filepath.Separator)
	return strings.Count(relative, separators)
}

func emitSingleFileContent(emitter *emitter, tracker *summaryTracker, opts ContentOptions, info os.FileInfo) error {
	fileBytes, readErr := os.ReadFile(opts.Root)
	if readErr != nil {
		return readErr
	}
	mimeType := utils.DetectMimeType(opts.Root)
	isBinary := utils.IsBinary(fileBytes)
	fileType := types.NodeTypeFile
	content := string(fileBytes)
	encoding := "utf-8"
	if isBinary {
		fileType = types.NodeTypeBinary
		relative := filepath.Base(opts.Root)
		if utils.ShouldDisplayBinaryContentByPath(relative, opts.BinaryContent) {
			content = base64.StdEncoding.EncodeToString(fileBytes)
			encoding = "base64"
		} else {
			content = ""
			encoding = ""
		}
	}
	var tokens int
	if opts.TokenCounter != nil && fileType != types.NodeTypeBinary {
		countResult, tokenErr := tokenizer.CountBytes(opts.TokenCounter, fileBytes)
		if tokenErr != nil {
			emitter.warn(opts.Root, tokenErr.Error())
		} else if countResult.Counted {
			tokens = countResult.Tokens
		}
	}
	tracker.add(info.Size(), tokens, opts.TokenModel)
	fileEvent := &FileEvent{
		Path:         opts.Root,
		Name:         filepath.Base(opts.Root),
		Depth:        0,
		SizeBytes:    info.Size(),
		LastModified: utils.FormatTimestamp(info.ModTime()),
		MimeType:     mimeType,
		IsBinary:     isBinary,
		Tokens:       tokens,
		Model:        opts.TokenModel,
		Type:         fileType,
	}
	if err := emitter.send(Event{Kind: EventKindFile, Path: opts.Root, File: fileEvent}); err != nil {
		return err
	}
	chunk := &ChunkEvent{
		Path:     opts.Root,
		Index:    0,
		Data:     content,
		Encoding: encoding,
		IsFinal:  true,
	}
	if err := emitter.send(Event{Kind: EventKindContentChunk, Path: opts.Root, Chunk: chunk}); err != nil {
		return err
	}
	fileOutput := types.FileOutput{
		Path:         opts.Root,
		Type:         fileType,
		Content:      content,
		Size:         utils.FormatFileSize(info.Size()),
		SizeBytes:    info.Size(),
		LastModified: utils.FormatTimestamp(info.ModTime()),
		MimeType:     mimeType,
		Tokens:       tokens,
	}
	if tokens > 0 && opts.TokenModel != "" {
		fileOutput.Model = opts.TokenModel
	}
	if tree, err := commands.BuildContentTree(opts.Root, []types.FileOutput{fileOutput}, opts.IncludeSummary, opts.TokenModel); err == nil && tree != nil {
		if err := emitter.send(Event{Kind: EventKindTree, Path: tree.Path, Tree: tree}); err != nil {
			return err
		}
	}
	return nil
}
