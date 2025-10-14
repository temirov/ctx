package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/temirov/ctx/internal/config"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

type treeStubCounter struct{}

func (treeStubCounter) Name() string { return "stub" }

func (treeStubCounter) CountString(input string) (int, error) {
	return len([]rune(input)), nil
}

type clipboardStub struct {
	copiedText      string
	errorToReturn   error
	invocationCount int
}

func (stub *clipboardStub) Copy(text string) error {
	stub.invocationCount++
	stub.copiedText = text
	return stub.errorToReturn
}

func boolPtr(value bool) *bool {
	pointer := value
	return &pointer
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writePipe

	var buffer bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buffer, readPipe)
		close(done)
	}()

	fn()

	writePipe.Close()
	os.Stdout = original
	<-done
	return buffer.String()
}

func TestRunTreeRawStreamingOutputsSummaryAfterFiles(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	rootFile := filepath.Join(tempDir, "root.txt")
	nestedFile := filepath.Join(nestedDir, "inner.txt")

	if err := os.WriteFile(rootFile, []byte("token"), 0o600); err != nil {
		t.Fatalf("write root file: %v", err)
	}
	if err := os.WriteFile(nestedFile, []byte("abc"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	outputText := captureStdout(t, func() {
		if err := runStreamCommand(
			context.Background(),
			types.CommandTree,
			[]string{tempDir},
			nil,
			true,
			true,
			false,
			types.FormatRaw,
			false,
			true,
			false,
			treeStubCounter{},
			"stub-model",
			nil,
			os.Stdout,
			os.Stderr,
		); err != nil {
			t.Fatalf("runStreamCommand error: %v", err)
		}
	})

	if !strings.Contains(outputText, tempDir) {
		t.Fatalf("expected directory path in output")
	}
	if !strings.Contains(outputText, nestedDir) {
		t.Fatalf("expected nested directory path in output")
	}
	if !strings.Contains(outputText, rootFile) {
		t.Fatalf("expected root file in output")
	}
	if !strings.Contains(outputText, nestedFile) {
		t.Fatalf("expected nested file in output")
	}

	if !strings.Contains(outputText, "├──") {
		t.Fatalf("expected tree branch connector in output")
	}
	if !strings.Contains(outputText, "└──") {
		t.Fatalf("expected tree leaf connector in output")
	}

	nestedDirIndex := strings.Index(outputText, nestedDir)
	if nestedDirIndex == -1 {
		t.Fatalf("expected nested directory entry in output")
	}
	nestedFileIndex := strings.Index(outputText, nestedFile)
	nestedSummaryKey := fmt.Sprintf("Summary: 1 file, %s", utils.FormatFileSize(3))
	nestedSummaryIndex := strings.Index(outputText, nestedSummaryKey)
	if nestedSummaryIndex == -1 {
		t.Fatalf("expected nested summary line in output")
	}
	if !(nestedDirIndex < nestedSummaryIndex && nestedSummaryIndex < nestedFileIndex) {
		t.Fatalf("nested summary ordering incorrect: %s", outputText)
	}

	if !strings.Contains(outputText, "Summary: 2 files") {
		t.Fatalf("expected global summary in output")
	}
}

func TestRunContentRawStreamingStreamsBeforeSummary(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "content.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	outputText := captureStdout(t, func() {
		if err := runStreamCommand(
			context.Background(),
			types.CommandContent,
			[]string{tempDir},
			nil,
			true,
			true,
			false,
			types.FormatRaw,
			false,
			true,
			true,
			treeStubCounter{},
			"stub-model",
			nil,
			os.Stdout,
			os.Stderr,
		); err != nil {
			t.Fatalf("runStreamCommand error: %v", err)
		}
	})

	if !strings.Contains(outputText, "File: "+filePath) {
		t.Fatalf("expected file header in output")
	}
	fileHeaderIndex := strings.Index(outputText, "File: "+filePath)
	endMarker := "End of file: " + filePath
	endIndex := strings.Index(outputText, endMarker)
	if endIndex == -1 {
		t.Fatalf("expected end of file marker")
	}
	if endIndex < fileHeaderIndex {
		t.Fatalf("end marker appeared before file header")
	}
	summaryIndex := strings.LastIndex(outputText, "Summary:")
	if summaryIndex == -1 {
		t.Fatalf("expected summary in output")
	}
	if summaryIndex < endIndex {
		t.Fatalf("summary appeared before file finished streaming")
	}
}

func TestRunToolCopiesOutputToClipboard(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "clipboard.txt")
	if err := os.WriteFile(filePath, []byte("clipboard data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	testCases := []struct {
		name           string
		clipboardError error
		expectError    bool
	}{
		{
			name:           "copies output when clipboard succeeds",
			clipboardError: nil,
			expectError:    false,
		},
		{
			name:           "returns error when clipboard copy fails",
			clipboardError: errors.New("clipboard failure"),
			expectError:    true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stub := &clipboardStub{errorToReturn: testCase.clipboardError}
			var outputBuffer bytes.Buffer
			executionError := runTool(
				context.Background(),
				types.CommandContent,
				[]string{tempDir},
				nil,
				false,
				false,
				false,
				defaultCallChainDepth,
				types.FormatJSON,
				false,
				false,
				true,
				tokenOptions{},
				&outputBuffer,
				io.Discard,
				true,
				stub,
			)

			if testCase.expectError {
				if executionError == nil {
					t.Fatalf("expected error when clipboard copy fails")
				}
				return
			}

			if executionError != nil {
				t.Fatalf("unexpected error: %v", executionError)
			}
			if stub.invocationCount != 1 {
				t.Fatalf("expected clipboard to be used once, got %d", stub.invocationCount)
			}
			if stub.copiedText != outputBuffer.String() {
				t.Fatalf("clipboard text mismatch\nexpected: %q\nactual: %q", outputBuffer.String(), stub.copiedText)
			}
		})
	}
}

func TestApplyStreamConfigurationUsesDefaults(t *testing.T) {
	var pathConfiguration pathOptions
	format := types.FormatJSON
	summaryEnabled := true
	documentationEnabled := false
	tokens := tokenOptions{model: "initial"}
	includeContent := false

	command := &cobra.Command{Use: "test"}
	command.Flags().StringVar(&format, formatFlagName, format, "")
	command.Flags().BoolVar(&summaryEnabled, summaryFlagName, summaryEnabled, "")
	command.Flags().BoolVar(&documentationEnabled, documentationFlagName, documentationEnabled, "")
	command.Flags().BoolVar(&tokens.enabled, tokensFlagName, tokens.enabled, "")
	command.Flags().StringVar(&tokens.model, modelFlagName, tokens.model, "")
	command.Flags().StringArrayVarP(&pathConfiguration.exclusionPatterns, exclusionFlagName, exclusionFlagName, pathConfiguration.exclusionPatterns, "")
	command.Flags().BoolVar(&pathConfiguration.disableGitignore, noGitignoreFlagName, pathConfiguration.disableGitignore, "")
	command.Flags().BoolVar(&pathConfiguration.disableIgnoreFile, noIgnoreFlagName, pathConfiguration.disableIgnoreFile, "")
	command.Flags().BoolVar(&pathConfiguration.includeGit, includeGitFlagName, pathConfiguration.includeGit, "")
	command.Flags().BoolVar(&includeContent, contentFlagName, includeContent, "")

	configuration := config.StreamCommandConfiguration{
		Format:         types.FormatXML,
		Summary:        boolPtr(false),
		Documentation:  boolPtr(true),
		IncludeContent: boolPtr(true),
		Tokens: config.TokenConfiguration{
			Enabled: boolPtr(true),
			Model:   "custom",
		},
		Paths: config.PathConfiguration{
			Exclude:       []string{"vendor"},
			UseGitignore:  boolPtr(false),
			UseIgnoreFile: boolPtr(false),
			IncludeGit:    boolPtr(true),
		},
	}

	applyStreamConfiguration(command, configuration, &pathConfiguration, &format, &documentationEnabled, &summaryEnabled, &includeContent, &tokens)

	if format != types.FormatXML {
		t.Fatalf("expected format %s, got %s", types.FormatXML, format)
	}
	if summaryEnabled {
		t.Fatalf("expected summary to be disabled")
	}
	if !documentationEnabled {
		t.Fatalf("expected documentation to be enabled")
	}
	if !includeContent {
		t.Fatalf("expected includeContent to be enabled")
	}
	if !tokens.enabled {
		t.Fatalf("expected tokens to be enabled")
	}
	if tokens.model != "custom" {
		t.Fatalf("expected token model custom, got %s", tokens.model)
	}
	if len(pathConfiguration.exclusionPatterns) != 1 || pathConfiguration.exclusionPatterns[0] != "vendor" {
		t.Fatalf("expected exclusion patterns to include vendor: %v", pathConfiguration.exclusionPatterns)
	}
	if !pathConfiguration.disableGitignore {
		t.Fatalf("expected gitignore to be disabled")
	}
	if !pathConfiguration.disableIgnoreFile {
		t.Fatalf("expected ignore file to be disabled")
	}
	if !pathConfiguration.includeGit {
		t.Fatalf("expected git directory to be included")
	}
}

func TestApplyStreamConfigurationRespectsCliOverrides(t *testing.T) {
	pathConfiguration := pathOptions{
		exclusionPatterns: []string{"cli"},
		disableGitignore:  true,
		disableIgnoreFile: true,
		includeGit:        false,
	}
	format := "cli"
	summaryEnabled := true
	documentationEnabled := true
	tokens := tokenOptions{enabled: true, model: "cli"}
	includeContent := true

	command := &cobra.Command{Use: "test"}
	command.Flags().StringVar(&format, formatFlagName, format, "")
	command.Flags().BoolVar(&summaryEnabled, summaryFlagName, summaryEnabled, "")
	command.Flags().BoolVar(&documentationEnabled, documentationFlagName, documentationEnabled, "")
	command.Flags().BoolVar(&tokens.enabled, tokensFlagName, tokens.enabled, "")
	command.Flags().StringVar(&tokens.model, modelFlagName, tokens.model, "")
	command.Flags().StringArrayVarP(&pathConfiguration.exclusionPatterns, exclusionFlagName, exclusionFlagName, pathConfiguration.exclusionPatterns, "")
	command.Flags().BoolVar(&pathConfiguration.disableGitignore, noGitignoreFlagName, pathConfiguration.disableGitignore, "")
	command.Flags().BoolVar(&pathConfiguration.disableIgnoreFile, noIgnoreFlagName, pathConfiguration.disableIgnoreFile, "")
	command.Flags().BoolVar(&pathConfiguration.includeGit, includeGitFlagName, pathConfiguration.includeGit, "")
	command.Flags().BoolVar(&includeContent, contentFlagName, includeContent, "")

	// Simulate CLI overrides.
	if err := command.Flags().Set(formatFlagName, format); err != nil {
		t.Fatalf("set format flag: %v", err)
	}
	if err := command.Flags().Set(summaryFlagName, "true"); err != nil {
		t.Fatalf("set summary flag: %v", err)
	}
	if err := command.Flags().Set(documentationFlagName, "true"); err != nil {
		t.Fatalf("set doc flag: %v", err)
	}
	if err := command.Flags().Set(tokensFlagName, "true"); err != nil {
		t.Fatalf("set tokens flag: %v", err)
	}
	if err := command.Flags().Set(modelFlagName, tokens.model); err != nil {
		t.Fatalf("set model flag: %v", err)
	}
	if err := command.Flags().Set(exclusionFlagName, "cli"); err != nil {
		t.Fatalf("set exclusion flag: %v", err)
	}
	if err := command.Flags().Set(noGitignoreFlagName, "true"); err != nil {
		t.Fatalf("set gitignore flag: %v", err)
	}
	if err := command.Flags().Set(noIgnoreFlagName, "true"); err != nil {
		t.Fatalf("set ignore flag: %v", err)
	}
	if err := command.Flags().Set(includeGitFlagName, "false"); err != nil {
		t.Fatalf("set include git flag: %v", err)
	}
	if err := command.Flags().Set(contentFlagName, "true"); err != nil {
		t.Fatalf("set content flag: %v", err)
	}

	configuration := config.StreamCommandConfiguration{
		Format:         types.FormatXML,
		Summary:        boolPtr(false),
		Documentation:  boolPtr(false),
		IncludeContent: boolPtr(false),
		Tokens: config.TokenConfiguration{
			Enabled: boolPtr(false),
			Model:   "config",
		},
		Paths: config.PathConfiguration{
			Exclude:       []string{"config"},
			UseGitignore:  boolPtr(true),
			UseIgnoreFile: boolPtr(true),
			IncludeGit:    boolPtr(true),
		},
	}

	applyStreamConfiguration(command, configuration, &pathConfiguration, &format, &documentationEnabled, &summaryEnabled, &includeContent, &tokens)

	if format != "cli" {
		t.Fatalf("expected format to remain cli, got %s", format)
	}
	if !summaryEnabled {
		t.Fatalf("expected summary to remain true")
	}
	if !documentationEnabled {
		t.Fatalf("expected documentation to remain true")
	}
	if !includeContent {
		t.Fatalf("expected includeContent to remain true")
	}
	if !tokens.enabled {
		t.Fatalf("expected tokens to remain enabled")
	}
	if tokens.model != "cli" {
		t.Fatalf("expected token model to remain cli, got %s", tokens.model)
	}
	if len(pathConfiguration.exclusionPatterns) != 1 || pathConfiguration.exclusionPatterns[0] != "cli" {
		t.Fatalf("expected exclusion patterns to remain cli: %v", pathConfiguration.exclusionPatterns)
	}
	if !pathConfiguration.disableGitignore {
		t.Fatalf("expected gitignore to remain disabled")
	}
	if !pathConfiguration.disableIgnoreFile {
		t.Fatalf("expected ignore file to remain disabled")
	}
	if pathConfiguration.includeGit {
		t.Fatalf("expected include git to remain false")
	}
}

func TestApplyCallChainConfigurationUsesDefaults(t *testing.T) {
	format := types.FormatJSON
	documentationEnabled := false
	depth := defaultCallChainDepth

	command := &cobra.Command{Use: "callchain"}
	command.Flags().StringVar(&format, formatFlagName, format, "")
	command.Flags().BoolVar(&documentationEnabled, documentationFlagName, documentationEnabled, "")
	command.Flags().IntVar(&depth, callChainDepthFlagName, depth, "")

	configuration := config.CallChainConfiguration{
		Format:        types.FormatXML,
		Depth:         func() *int { value := 2; return &value }(),
		Documentation: boolPtr(true),
	}

	applyCallChainConfiguration(command, configuration, &format, &depth, &documentationEnabled)

	if format != types.FormatXML {
		t.Fatalf("expected format to change to xml")
	}
	if depth != 2 {
		t.Fatalf("expected depth to be updated to 2")
	}
	if !documentationEnabled {
		t.Fatalf("expected documentation to be enabled")
	}
}
