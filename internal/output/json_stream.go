package output

import (
	"fmt"
	"io"

	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/types"
)

type jsonStreamRenderer struct {
	stdout              io.Writer
	stderr              io.Writer
	command             string
	roots               []*types.TreeOutputNode
	currentRootPath     string
	currentRootCaptured bool
}

func NewJSONStreamRenderer(stdout, stderr io.Writer, command string) StreamRenderer {
	return &jsonStreamRenderer{stdout: stdout, stderr: stderr, command: command}
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
		renderer.captureRoot(event.Tree, event.Path)
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

	items := make([]interface{}, 0, len(renderer.roots))
	for _, node := range renderer.roots {
		items = append(items, node)
	}

	encoded, err := RenderJSON(items)
	if err != nil {
		return err
	}

	if _, err := io.WriteString(renderer.stdout, encoded); err != nil {
		return err
	}

	renderer.roots = nil
	return nil
}

func (renderer *jsonStreamRenderer) captureRoot(node *types.TreeOutputNode, path string) {
	if node == nil {
		return
	}

	if renderer.currentRootPath != "" {
		if renderer.currentRootCaptured {
			return
		}
		if !pathsEqual(renderer.currentRootPath, path) && !pathsEqual(renderer.currentRootPath, node.Path) {
			return
		}
		renderer.roots = append(renderer.roots, cloneTreeNode(node))
		renderer.currentRootCaptured = true
		return
	}

	renderer.roots = append(renderer.roots, cloneTreeNode(node))
}
