package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tyemirov/ctx/internal/docs/webdoc"
	"github.com/tyemirov/ctx/internal/services/clipboard"
)

type docWebFetcher interface {
	Fetch(ctx context.Context, path string, depth int) ([]webdoc.Page, error)
}

type docWebCommandOptions struct {
	Path             string
	Depth            int
	ClipboardEnabled bool
	CopyOnly         bool
	Clipboard        clipboard.Copier
	Writer           io.Writer
	Fetcher          docWebFetcher
}

func runDocWebCommand(ctx context.Context, options docWebCommandOptions) error {
	outputWriter := options.Writer
	if outputWriter == nil {
		outputWriter = os.Stdout
	}
	targetPath := strings.TrimSpace(options.Path)
	if targetPath == "" {
		return fmt.Errorf("doc command requires a non-empty --%s value for web URLs", repositoryPathFlagName)
	}
	if options.Depth < webdoc.MinDepth {
		return fmt.Errorf("depth must be >= %d", webdoc.MinDepth)
	}
	if options.Depth > webdoc.MaxDepth {
		return fmt.Errorf("depth must be <= %d", webdoc.MaxDepth)
	}
	fetcher := options.Fetcher
	if fetcher == nil {
		fetcher = webdoc.NewFetcher(nil)
	}
	pages, fetchErr := fetcher.Fetch(ctx, targetPath, options.Depth)
	if fetchErr != nil {
		return fetchErr
	}
	rendered := renderDocWebOutput(targetPath, options.Depth, pages)
	copyRequested := options.ClipboardEnabled || options.CopyOnly
	var clipboardBuffer *bytes.Buffer
	if copyRequested {
		if options.Clipboard == nil {
			return errors.New(clipboardServiceMissingMessage)
		}
		clipboardBuffer = &bytes.Buffer{}
		if options.CopyOnly {
			outputWriter = clipboardBuffer
		} else {
			outputWriter = io.MultiWriter(outputWriter, clipboardBuffer)
		}
	}
	if _, writeErr := fmt.Fprint(outputWriter, rendered); writeErr != nil {
		return writeErr
	}
	if !strings.HasSuffix(rendered, "\n") {
		if _, writeErr := fmt.Fprintln(outputWriter); writeErr != nil {
			return writeErr
		}
	}
	if copyRequested && clipboardBuffer != nil {
		if copyErr := options.Clipboard.Copy(clipboardBuffer.String()); copyErr != nil {
			return fmt.Errorf(clipboardCopyErrorFormat, copyErr)
		}
	}
	return nil
}

func renderDocWebOutput(path string, depth int, pages []webdoc.Page) string {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "# Web documentation for %s (depth %d)\n\n", path, depth)
	if len(pages) == 0 {
		builder.WriteString("_No documents were collected._\n")
		return builder.String()
	}
	for _, page := range pages {
		title := strings.TrimSpace(page.Title)
		if title == "" {
			title = page.URL
		}
		fmt.Fprintf(builder, "## %s\n\n", title)
		fmt.Fprintf(builder, "Source: %s\n\n", page.URL)
		trimmedContent := strings.TrimSpace(page.Content)
		if trimmedContent != "" {
			builder.WriteString(trimmedContent)
			builder.WriteString("\n\n")
		}
	}
	return builder.String()
}
