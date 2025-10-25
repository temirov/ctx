package commands_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/tyemirov/ctx/internal/commands"
	"github.com/tyemirov/ctx/internal/types"
	"github.com/tyemirov/ctx/internal/utils"
)

// textFileName defines the text file name used in tests.
const textFileName = "example.txt"

// textFileContent provides the text content written to the text file.
const textFileContent = "hello"

// textMimeTypeExpected is the expected MIME type for the text file.
const textMimeTypeExpected = "text/plain; charset=utf-8"

// binaryFileName defines the binary file name used in tests.
const binaryFileName = "example.bin"

// binaryFileContent contains the raw bytes written to the binary file.
var binaryFileContent = []byte{0x00, 0x01}

// binaryBase64Content is the base64 representation of binaryFileContent.
const binaryBase64Content = "AAE="

// binaryMimeTypeExpected is the expected MIME type for the binary file.
const binaryMimeTypeExpected = "application/octet-stream"

// testCaseFailureMessage defines the format for subtest failure descriptions.
// stubCounter implements tokenizer.Counter for testing purposes.
type stubCounter struct{}

func (stubCounter) Name() string { return "stub" }

func (stubCounter) CountString(input string) (int, error) {
	return len([]rune(input)), nil
}

const testCaseFailureMessage = "case %d (%s): %v"

// TestGetContentData verifies content collection behavior.
func TestGetContentData(testingInstance *testing.T) {
	temporaryRoot := testingInstance.TempDir()
	textPath := filepath.Join(temporaryRoot, textFileName)
	binaryPath := filepath.Join(temporaryRoot, binaryFileName)
	textWriteError := os.WriteFile(textPath, []byte(textFileContent), 0600)
	if textWriteError != nil {
		testingInstance.Fatalf("writing text file: %v", textWriteError)
	}
	binaryWriteError := os.WriteFile(binaryPath, binaryFileContent, 0600)
	if binaryWriteError != nil {
		testingInstance.Fatalf("writing binary file: %v", binaryWriteError)
	}
	testCases := []struct {
		testName             string
		ignorePatterns       []string
		binaryContentPattern []string
		expectedOutputs      []types.FileOutput
	}{
		{
			testName:             "includes binary without content",
			ignorePatterns:       nil,
			binaryContentPattern: nil,
			expectedOutputs: []types.FileOutput{
				{Path: textPath, Type: types.NodeTypeFile, Content: textFileContent, MimeType: textMimeTypeExpected},
				{Path: binaryPath, Type: types.NodeTypeBinary, Content: utils.EmptyString, MimeType: binaryMimeTypeExpected},
			},
		},
		{
			testName:             "ignores by pattern",
			ignorePatterns:       []string{binaryFileName},
			binaryContentPattern: nil,
			expectedOutputs: []types.FileOutput{
				{Path: textPath, Type: types.NodeTypeFile, Content: textFileContent, MimeType: textMimeTypeExpected},
			},
		},
		{
			testName:             "displays binary content when allowed",
			ignorePatterns:       nil,
			binaryContentPattern: []string{binaryFileName},
			expectedOutputs: []types.FileOutput{
				{Path: textPath, Type: types.NodeTypeFile, Content: textFileContent, MimeType: textMimeTypeExpected},
				{Path: binaryPath, Type: types.NodeTypeBinary, Content: binaryBase64Content, MimeType: binaryMimeTypeExpected},
			},
		},
	}
	for index, testCase := range testCases {
		actualOutputs, contentRetrievalError := commands.GetContentData(temporaryRoot, testCase.ignorePatterns, testCase.binaryContentPattern, nil, "")
		if contentRetrievalError != nil {
			testingInstance.Fatalf(testCaseFailureMessage, index, testCase.testName, contentRetrievalError)
		}
		if len(actualOutputs) != len(testCase.expectedOutputs) {
			testingInstance.Errorf("case %d (%s): expected %d items, got %d", index, testCase.testName, len(testCase.expectedOutputs), len(actualOutputs))
			continue
		}
		sort.Slice(actualOutputs, func(firstIndex, secondIndex int) bool {
			return actualOutputs[firstIndex].Path < actualOutputs[secondIndex].Path
		})
		sort.Slice(testCase.expectedOutputs, func(firstIndex, secondIndex int) bool {
			return testCase.expectedOutputs[firstIndex].Path < testCase.expectedOutputs[secondIndex].Path
		})
		for position := range actualOutputs {
			actual := actualOutputs[position]
			expected := testCase.expectedOutputs[position]
			if actual.Path != expected.Path || actual.Type != expected.Type || actual.Content != expected.Content || actual.MimeType != expected.MimeType {
				testingInstance.Errorf("case %d (%s): mismatch at position %d", index, testCase.testName, position)
			}
			info, statError := os.Stat(actual.Path)
			if statError != nil {
				testingInstance.Fatalf("case %d (%s): stat failed for %s: %v", index, testCase.testName, actual.Path, statError)
			}
			expectedSize := utils.FormatFileSize(info.Size())
			if actual.Size != expectedSize {
				testingInstance.Errorf("case %d (%s): expected size %s for %s, got %s", index, testCase.testName, expectedSize, actual.Path, actual.Size)
			}
			expectedTimestamp := utils.FormatTimestamp(info.ModTime())
			if actual.LastModified != expectedTimestamp {
				testingInstance.Errorf("case %d (%s): expected last modified %s for %s, got %s", index, testCase.testName, expectedTimestamp, actual.Path, actual.LastModified)
			}
		}
	}
}

// TestGetTreeData verifies tree data generation behavior.
func TestGetTreeData(testingInstance *testing.T) {
	temporaryRoot := testingInstance.TempDir()
	textPath := filepath.Join(temporaryRoot, textFileName)
	binaryPath := filepath.Join(temporaryRoot, binaryFileName)
	textWriteError := os.WriteFile(textPath, []byte(textFileContent), 0600)
	if textWriteError != nil {
		testingInstance.Fatalf("writing text file: %v", textWriteError)
	}
	binaryWriteError := os.WriteFile(binaryPath, binaryFileContent, 0600)
	if binaryWriteError != nil {
		testingInstance.Fatalf("writing binary file: %v", binaryWriteError)
	}
	testCases := []struct {
		testName         string
		ignorePatterns   []string
		expectedChildren []types.TreeOutputNode
	}{
		{
			testName:       "includes binary file",
			ignorePatterns: nil,
			expectedChildren: []types.TreeOutputNode{
				{Path: textPath, Name: textFileName, Type: types.NodeTypeFile, MimeType: textMimeTypeExpected},
				{Path: binaryPath, Name: binaryFileName, Type: types.NodeTypeBinary, MimeType: binaryMimeTypeExpected},
			},
		},
		{
			testName:       "ignores by pattern",
			ignorePatterns: []string{binaryFileName},
			expectedChildren: []types.TreeOutputNode{
				{Path: textPath, Name: textFileName, Type: types.NodeTypeFile, MimeType: textMimeTypeExpected},
			},
		},
	}
	for index, testCase := range testCases {
		treeBuilder := commands.TreeBuilder{IgnorePatterns: testCase.ignorePatterns}
		nodes, treeConstructionError := treeBuilder.GetTreeData(temporaryRoot)
		if treeConstructionError != nil {
			testingInstance.Fatalf(testCaseFailureMessage, index, testCase.testName, treeConstructionError)
		}
		if len(nodes) != 1 {
			testingInstance.Errorf("case %d (%s): expected one root node, got %d", index, testCase.testName, len(nodes))
			continue
		}
		rootNode := nodes[0]
		actualChildren := rootNode.Children
		if len(actualChildren) != len(testCase.expectedChildren) {
			testingInstance.Errorf("case %d (%s): expected %d children, got %d", index, testCase.testName, len(testCase.expectedChildren), len(actualChildren))
			continue
		}
		sort.Slice(actualChildren, func(firstIndex, secondIndex int) bool {
			return actualChildren[firstIndex].Path < actualChildren[secondIndex].Path
		})
		sort.Slice(testCase.expectedChildren, func(firstIndex, secondIndex int) bool {
			return testCase.expectedChildren[firstIndex].Path < testCase.expectedChildren[secondIndex].Path
		})
		for position := range actualChildren {
			actual := actualChildren[position]
			expected := testCase.expectedChildren[position]
			if actual.Path != expected.Path || actual.Type != expected.Type || actual.Name != expected.Name || actual.MimeType != expected.MimeType {
				testingInstance.Errorf("case %d (%s): child mismatch at position %d", index, testCase.testName, position)
			}
			info, statError := os.Stat(actual.Path)
			if statError != nil {
				testingInstance.Fatalf("case %d (%s): stat failed for %s: %v", index, testCase.testName, actual.Path, statError)
			}
			expectedSize := utils.FormatFileSize(info.Size())
			if actual.Size != expectedSize {
				testingInstance.Errorf("case %d (%s): expected size %s for %s, got %s", index, testCase.testName, expectedSize, actual.Path, actual.Size)
			}
			expectedTimestamp := utils.FormatTimestamp(info.ModTime())
			if actual.LastModified != expectedTimestamp {
				testingInstance.Errorf("case %d (%s): expected last modified %s for %s, got %s", index, testCase.testName, expectedTimestamp, actual.Path, actual.LastModified)
			}
		}
	}
}

// TestGetContentDataWithTokens verifies token counting integration for content data.
func TestGetContentDataWithTokens(testingInstance *testing.T) {
	temporaryRoot := testingInstance.TempDir()
	textPath := filepath.Join(temporaryRoot, "tokens.txt")
	const fileContent = "token count"
	if err := os.WriteFile(textPath, []byte(fileContent), 0o600); err != nil {
		testingInstance.Fatalf("writing file: %v", err)
	}
	outputs, err := commands.GetContentData(temporaryRoot, nil, nil, stubCounter{}, "stub")
	if err != nil {
		testingInstance.Fatalf("GetContentData error: %v", err)
	}
	if len(outputs) == 0 {
		testingInstance.Fatalf("expected at least one output")
	}
	var tokens int
	var model string
	for _, output := range outputs {
		if output.Path == textPath {
			tokens = output.Tokens
			model = output.Model
			break
		}
	}
	if tokens != len([]rune(fileContent)) {
		testingInstance.Errorf("expected %d tokens, got %d", len([]rune(fileContent)), tokens)
	}
	if model != "stub" {
		testingInstance.Errorf("expected model 'stub', got %q", model)
	}
}

// TestTreeBuilderTokenCounts verifies token aggregation in tree summaries.
func TestTreeBuilderTokenCounts(testingInstance *testing.T) {
	temporaryRoot := testingInstance.TempDir()
	textPath := filepath.Join(temporaryRoot, "tokens.txt")
	const fileContent = "token count"
	if err := os.WriteFile(textPath, []byte(fileContent), 0o600); err != nil {
		testingInstance.Fatalf("writing file: %v", err)
	}
	treeBuilder := commands.TreeBuilder{TokenCounter: stubCounter{}, IncludeSummary: true, TokenModel: "stub"}
	nodes, err := treeBuilder.GetTreeData(temporaryRoot)
	if err != nil {
		testingInstance.Fatalf("GetTreeData error: %v", err)
	}
	if len(nodes) != 1 {
		testingInstance.Fatalf("expected single root node")
	}
	root := nodes[0]
	if root == nil {
		testingInstance.Fatalf("root node nil")
	}
	var fileNode *types.TreeOutputNode
	for _, child := range root.Children {
		if child != nil && child.Path == textPath {
			fileNode = child
			break
		}
	}
	if fileNode == nil {
		testingInstance.Fatalf("file node not found")
	}
	expectedTokens := len([]rune(fileContent))
	if fileNode.Tokens != expectedTokens {
		testingInstance.Errorf("expected file tokens %d, got %d", expectedTokens, fileNode.Tokens)
	}
	if fileNode.Model != "stub" {
		testingInstance.Errorf("expected file model 'stub', got %q", fileNode.Model)
	}
	if root.TotalTokens != expectedTokens {
		testingInstance.Errorf("expected root tokens %d, got %d", expectedTokens, root.TotalTokens)
	}
	if root.Model != "stub" {
		testingInstance.Errorf("expected root model 'stub', got %q", root.Model)
	}
}

// TestGetTreeDataSummary verifies summary aggregation for directory trees.
// TestGetTreeDataSummary verifies summary aggregation for directory trees.
func TestGetTreeDataSummary(testingInstance *testing.T) {
	temporaryRoot := testingInstance.TempDir()
	childDir := filepath.Join(temporaryRoot, "pkg")
	if mkdirError := os.Mkdir(childDir, 0700); mkdirError != nil {
		testingInstance.Fatalf("creating child directory: %v", mkdirError)
	}
	rootFilePath := filepath.Join(temporaryRoot, "root.txt")
	childFilePath := filepath.Join(childDir, "child.txt")
	rootContent := []byte("root-data")
	childContent := []byte("child-data")
	if writeError := os.WriteFile(rootFilePath, rootContent, 0600); writeError != nil {
		testingInstance.Fatalf("writing root file: %v", writeError)
	}
	if writeError := os.WriteFile(childFilePath, childContent, 0600); writeError != nil {
		testingInstance.Fatalf("writing child file: %v", writeError)
	}
	treeBuilder := commands.TreeBuilder{IncludeSummary: true}
	nodes, treeConstructionError := treeBuilder.GetTreeData(temporaryRoot)
	if treeConstructionError != nil {
		testingInstance.Fatalf("building tree with summary: %v", treeConstructionError)
	}
	if len(nodes) != 1 {
		testingInstance.Fatalf("expected single root node, got %d", len(nodes))
	}
	rootNode := nodes[0]
	expectedRootFiles := 2
	if rootNode.TotalFiles != expectedRootFiles {
		testingInstance.Fatalf("expected %d files at root, got %d", expectedRootFiles, rootNode.TotalFiles)
	}
	rootExpectedSize := utils.FormatFileSize(int64(len(rootContent) + len(childContent)))
	if rootNode.TotalSize != rootExpectedSize {
		testingInstance.Fatalf("expected root size %s, got %s", rootExpectedSize, rootNode.TotalSize)
	}
	var directoryNode, fileNode *types.TreeOutputNode
	for _, child := range rootNode.Children {
		switch child.Path {
		case rootFilePath:
			fileNode = child
		case childDir:
			directoryNode = child
		}
	}
	if fileNode == nil || directoryNode == nil {
		testingInstance.Fatalf("expected both file and directory children, got %+v", rootNode.Children)
	}
	if fileNode.TotalFiles != 0 {
		testingInstance.Fatalf("expected no summary on file node, got %+v", fileNode)
	}
	if fileNode.TotalSize != "" {
		testingInstance.Fatalf("expected empty total size on file node, got %s", fileNode.TotalSize)
	}
	if directoryNode.TotalFiles != 1 {
		testingInstance.Fatalf("directory summary incorrect: %+v", directoryNode)
	}
	directoryExpectedSize := utils.FormatFileSize(int64(len(childContent)))
	if directoryNode.TotalSize != directoryExpectedSize {
		testingInstance.Fatalf("expected directory summary size %s, got %s", directoryExpectedSize, directoryNode.TotalSize)
	}
	if len(directoryNode.Children) != 1 {
		testingInstance.Fatalf("expected one child under directory, got %d", len(directoryNode.Children))
	}
	nestedFile := directoryNode.Children[0]
	if nestedFile.TotalFiles != 0 {
		testingInstance.Fatalf("expected no summary on nested file node, got %+v", nestedFile)
	}
	if nestedFile.TotalSize != "" {
		testingInstance.Fatalf("expected empty total size on nested file node, got %s", nestedFile.TotalSize)
	}
}
