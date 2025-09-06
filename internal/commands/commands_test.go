package commands_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/temirov/ctx/internal/commands"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
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
		actualOutputs, err := commands.GetContentData(temporaryRoot, testCase.ignorePatterns, testCase.binaryContentPattern)
		if err != nil {
			testingInstance.Fatalf("case %d (%s): %v", index, testCase.testName, err)
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
		nodes, err := commands.GetTreeData(temporaryRoot, testCase.ignorePatterns)
		if err != nil {
			testingInstance.Fatalf("case %d (%s): %v", index, testCase.testName, err)
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
		}
	}
}
