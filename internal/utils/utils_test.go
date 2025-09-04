package utils_test

import (
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/temirov/ctx/internal/utils"
)

// textFileName defines the name of the text file used in tests.
const textFileName = "sample.txt"

// binaryFileName defines the name of the binary file used in tests.
const binaryFileName = "sample.bin"

// binaryBase64Content holds the base64 representation of the binary file content.
const binaryBase64Content = "AAE="

// directoryName defines the directory used for ignore tests.
const directoryName = "dir"

// exclusionPattern defines root-level exclusion pattern for the directory.
const exclusionPattern = utils.ExclusionPrefix + directoryName

// directoryPattern defines a pattern matching the directory.
const directoryPattern = directoryName + "/"

// wildcardTextPattern defines a pattern matching text files.
const wildcardTextPattern = "*.txt"

// wildcardMarkdownPattern defines a pattern matching markdown files.
const wildcardMarkdownPattern = "*.md"

// nestedDirectoryName defines the directory used for nested path tests.
const nestedDirectoryName = "subdir"

// nodeModulesDirectoryPattern defines the ignore pattern for the node_modules directory inside nestedDirectoryName.
const nodeModulesDirectoryPattern = nestedDirectoryName + "/node_modules/"

// backslashNodeModulesDirectoryPattern defines the same pattern with backslashes to verify normalization.
const backslashNodeModulesDirectoryPattern = nestedDirectoryName + `\node_modules\`

// nodeModulesDirectoryPath defines the path to the node_modules directory.
const nodeModulesDirectoryPath = nestedDirectoryName + "/node_modules"

// nodeModulesFilePath defines a file inside the node_modules directory.
const nodeModulesFilePath = nestedDirectoryName + "/node_modules/index.js"

// claspFilePattern defines the ignore pattern for a clasp configuration file inside nestedDirectoryName.
const claspFilePattern = nestedDirectoryName + "/.clasp.json"

// nestedFilePath defines the path to the clasp configuration file inside nestedDirectoryName.
const nestedFilePath = nestedDirectoryName + "/.clasp.json"

// unrelatedNodeModulesFilePath defines a node_modules path in an unrelated directory.
const unrelatedNodeModulesFilePath = "other/" + nodeModulesFilePath

// unrelatedNestedFilePath defines the clasp configuration file path in an unrelated directory.
const unrelatedNestedFilePath = "other/" + nestedFilePath

// mockDirEntry implements os.DirEntry for testing.
type mockDirEntry struct {
	entryName string
	directory bool
}

// Name returns the entry name.
func (entry mockDirEntry) Name() string { return entry.entryName }

// IsDir reports if the entry represents a directory.
func (entry mockDirEntry) IsDir() bool { return entry.directory }

// Type returns the file mode type.
func (entry mockDirEntry) Type() fs.FileMode {
	if entry.directory {
		return fs.ModeDir
	}
	return 0
}

// Info returns file information.
func (entry mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// TestDeduplicatePatterns verifies that DeduplicatePatterns removes duplicate patterns.
func TestDeduplicatePatterns(testingInstance *testing.T) {
	testCases := []struct {
		testName string
		patterns []string
		expected []string
	}{
		{
			testName: "removes duplicates",
			patterns: []string{"a", "b", "a"},
			expected: []string{"a", "b"},
		},
		{
			testName: "keeps unique",
			patterns: []string{"a", "b"},
			expected: []string{"a", "b"},
		},
	}
	for index, testCase := range testCases {
		actual := utils.DeduplicatePatterns(testCase.patterns)
		if len(actual) != len(testCase.expected) {
			testingInstance.Errorf("case %d (%s): expected length %d, got %d", index, testCase.testName, len(testCase.expected), len(actual))
			continue
		}
		for position, value := range actual {
			if value != testCase.expected[position] {
				testingInstance.Errorf("case %d (%s): expected %s at position %d, got %s", index, testCase.testName, testCase.expected[position], position, value)
			}
		}
	}
}

// TestContainsString verifies that ContainsString locates strings in a slice.
func TestContainsString(testingInstance *testing.T) {
	testCases := []struct {
		testName string
		slice    []string
		target   string
		expected bool
	}{
		{
			testName: "contains target",
			slice:    []string{"alpha", "beta"},
			target:   "beta",
			expected: true,
		},
		{
			testName: "missing target",
			slice:    []string{"alpha", "beta"},
			target:   "gamma",
			expected: false,
		},
	}
	for index, testCase := range testCases {
		actual := utils.ContainsString(testCase.slice, testCase.target)
		if actual != testCase.expected {
			testingInstance.Errorf("case %d (%s): expected %t, got %t", index, testCase.testName, testCase.expected, actual)
		}
	}
}

// TestRelativePathOrSelf verifies relative path calculations.
func TestRelativePathOrSelf(testingInstance *testing.T) {
	temporaryRoot := testingInstance.TempDir()
	subPath := filepath.Join(temporaryRoot, textFileName)
	creationError := os.WriteFile(subPath, []byte("content"), 0600)
	if creationError != nil {
		testingInstance.Fatalf("failed to create file: %v", creationError)
	}
	testCases := []struct {
		testName string
		fullPath string
		root     string
		expected string
	}{
		{
			testName: "root path returns dot",
			fullPath: temporaryRoot,
			root:     temporaryRoot,
			expected: ".",
		},
		{
			testName: "sub path returns relative",
			fullPath: subPath,
			root:     temporaryRoot,
			expected: textFileName,
		},
	}
	for index, testCase := range testCases {
		actual := utils.RelativePathOrSelf(testCase.fullPath, testCase.root)
		if actual != testCase.expected {
			testingInstance.Errorf("case %d (%s): expected %s, got %s", index, testCase.testName, testCase.expected, actual)
		}
	}
}

// TestShouldIgnore verifies directory entry ignoring rules.
func TestShouldIgnore(testingInstance *testing.T) {
	testCases := []struct {
		testName       string
		entry          mockDirEntry
		patterns       []string
		isRootLevel    bool
		expectedIgnore bool
	}{
		{
			testName:       "service file",
			entry:          mockDirEntry{entryName: utils.GitIgnoreFileName, directory: false},
			patterns:       nil,
			isRootLevel:    false,
			expectedIgnore: true,
		},
		{
			testName:       "exclude pattern",
			entry:          mockDirEntry{entryName: directoryName, directory: true},
			patterns:       []string{exclusionPattern},
			isRootLevel:    true,
			expectedIgnore: true,
		},
		{
			testName:       "directory pattern",
			entry:          mockDirEntry{entryName: directoryName, directory: true},
			patterns:       []string{directoryPattern},
			isRootLevel:    false,
			expectedIgnore: true,
		},
		{
			testName:       "wildcard file pattern",
			entry:          mockDirEntry{entryName: textFileName, directory: false},
			patterns:       []string{wildcardTextPattern},
			isRootLevel:    false,
			expectedIgnore: true,
		},
		{
			testName:       "not ignored",
			entry:          mockDirEntry{entryName: textFileName, directory: false},
			patterns:       []string{wildcardMarkdownPattern},
			isRootLevel:    false,
			expectedIgnore: false,
		},
	}
	for index, testCase := range testCases {
		actual := utils.ShouldIgnore(testCase.entry, testCase.patterns, testCase.isRootLevel)
		if actual != testCase.expectedIgnore {
			testingInstance.Errorf("case %d (%s): expected %t, got %t", index, testCase.testName, testCase.expectedIgnore, actual)
		}
	}
}

// TestShouldIgnoreByPath verifies path ignoring logic.
func TestShouldIgnoreByPath(testingInstance *testing.T) {
	testCases := []struct {
		testName       string
		relativePath   string
		patterns       []string
		expectedIgnore bool
	}{
		{
			testName:       "service file",
			relativePath:   ".gitignore",
			patterns:       nil,
			expectedIgnore: true,
		},
		{
			testName:       "exclude pattern",
			relativePath:   "dir/file.txt",
			patterns:       []string{"EXCL:dir"},
			expectedIgnore: true,
		},
		{
			testName:       "directory pattern for directory",
			relativePath:   "dir",
			patterns:       []string{"dir/"},
			expectedIgnore: true,
		},
		{
			testName:       "nested directory pattern",
			relativePath:   "dir/file.txt",
			patterns:       []string{"dir/*"},
			expectedIgnore: true,
		},
		{
			testName:       "wildcard file pattern",
			relativePath:   "dir/file.txt",
			patterns:       []string{"*.txt"},
			expectedIgnore: true,
		},
		{
			testName:       "path pattern",
			relativePath:   "dir/file.txt",
			patterns:       []string{"dir/*.txt"},
			expectedIgnore: true,
		},
		{
			testName:       "not ignored",
			relativePath:   "dir/file.txt",
			patterns:       []string{"*.md"},
			expectedIgnore: false,
		},
		{
			testName:       "nested directory with slash",
			relativePath:   nodeModulesFilePath,
			patterns:       []string{nodeModulesDirectoryPattern},
			expectedIgnore: true,
		},
		{
			testName:       "nested directory with backslashes",
			relativePath:   nodeModulesFilePath,
			patterns:       []string{backslashNodeModulesDirectoryPattern},
			expectedIgnore: true,
		},
		{
			testName:       "directory match short circuits",
			relativePath:   nodeModulesDirectoryPath,
			patterns:       []string{nodeModulesDirectoryPattern},
			expectedIgnore: true,
		},
		{
			testName:       "nested file pattern",
			relativePath:   nestedFilePath,
			patterns:       []string{claspFilePattern},
			expectedIgnore: true,
		},
		{
			testName:       "nested file pattern no match",
			relativePath:   unrelatedNestedFilePath,
			patterns:       []string{claspFilePattern},
			expectedIgnore: false,
		},
		{
			testName:       "nested directory pattern no match",
			relativePath:   unrelatedNodeModulesFilePath,
			patterns:       []string{nodeModulesDirectoryPattern},
			expectedIgnore: false,
		},
	}
	for index, testCase := range testCases {
		actual := utils.ShouldIgnoreByPath(testCase.relativePath, testCase.patterns)
		if actual != testCase.expectedIgnore {
			testingInstance.Errorf("case %d (%s): expected %t, got %t", index, testCase.testName, testCase.expectedIgnore, actual)
		}
	}
}

// TestShouldDisplayBinaryContentByPath verifies binary content display rules.
func TestShouldDisplayBinaryContentByPath(testingInstance *testing.T) {
	testCases := []struct {
		testName     string
		relativePath string
		patterns     []string
		expected     bool
	}{
		{
			testName:     "exact match",
			relativePath: "dir/" + binaryFileName,
			patterns:     []string{"dir/" + binaryFileName},
			expected:     true,
		},
		{
			testName:     "directory pattern",
			relativePath: "dir/sub/" + binaryFileName,
			patterns:     []string{"dir/"},
			expected:     true,
		},
		{
			testName:     "wildcard pattern",
			relativePath: binaryFileName,
			patterns:     []string{"*.bin"},
			expected:     true,
		},
		{
			testName:     "no match",
			relativePath: "dir/" + binaryFileName,
			patterns:     []string{"other/"},
			expected:     false,
		},
	}
	for index, testCase := range testCases {
		actual := utils.ShouldDisplayBinaryContentByPath(testCase.relativePath, testCase.patterns)
		if actual != testCase.expected {
			testingInstance.Errorf("case %d (%s): expected %t, got %t", index, testCase.testName, testCase.expected, actual)
		}
	}
}

// TestIsBinary verifies detection of binary data in byte slices.
func TestIsBinary(testingInstance *testing.T) {
	testCases := []struct {
		testName string
		data     []byte
		expected bool
	}{
		{
			testName: "utf8 text",
			data:     []byte("hello"),
			expected: false,
		},
		{
			testName: "null byte",
			data:     []byte{0x00, 0x01},
			expected: true,
		},
		{
			testName: "invalid utf8",
			data:     []byte{0xff},
			expected: true,
		},
		{
			testName: "empty slice",
			data:     []byte{},
			expected: false,
		},
	}
	for index, testCase := range testCases {
		actual := utils.IsBinary(testCase.data)
		if actual != testCase.expected {
			testingInstance.Errorf("case %d (%s): expected %t, got %t", index, testCase.testName, testCase.expected, actual)
		}
	}
}

// TestIsFileBinary verifies binary file detection.
func TestIsFileBinary(testingInstance *testing.T) {
	temporaryRoot := testingInstance.TempDir()
	textPath := filepath.Join(temporaryRoot, textFileName)
	binaryPath := filepath.Join(temporaryRoot, binaryFileName)
	textWriteError := os.WriteFile(textPath, []byte("hello"), 0600)
	if textWriteError != nil {
		testingInstance.Fatalf("writing text file: %v", textWriteError)
	}
	binaryBytes, decodeError := base64.StdEncoding.DecodeString(binaryBase64Content)
	if decodeError != nil {
		testingInstance.Fatalf("decoding base64: %v", decodeError)
	}
	binaryWriteError := os.WriteFile(binaryPath, binaryBytes, 0600)
	if binaryWriteError != nil {
		testingInstance.Fatalf("writing binary file: %v", binaryWriteError)
	}
	testCases := []struct {
		testName string
		path     string
		expected bool
	}{
		{
			testName: "text file",
			path:     textPath,
			expected: false,
		},
		{
			testName: "binary file",
			path:     binaryPath,
			expected: true,
		},
	}
	for index, testCase := range testCases {
		actual := utils.IsFileBinary(testCase.path)
		if actual != testCase.expected {
			testingInstance.Errorf("case %d (%s): expected %t, got %t", index, testCase.testName, testCase.expected, actual)
		}
	}
}
