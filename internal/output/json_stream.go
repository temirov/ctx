package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

type jsonStreamRenderer struct {
	stdout              io.Writer
	stderr              io.Writer
	command             string
	totalRoots          int
	rootsEmitted        int
	arrayOpened         bool
	currentRootPath     string
	currentRootCaptured bool
}

func NewJSONStreamRenderer(stdout, stderr io.Writer, command string, totalRoots int) StreamRenderer {
	if totalRoots < 1 {
		totalRoots = 1
	}
	return &jsonStreamRenderer{
		stdout:     stdout,
		stderr:     stderr,
		command:    command,
		totalRoots: totalRoots,
	}
}

func (renderer *jsonStreamRenderer) Handle(event stream.Event) error {
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

func (renderer *jsonStreamRenderer) Flush() error {
	if renderer.stdout == nil {
		return nil
	}
	if renderer.arrayOpened {
		if _, flushError := renderer.stdout.Write([]byte("]\n")); flushError != nil {
			return flushError
		}
		renderer.arrayOpened = false
		return nil
	}
	if renderer.rootsEmitted > 0 {
		if _, flushError := renderer.stdout.Write([]byte("\n")); flushError != nil {
			return flushError
		}
	}
	return nil
}

func (renderer *jsonStreamRenderer) handleTree(node *types.TreeOutputNode, path string) error {
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
	if emitError := renderer.emitNode(node); emitError != nil {
		return emitError
	}
	renderer.currentRootCaptured = true
	return nil
}

func (renderer *jsonStreamRenderer) emitNode(node *types.TreeOutputNode) error {
	var (
		encoded     []byte
		encodeError error
	)
	if renderer.totalRoots > 1 {
		encoded, encodeError = json.MarshalIndent(node, indentSpacer, indentSpacer)
	} else {
		encoded, encodeError = json.MarshalIndent(node, indentPrefix, indentSpacer)
	}
	if encodeError != nil {
		return encodeError
	}
	if renderer.totalRoots > 1 {
		if renderer.rootsEmitted == 0 && !renderer.arrayOpened {
			if _, writeError := renderer.stdout.Write([]byte("[\n")); writeError != nil {
				return writeError
			}
			renderer.arrayOpened = true
		} else {
			if _, writeError := renderer.stdout.Write([]byte(",\n")); writeError != nil {
				return writeError
			}
		}
	}
	if _, writeError := renderer.stdout.Write(encoded); writeError != nil {
		return writeError
	}
	renderer.rootsEmitted++
	if renderer.totalRoots > 1 && renderer.rootsEmitted == renderer.totalRoots {
		if _, writeError := renderer.stdout.Write([]byte("\n")); writeError != nil {
			return writeError
		}
	}
	return nil
}
