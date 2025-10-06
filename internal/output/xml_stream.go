package output

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

type xmlStreamRenderer struct {
	stdout              io.Writer
	stderr              io.Writer
	command             string
	totalRoots          int
	rootsEmitted        int
	headerWritten       bool
	wrapperOpened       bool
	currentRootPath     string
	currentRootCaptured bool
}

func NewXMLStreamRenderer(stdout, stderr io.Writer, command string, totalRoots int) StreamRenderer {
	if totalRoots < 1 {
		totalRoots = 1
	}
	return &xmlStreamRenderer{stdout: stdout, stderr: stderr, command: command, totalRoots: totalRoots}
}

func (renderer *xmlStreamRenderer) Handle(event stream.Event) error {
	switch event.Kind {
	case stream.EventKindWarning:
		if event.Message != nil && renderer.stderr != nil {
			fmt.Fprintln(renderer.stderr, event.Message.Message)
		}
	case stream.EventKindError:
		if event.Err != nil && renderer.stderr != nil {
			fmt.Fprintln(renderer.stderr, event.Err.Message)
		}
	case stream.EventKindStart:
		renderer.currentRootPath = normalizePath(event.Path)
		renderer.currentRootCaptured = false
	case stream.EventKindTree:
		return renderer.handleTree(event.Tree, event.Path)
	case stream.EventKindDone:
		renderer.currentRootPath = ""
		renderer.currentRootCaptured = false
	}
	return nil
}

func (renderer *xmlStreamRenderer) Flush() error {
	if renderer.stdout == nil {
		return nil
	}
	if renderer.wrapperOpened {
		if _, err := renderer.stdout.Write([]byte("</results>\n")); err != nil {
			return err
		}
		renderer.wrapperOpened = false
		return nil
	}
	if renderer.rootsEmitted > 0 {
		if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (renderer *xmlStreamRenderer) handleTree(node *types.TreeOutputNode, path string) error {
	if node == nil || renderer.stdout == nil {
		return nil
	}
	if renderer.currentRootCaptured {
		return nil
	}
	expected := renderer.currentRootPath
	if expected != "" {
		normalizedPath := normalizePath(path)
		if !pathsEqual(expected, normalizedPath) && !pathsEqual(expected, node.Path) {
			return nil
		}
	}
	if err := renderer.emitNode(node); err != nil {
		return err
	}
	renderer.currentRootCaptured = true
	return nil
}

func (renderer *xmlStreamRenderer) emitNode(node *types.TreeOutputNode) error {
	if renderer.totalRoots > 1 {
		if !renderer.headerWritten {
			if _, err := renderer.stdout.Write([]byte(xmlHeader)); err != nil {
				return err
			}
			renderer.headerWritten = true
		}
		if !renderer.wrapperOpened {
			if _, err := renderer.stdout.Write([]byte("<results>\n")); err != nil {
				return err
			}
			renderer.wrapperOpened = true
		}
		encoded, err := xml.MarshalIndent(node, indentSpacer, indentSpacer)
		if err != nil {
			return err
		}
		if _, err := renderer.stdout.Write(encoded); err != nil {
			return err
		}
		if _, err := renderer.stdout.Write([]byte("\n")); err != nil {
			return err
		}
	} else {
		if !renderer.headerWritten {
			if _, err := renderer.stdout.Write([]byte(xmlHeader)); err != nil {
				return err
			}
			renderer.headerWritten = true
		}
		encoded, err := xml.MarshalIndent(node, indentPrefix, indentSpacer)
		if err != nil {
			return err
		}
		if _, err := renderer.stdout.Write(encoded); err != nil {
			return err
		}
	}
	renderer.rootsEmitted++
	return nil
}
