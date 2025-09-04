package utils_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/temirov/ctx/internal/utils"
)

const (
	textFileName      = "plain.txt"
	binaryFileName    = "data.bin"
	ignoredFileName   = "ignored.txt"
	textFileContent   = "hello world"
	binaryFileContent = "\x00\xff\x00\xff"
	filePatternIgnore = ignoredFileName
	filePatternBinary = binaryFileName
	directoryPattern  = "dir/"
	exclusionPattern  = utils.ExclusionPrefix + "node"
	binaryPattern     = "bin/"
)

// TestIsBinary verifies binary data detection for byte slices.
func TestIsBinary(testingHandle *testing.T) {
	const (
		asciiData    = "hello"
		nullByteData = "hi\x00"
		invalidUTF8  = "\xff\xfe"
	)
	testCases := []struct {
		name           string
		data           []byte
		expectedBinary bool
	}{
		{name: "empty", data: []byte{}, expectedBinary: false},
		{name: "ascii", data: []byte(asciiData), expectedBinary: false},
		{name: "null byte", data: []byte(nullByteData), expectedBinary: true},
		{name: "invalid utf8", data: []byte(invalidUTF8), expectedBinary: true},
	}
	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			isBinary := utils.IsBinary(testCase.data)
			if isBinary != testCase.expectedBinary {
				testingHandle.Fatalf("IsBinary(%v) = %v, want %v", testCase.data, isBinary, testCase.expectedBinary)
			}
		})
	}
}

// TestIsFileBinary verifies binary file detection.
func TestIsFileBinary(testingHandle *testing.T) {
	rootDirectory := testingHandle.TempDir()
	testCases := []struct {
		name           string
		fileName       string
		content        []byte
		expectedBinary bool
	}{
		{name: "text file", fileName: textFileName, content: []byte(textFileContent), expectedBinary: false},
		{name: "binary file", fileName: binaryFileName, content: []byte(binaryFileContent), expectedBinary: true},
	}
	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			filePath := filepath.Join(rootDirectory, testCase.fileName)
			writeError := os.WriteFile(filePath, testCase.content, 0o644)
			if writeError != nil {
				testingHandle.Fatalf("writing test file: %v", writeError)
			}
			isBinary := utils.IsFileBinary(filePath)
			if isBinary != testCase.expectedBinary {
				testingHandle.Fatalf("IsFileBinary(%s) = %v, want %v", filePath, isBinary, testCase.expectedBinary)
			}
		})
	}
}

// TestShouldIgnoreByPath verifies path-based ignore matching.
func TestShouldIgnoreByPath(testingHandle *testing.T) {
	testCases := []struct {
		name            string
		path            string
		patterns        []string
		expectedIgnored bool
	}{
		{name: "service file", path: utils.IgnoreFileName, patterns: nil, expectedIgnored: true},
		{name: "file pattern", path: ignoredFileName, patterns: []string{filePatternIgnore}, expectedIgnored: true},
		{name: "directory pattern", path: "dir", patterns: []string{directoryPattern}, expectedIgnored: true},
		{name: "exclusion pattern", path: "node/item.txt", patterns: []string{exclusionPattern}, expectedIgnored: true},
		{name: "no match", path: "keep.txt", patterns: []string{filePatternIgnore}, expectedIgnored: false},
	}
	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			ignored := utils.ShouldIgnoreByPath(testCase.path, testCase.patterns)
			if ignored != testCase.expectedIgnored {
				testingHandle.Fatalf("ShouldIgnoreByPath(%s) = %v, want %v", testCase.path, ignored, testCase.expectedIgnored)
			}
		})
	}
}

// TestShouldDisplayBinaryContentByPath verifies binary content display decisions.
func TestShouldDisplayBinaryContentByPath(testingHandle *testing.T) {
	testCases := []struct {
		name             string
		path             string
		patterns         []string
		expectedDecision bool
	}{
		{name: "exact match", path: filePatternBinary, patterns: []string{filePatternBinary}, expectedDecision: true},
		{name: "directory match", path: "bin/a.dat", patterns: []string{binaryPattern}, expectedDecision: true},
		{name: "no match", path: "other.bin", patterns: []string{binaryPattern}, expectedDecision: false},
	}
	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			decision := utils.ShouldDisplayBinaryContentByPath(testCase.path, testCase.patterns)
			if decision != testCase.expectedDecision {
				testingHandle.Fatalf("ShouldDisplayBinaryContentByPath(%s) = %v, want %v", testCase.path, decision, testCase.expectedDecision)
			}
		})
	}
}

// BenchmarkIsBinary provides a basic benchmark for the IsBinary function.
func BenchmarkIsBinary(benchmarkHandle *testing.B) {
	data := []byte(base64.StdEncoding.EncodeToString([]byte(binaryFileContent)))
	for benchmarkIteration := 0; benchmarkIteration < benchmarkHandle.N; benchmarkIteration++ {
		utils.IsBinary(data)
	}
}
