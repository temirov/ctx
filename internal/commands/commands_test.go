package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/temirov/ctx/internal/commands"
	"github.com/temirov/ctx/internal/types"
)

const (
	textFileName      = "plain.txt"
	textFileContent   = "hello"
	binaryFileName    = "data.bin"
	binaryFileContent = "\x00\xff"
	ignoredFileName   = "ignored.txt"
	directoryName     = "dir"
	ignoredDirName    = "skip"
)

// TestGetContentData verifies file data collection respecting ignore patterns and binary detection.
func TestGetContentData(testingHandle *testing.T) {
	rootDirectory := testingHandle.TempDir()
	textFilePath := filepath.Join(rootDirectory, textFileName)
	binaryFilePath := filepath.Join(rootDirectory, binaryFileName)
	ignoredFilePath := filepath.Join(rootDirectory, ignoredFileName)
	writeError := os.WriteFile(textFilePath, []byte(textFileContent), 0o644)
	if writeError != nil {
		testingHandle.Fatalf("writing text file: %v", writeError)
	}
	writeError = os.WriteFile(binaryFilePath, []byte(binaryFileContent), 0o644)
	if writeError != nil {
		testingHandle.Fatalf("writing binary file: %v", writeError)
	}
	writeError = os.WriteFile(ignoredFilePath, []byte("x"), 0o644)
	if writeError != nil {
		testingHandle.Fatalf("writing ignored file: %v", writeError)
	}
	fileOutputs, getError := commands.GetContentData(rootDirectory, []string{ignoredFileName}, nil)
	if getError != nil {
		testingHandle.Fatalf("GetContentData error: %v", getError)
	}
	if len(fileOutputs) != 2 {
		testingHandle.Fatalf("expected 2 files, got %d", len(fileOutputs))
	}
	foundText := false
	foundBinary := false
	for _, fileOutput := range fileOutputs {
		baseName := filepath.Base(fileOutput.Path)
		if baseName == textFileName {
			if fileOutput.Type != types.NodeTypeFile || fileOutput.Content != textFileContent {
				testingHandle.Fatalf("unexpected text file output: %+v", fileOutput)
			}
			foundText = true
		} else if baseName == binaryFileName {
			if fileOutput.Type != types.NodeTypeBinary || fileOutput.Content != "" || fileOutput.MimeType == "" {
				testingHandle.Fatalf("unexpected binary file output: %+v", fileOutput)
			}
			foundBinary = true
		} else {
			testingHandle.Fatalf("unexpected file: %s", baseName)
		}
	}
	if !foundText || !foundBinary {
		testingHandle.Fatalf("missing expected files")
	}
}

// TestGetTreeData verifies directory tree generation and ignore handling.
func TestGetTreeData(testingHandle *testing.T) {
	rootDirectory := testingHandle.TempDir()
	directoryPath := filepath.Join(rootDirectory, directoryName)
	ignoredDirectoryPath := filepath.Join(rootDirectory, ignoredDirName)
	makeDirError := os.MkdirAll(directoryPath, 0o755)
	if makeDirError != nil {
		testingHandle.Fatalf("mkdir: %v", makeDirError)
	}
	makeDirError = os.MkdirAll(ignoredDirectoryPath, 0o755)
	if makeDirError != nil {
		testingHandle.Fatalf("mkdir ignored: %v", makeDirError)
	}
	textFilePath := filepath.Join(directoryPath, textFileName)
	binaryFilePath := filepath.Join(rootDirectory, binaryFileName)
	writeError := os.WriteFile(textFilePath, []byte(textFileContent), 0o644)
	if writeError != nil {
		testingHandle.Fatalf("write text: %v", writeError)
	}
	writeError = os.WriteFile(binaryFilePath, []byte(binaryFileContent), 0o644)
	if writeError != nil {
		testingHandle.Fatalf("write binary: %v", writeError)
	}
	tree, treeError := commands.GetTreeData(rootDirectory, []string{ignoredDirName + "/"})
	if treeError != nil {
		testingHandle.Fatalf("GetTreeData error: %v", treeError)
	}
	if len(tree) != 1 {
		testingHandle.Fatalf("expected 1 root node, got %d", len(tree))
	}
	rootNode := tree[0]
	if len(rootNode.Children) != 2 {
		testingHandle.Fatalf("expected 2 children, got %d", len(rootNode.Children))
	}
	foundDir := false
	foundBinary := false
	for _, child := range rootNode.Children {
		baseName := filepath.Base(child.Path)
		if baseName == directoryName {
			if child.Type != types.NodeTypeDirectory || len(child.Children) != 1 {
				testingHandle.Fatalf("unexpected directory node: %+v", child)
			}
			if filepath.Base(child.Children[0].Path) != textFileName {
				testingHandle.Fatalf("missing text file in directory")
			}
			foundDir = true
		} else if baseName == binaryFileName {
			if child.Type != types.NodeTypeBinary {
				testingHandle.Fatalf("expected binary node, got %s", child.Type)
			}
			foundBinary = true
		} else {
			testingHandle.Fatalf("unexpected child: %s", baseName)
		}
	}
	if !foundDir || !foundBinary {
		testingHandle.Fatalf("missing expected nodes")
	}
}
