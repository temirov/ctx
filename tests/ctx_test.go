// Package tests contains the integration‑level test‑suite for ctx.
package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	appTypes "github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

const (
	visibleFileName    = "visible.txt"
	hiddenFileName     = "hidden.txt"
	visibleFileContent = "visible"
	hiddenFileContent  = "secret"
	includeGitFlag     = "--git"
	versionFlag        = "--version"
	// documentationFlag enables inclusion of documentation.
	documentationFlag = "--doc"
	treeAlias         = "t"
	// contentAlias represents the shorthand for the content command.
	contentAlias          = "c"
	subDirectoryName      = "sub"
	gitignoreFileName     = ".gitignore"
	nodeModulesDirName    = "node_modules"
	dependencyFileName    = "dependency.js"
	nodeModulesPattern    = nodeModulesDirName + "/\n"
	dependencyFileContent = "dependency"
	// ignoredFileName names a file that should be excluded by gitignore.
	ignoredFileName = "ignored.txt"
	// ignoredFileContent supplies the content for ignoredFileName.
	ignoredFileContent = "ignore"
	// googleSheetsAddonDirectoryName names the root directory of the Google Sheets add-on fixture.
	googleSheetsAddonDirectoryName = "google-sheets-addon"
	// claspConfigurationFileName is the configuration file name for the Apps Script CLI.
	claspConfigurationFileName = ".clasp.json"
	// claspConfigurationPattern instructs git to ignore the clasp configuration file.
	claspConfigurationPattern = claspConfigurationFileName + "\n"
	// claspConfigurationContent is placeholder JSON content for the clasp configuration file.
	claspConfigurationContent = "{\"scriptId\":\"1\"}"
	// googleSheetsAddonGitignoreContent combines ignore rules for the add-on fixture.
	googleSheetsAddonGitignoreContent = nodeModulesPattern + claspConfigurationPattern
	commandDirectoryRelativePath      = "cmd/ctx"
	integrationBinaryBaseName         = "ctx_integration_binary"
	contentDataFunction               = "github.com/temirov/ctx/internal/commands.GetContentData"
	runTreeOrContentCommandFunction   = "github.com/temirov/ctx/internal/cli.runTreeOrContentCommand"
	runToolFunction                   = "github.com/temirov/ctx/internal/cli.runTool"
	callChainAlias                    = "cc"
	formatFlag                        = "--format"
	depthFlag                         = "--depth"
	depthTwoValue                     = "2"

	usageSnippet = "Usage:\n  ctx"
	// unknownDocumentationFlagErrorSnippet captures the error when documentation flag is unsupported.
	unknownDocumentationFlagErrorSnippet = "unknown flag: --doc"

	binaryFixtureFileName  = "fixture.png"
	expectedBinaryMimeType = "image/png"
	ignoreFileName         = ".ignore"
	// binarySectionHeader identifies the section that lists binary content patterns in an ignore file.
	binarySectionHeader        = "[binary]"
	unmatchedBinaryFixtureName = "unmatched.bin"
	onePixelPNGBase64Content   = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	expectedTextMimeType       = "text/plain; charset=utf-8"
	mimeTypeIndicator          = "Mime Type:"
	toolsDirectoryName         = "tools"
	githubDirectoryName        = ".github"
	yamlRootFileName           = "config.yml"
	yamlPattern                = "*.yml"
	nestedDirectoryName        = "nested"
	nestedYamlFileName         = "nested.yml"
	nestedTextFileName         = "keep.txt"
)

// buildBinary compiles the ctx binary and returns its path.
func buildBinary(testingHandle *testing.T) string {
	testingHandle.Helper()

	temporaryDirectory := testingHandle.TempDir()
	binaryName := integrationBinaryBaseName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(temporaryDirectory, binaryName)

	moduleRootDirectory := getModuleRoot(testingHandle)
	commandDirectory := filepath.Join(moduleRootDirectory, commandDirectoryRelativePath)
	buildCommand := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCommand.Dir = commandDirectory

	combinedOutput, buildError := buildCommand.CombinedOutput()
	if buildError != nil {
		testingHandle.Fatalf("build failed in %s: %v\n%s", commandDirectory, buildError, string(combinedOutput))
	}

	return binaryPath
}

// runCommand executes the binary with arguments and returns stdout.
func runCommand(testingHandle *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testingHandle.Helper()

	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory

	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer

	runError := command.Run()

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	if runError != nil {
		if exitError, ok := runError.(*exec.ExitError); ok {
			testingHandle.Fatalf("command failed (%d): %v\nstdout:\n%s\nstderr:\n%s",
				exitError.ExitCode(), runError, stdout, stderr)
		}
		testingHandle.Fatalf("command failed: %v\nstdout:\n%s\nstderr:\n%s", runError, stdout, stderr)
	}

	if strings.Contains(stderr, "Warning:") {
		testingHandle.Logf("command produced warnings:\n%s", stderr)
	}

	return stdout
}

// runCommandExpectError runs the binary expecting a failure and returns combined output.
func runCommandExpectError(testingHandle *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testingHandle.Helper()

	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory

	var buffer bytes.Buffer
	command.Stdout = &buffer
	command.Stderr = &buffer

	runError := command.Run()
	output := buffer.String()

	if runError == nil {
		testingHandle.Fatalf("command succeeded unexpectedly\noutput:\n%s", output)
	}

	return output
}

// runCommandWithWarnings runs the binary and returns stdout while allowing warnings.
func runCommandWithWarnings(testingHandle *testing.T, binaryPath string, arguments []string, workingDirectory string) string {
	testingHandle.Helper()

	command := exec.Command(binaryPath, arguments...)
	command.Dir = workingDirectory

	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer

	runError := command.Run()

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	if runError != nil {
		var exitError *exec.ExitError
		if errors.As(runError, &exitError) {
			testingHandle.Fatalf("command failed when warnings were expected (%d): %v\nstderr:\n%s",
				exitError.ExitCode(), runError, stderr)
		}
		testingHandle.Fatalf("command failed when warnings were expected: %v\nstderr:\n%s", runError, stderr)
	}

	if !strings.Contains(stderr, "Warning:") {
		testingHandle.Fatalf("expected warnings on stderr\nstderr:\n%s", stderr)
	}

	return stdout
}

// setupTestDirectory creates a temporary directory populated with the provided layout.
func setupTestDirectory(testingHandle *testing.T, layout map[string]string) string {
	testingHandle.Helper()

	root := testingHandle.TempDir()

	for relativePath, content := range layout {
		absolutePath := filepath.Join(root, relativePath)

		if strings.HasSuffix(relativePath, "/") || content == "" {
			_ = os.MkdirAll(absolutePath, 0o755)
			continue
		}

		parent := filepath.Dir(absolutePath)
		_ = os.MkdirAll(parent, 0o755)

		if content == "<UNREADABLE>" {
			_ = os.WriteFile(absolutePath, []byte("unreadable placeholder"), 0o644)
			_ = os.Chmod(absolutePath, 0o000)
			continue
		}

		_ = os.WriteFile(absolutePath, []byte(content), 0o644)
	}

	return root
}

// setupGoogleSheetsAddonFixture creates a directory tree resembling a minimal Google Sheets add-on.
// The returned path points to the add-on directory within the temporary workspace.
func setupGoogleSheetsAddonFixture(testingHandle *testing.T) string {
	layout := map[string]string{
		filepath.Join(googleSheetsAddonDirectoryName, gitignoreFileName):                      googleSheetsAddonGitignoreContent,
		filepath.Join(googleSheetsAddonDirectoryName, nodeModulesDirName, dependencyFileName): dependencyFileContent,
		filepath.Join(googleSheetsAddonDirectoryName, claspConfigurationFileName):             claspConfigurationContent,
		filepath.Join(googleSheetsAddonDirectoryName, visibleFileName):                        visibleFileContent,
	}
	root := setupTestDirectory(testingHandle, layout)
	return filepath.Join(root, googleSheetsAddonDirectoryName)
}

// setupExclusionPatternFixture creates a directory tree for exclusion flag tests.
func setupExclusionPatternFixture(testingHandle *testing.T) string {
	layout := map[string]string{
		toolsDirectoryName + "/":  "",
		githubDirectoryName + "/": "",
		yamlRootFileName:          visibleFileContent,
		visibleFileName:           visibleFileContent,
		filepath.Join(nestedDirectoryName, nestedYamlFileName): visibleFileContent,
		filepath.Join(nestedDirectoryName, nestedTextFileName): visibleFileContent,
	}
	return setupTestDirectory(testingHandle, layout)
}

// getModuleRoot returns the repository root directory.
func getModuleRoot(testingHandle *testing.T) string {
	testingHandle.Helper()

	directory, err := os.Getwd()
	if err != nil {
		testingHandle.Fatalf("failed to determine working directory: %v", err)
	}

	for {
		goMod := filepath.Join(directory, "go.mod")
		if _, statErr := os.Stat(goMod); statErr == nil {
			return directory
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			testingHandle.Fatalf("could not locate go.mod from %s", directory)
		}
		directory = parent
	}
}

// TestCTX verifies the ctx CLI across diverse scenarios.
func TestCTX(testingHandle *testing.T) {
	binary := buildBinary(testingHandle)

	var explicitFilePath string

	testCases := []struct {
		name          string
		arguments     []string
		prepare       func(*testing.T) string
		expectError   bool
		expectWarning bool
		validate      func(*testing.T, string)
	}{
		{
			name:      "NoArgumentsDisplaysHelp",
			arguments: nil,
			prepare:   func(testingHandle *testing.T) string { return setupTestDirectory(testingHandle, nil) },
			validate: func(testingHandle *testing.T, output string) {
				if !strings.Contains(output, usageSnippet) {
					testingHandle.Fatalf("expected help output containing %q\n%s", usageSnippet, output)
				}
			},
		},
		{
			name: "DocFlagCallChainRaw",
			arguments: []string{
				"callchain",
				contentDataFunction,
				documentationFlag,
				"--format",
				"raw",
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				if strings.Count(output, "strings.ToLower") == 0 {
					t.Errorf("expected documentation entry for strings.ToLower")
				}
			},
		},
		{
			name: "DefaultFormatTreeCommandJSON",
			arguments: []string{
				appTypes.CommandTree,
				"fileA.txt",
				"directoryB",
			},
			prepare: func(t *testing.T) string {
				return setupTestDirectory(t, map[string]string{
					"fileA.txt":             "A",
					"directoryB/":           "",
					"directoryB/itemB1.txt": "B1",
				})
			},
			validate: func(t *testing.T, output string) {
				var nodes []appTypes.TreeOutputNode
				if err := json.Unmarshal([]byte(output), &nodes); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(nodes) != 2 {
					t.Fatalf("expected two top‑level nodes, got %d", len(nodes))
				}
				var fileNode *appTypes.TreeOutputNode
				for i := range nodes {
					if nodes[i].Name == "fileA.txt" {
						fileNode = &nodes[i]
						break
					}
				}
				if fileNode == nil || fileNode.MimeType != expectedTextMimeType {
					t.Fatalf("expected MIME type %s for fileA.txt", expectedTextMimeType)
				}
			},
		},
		{
			name: "DocumentationFlagTreeUnsupported",
			arguments: []string{
				appTypes.CommandTree,
				documentationFlag,
			},
			prepare: func(t *testing.T) string {
				return setupTestDirectory(t, nil)
			},
			expectError: true,
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, unknownDocumentationFlagErrorSnippet) {
					t.Errorf("expected error for unsupported documentation flag\n%s", output)
				}
			},
		},
		{
			name: "DefaultFormatContentCommandJSON",
			arguments: []string{
				appTypes.CommandContent,
				"fileA.txt",
				"directoryB",
			},
			prepare: func(t *testing.T) string {
				return setupTestDirectory(t, map[string]string{
					"fileA.txt":             "Content A",
					"directoryB/":           "",
					"directoryB/itemB1.txt": "Content B1",
				})
			},
			validate: func(t *testing.T, output string) {
				var files []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &files); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(files) != 2 {
					t.Fatalf("expected two items, got %d", len(files))
				}
				for i := range files {
					if files[i].MimeType != expectedTextMimeType {
						t.Fatalf("expected MIME type %s for %s", expectedTextMimeType, files[i].Path)
					}
				}
			},
		},
		{
			name: "TreeXML",
			arguments: []string{
				appTypes.CommandTree,
				"fileA.txt",
				"--format",
				appTypes.FormatXML,
			},
			prepare: func(t *testing.T) string {
				return setupTestDirectory(t, map[string]string{
					"fileA.txt": "A",
				})
			},
			validate: func(t *testing.T, output string) {
				type resultWrapper struct {
					Nodes []appTypes.TreeOutputNode `xml:"code>item"`
				}
				var wrapper resultWrapper
				if err := xml.Unmarshal([]byte(output), &wrapper); err != nil {
					t.Fatalf("invalid XML: %v\n%s", err, output)
				}
				if len(wrapper.Nodes) != 1 {
					t.Fatalf("expected one top-level node, got %d", len(wrapper.Nodes))
				}
				if wrapper.Nodes[0].MimeType != expectedTextMimeType {
					t.Fatalf("expected MIME type %s", expectedTextMimeType)
				}
			},
		},
		{
			name: "ContentXML",
			arguments: []string{
				appTypes.CommandContent,
				"fileA.txt",
				"--format",
				appTypes.FormatXML,
			},
			prepare: func(t *testing.T) string {
				return setupTestDirectory(t, map[string]string{
					"fileA.txt": "Content A",
				})
			},
			validate: func(t *testing.T, output string) {
				type resultWrapper struct {
					Files []appTypes.FileOutput `xml:"code>item"`
				}
				var wrapper resultWrapper
				if err := xml.Unmarshal([]byte(output), &wrapper); err != nil {
					t.Fatalf("invalid XML: %v\n%s", err, output)
				}
				if len(wrapper.Files) != 1 {
					t.Fatalf("expected one item, got %d", len(wrapper.Files))
				}
				if wrapper.Files[0].MimeType != expectedTextMimeType {
					t.Fatalf("expected MIME type %s", expectedTextMimeType)
				}
			},
		},
		{
			name: "RawFormatExplicitFlag",
			arguments: []string{
				appTypes.CommandContent,
				"onlyfile.txt",
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string {
				dir := setupTestDirectory(t, map[string]string{
					"onlyfile.txt": "Explicit raw content",
				})
				explicitFilePath = filepath.Join(dir, "onlyfile.txt")
				return dir
			},
			validate: func(t *testing.T, output string) {
				if !(strings.Contains(output, "File: "+explicitFilePath) &&
					strings.Contains(output, "Explicit raw content") &&
					strings.Contains(output, "End of file: "+explicitFilePath)) {
					t.Fatalf("unexpected raw content output\n%s", output)
				}
				if strings.Contains(output, mimeTypeIndicator) {
					t.Fatalf("unexpected MIME type in raw output\n%s", output)
				}
			},
		},
		{
			name: "RawFormatExplicitTree",
			arguments: []string{
				appTypes.CommandTree,
				"onlyfile.txt",
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string {
				dir := setupTestDirectory(t, map[string]string{
					"onlyfile.txt": "Explicit raw content",
				})
				explicitFilePath = filepath.Join(dir, "onlyfile.txt")
				return dir
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "[File] "+explicitFilePath) {
					t.Fatalf("unexpected raw tree output\n%s", output)
				}
				if strings.Contains(output, mimeTypeIndicator) {
					t.Fatalf("unexpected MIME type in raw output\n%s", output)
				}
			},
		},
		{
			name: "CallChainRaw",
			arguments: []string{
				appTypes.CommandCallChain,
				contentDataFunction,
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "Target Function: "+contentDataFunction) {
					t.Fatalf("missing target function in output")
				}
			},
		},
		{
			name: "CallChainJSON",
			arguments: []string{
				appTypes.CommandCallChain,
				contentDataFunction,
				"--format",
				appTypes.FormatJSON,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				var list []appTypes.CallChainOutput
				if err := json.Unmarshal([]byte(output), &list); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(list) == 0 {
					t.Fatalf("expected at least one element, got zero")
				}
				chain := list[0]
				if chain.TargetFunction != contentDataFunction {
					t.Fatalf("unexpected target function %q", chain.TargetFunction)
				}
			},
		},
		{
			name: "CallChainXML",
			arguments: []string{
				appTypes.CommandCallChain,
				contentDataFunction,
				"--format",
				appTypes.FormatXML,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				type callChainsWrapper struct {
					XMLName xml.Name `xml:"callchains"`
					Chains  []struct {
						TargetFunction string `xml:"targetFunction"`
					} `xml:"callchain"`
				}
				var wrapper callChainsWrapper
				if err := xml.Unmarshal([]byte(output), &wrapper); err != nil {
					t.Fatalf("invalid XML: %v\n%s", err, output)
				}
				if len(wrapper.Chains) == 0 {
					t.Fatalf("expected at least one element, got zero")
				}
				if wrapper.Chains[0].TargetFunction != contentDataFunction {
					t.Fatalf("unexpected target function %q", wrapper.Chains[0].TargetFunction)
				}
			},
		},
		{
			name: "CallChainAliasDefaultDepthReturnsOnlyDirectCallers",
			arguments: []string{
				callChainAlias,
				contentDataFunction,
				formatFlag,
				appTypes.FormatJSON,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				var callChains []appTypes.CallChainOutput
				if err := json.Unmarshal([]byte(output), &callChains); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(callChains) != 1 {
					t.Fatalf("expected one call chain, got %d", len(callChains))
				}
				callers := callChains[0].Callers
				if len(callers) != 1 {
					t.Fatalf("expected one direct caller, got %d", len(callers))
				}
				if callers[0] != runTreeOrContentCommandFunction {
					t.Fatalf("expected caller %s, got %s", runTreeOrContentCommandFunction, callers[0])
				}
			},
		},
		{
			name: "CallChainAliasDepthTwoIncludesSecondLevelCallers",
			arguments: []string{
				callChainAlias,
				contentDataFunction,
				depthFlag,
				depthTwoValue,
				formatFlag,
				appTypes.FormatJSON,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				var callChains []appTypes.CallChainOutput
				if err := json.Unmarshal([]byte(output), &callChains); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(callChains) != 1 {
					t.Fatalf("expected one call chain, got %d", len(callChains))
				}
				callers := callChains[0].Callers
				if len(callers) != 2 {
					t.Fatalf("expected two callers, got %d", len(callers))
				}
				hasDirectCaller := false
				hasSecondLevelCaller := false
				for _, caller := range callers {
					if caller == runTreeOrContentCommandFunction {
						hasDirectCaller = true
					}
					if caller == runToolFunction {
						hasSecondLevelCaller = true
					}
				}
				if !hasDirectCaller {
					t.Fatalf("missing caller %s", runTreeOrContentCommandFunction)
				}
				if !hasSecondLevelCaller {
					t.Fatalf("missing caller %s", runToolFunction)
				}
			},
		},
		{
			name:          "VersionFlag",
			arguments:     []string{versionFlag},
			prepare:       func(t *testing.T) string { return setupTestDirectory(t, nil) },
			expectError:   false,
			expectWarning: false,
			validate: func(t *testing.T, output string) {
				const prefix = "ctx version:"
				if !strings.HasPrefix(output, prefix) {
					t.Fatalf("version output should start with %q\n%s", prefix, output)
				}
			},
		},
		{
			name: "JsonFormatContentUnreadableFile",
			arguments: []string{
				appTypes.CommandContent,
				"readable.txt",
				"unreadable.txt",
				"--format",
				appTypes.FormatJSON,
			},
			prepare: func(t *testing.T) string {
				if runtime.GOOS == "windows" || os.Geteuid() == 0 {
					t.Skip("Skipping unreadable file test on this platform")
				}
				return setupTestDirectory(t, map[string]string{
					"readable.txt":   "OK",
					"unreadable.txt": "<UNREADABLE>",
				})
			},
			expectWarning: true,
			validate: func(t *testing.T, output string) {
				var files []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &files); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(files) != 1 {
					t.Fatalf("expected one readable item, got %d", len(files))
				}
				if files[0].MimeType != expectedTextMimeType {
					t.Fatalf("expected MIME type %s", expectedTextMimeType)
				}
			},
		},
		{
			name: "BinaryFileContentJSON",
			arguments: []string{
				appTypes.CommandContent,
				".",
			},
			prepare: func(t *testing.T) string {
				binaryBytes, decodeError := base64.StdEncoding.DecodeString(onePixelPNGBase64Content)
				if decodeError != nil {
					t.Fatalf("failed to decode binary fixture: %v", decodeError)
				}
				return setupTestDirectory(t, map[string]string{
					binaryFixtureFileName: string(binaryBytes),
				})
			},
			validate: func(t *testing.T, output string) {
				var files []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &files); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(files) != 1 {
					t.Fatalf("expected one item, got %d", len(files))
				}
				fileOutput := files[0]
				if fileOutput.Type != appTypes.NodeTypeBinary {
					t.Fatalf("expected type %q, got %q", appTypes.NodeTypeBinary, fileOutput.Type)
				}
				if fileOutput.Content != "" {
					t.Fatalf("expected empty content for binary file, got %q", fileOutput.Content)
				}
				if fileOutput.MimeType != expectedBinaryMimeType {
					t.Fatalf("expected MIME type %q, got %q", expectedBinaryMimeType, fileOutput.MimeType)
				}
			},
		},
		{
			name: "BinaryFileTreeJSON",
			arguments: []string{
				appTypes.CommandTree,
				".",
			},
			prepare: func(t *testing.T) string {
				binaryBytes, decodeError := base64.StdEncoding.DecodeString(onePixelPNGBase64Content)
				if decodeError != nil {
					t.Fatalf("failed to decode binary fixture: %v", decodeError)
				}
				return setupTestDirectory(t, map[string]string{
					binaryFixtureFileName: string(binaryBytes),
				})
			},
			validate: func(t *testing.T, output string) {
				var nodes []appTypes.TreeOutputNode
				if err := json.Unmarshal([]byte(output), &nodes); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(nodes) != 1 {
					t.Fatalf("expected one top-level node, got %d", len(nodes))
				}
				if len(nodes[0].Children) != 1 {
					t.Fatalf("expected one child, got %d", len(nodes[0].Children))
				}
				child := nodes[0].Children[0]
				if child.Type != appTypes.NodeTypeBinary {
					t.Fatalf("expected type %q, got %q", appTypes.NodeTypeBinary, child.Type)
				}
				if child.MimeType != expectedBinaryMimeType {
					t.Fatalf("expected MIME type %q, got %q", expectedBinaryMimeType, child.MimeType)
				}
			},
		},
		{
			name: "BinaryFileContentIgnoreDefault",
			arguments: []string{
				appTypes.CommandContent,
				".",
			},
			prepare: func(t *testing.T) string {
				binaryBytes, decodeError := base64.StdEncoding.DecodeString(onePixelPNGBase64Content)
				if decodeError != nil {
					t.Fatalf("failed to decode binary fixture: %v", decodeError)
				}
				ignoreContent := binarySectionHeader + "\n" + unmatchedBinaryFixtureName + "\n"
				return setupTestDirectory(t, map[string]string{
					binaryFixtureFileName: string(binaryBytes),
					ignoreFileName:        ignoreContent,
				})
			},
			validate: func(t *testing.T, output string) {
				var files []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &files); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(files) != 1 {
					t.Fatalf("expected one item, got %d", len(files))
				}
				fileOutput := files[0]
				if fileOutput.Content != "" {
					t.Fatalf("expected empty content for binary file, got %q", fileOutput.Content)
				}
				if fileOutput.MimeType != expectedBinaryMimeType {
					t.Fatalf("expected MIME type %q, got %q", expectedBinaryMimeType, fileOutput.MimeType)
				}
			},
		},
		{
			name: "BinaryFileContentBinarySection",
			arguments: []string{
				appTypes.CommandContent,
				".",
			},
			prepare: func(t *testing.T) string {
				binaryBytes, decodeError := base64.StdEncoding.DecodeString(onePixelPNGBase64Content)
				if decodeError != nil {
					t.Fatalf("failed to decode binary fixture: %v", decodeError)
				}
				ignoreContent := binarySectionHeader + "\n" + binaryFixtureFileName + "\n"
				return setupTestDirectory(t, map[string]string{
					binaryFixtureFileName: string(binaryBytes),
					ignoreFileName:        ignoreContent,
				})
			},
			validate: func(t *testing.T, output string) {
				var files []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &files); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(files) != 1 {
					t.Fatalf("expected one item, got %d", len(files))
				}
				fileOutput := files[0]
				if fileOutput.Content != onePixelPNGBase64Content {
					t.Fatalf("expected base64 content for binary file")
				}
				if fileOutput.MimeType != expectedBinaryMimeType {
					t.Fatalf("expected MIME type %q, got %q", expectedBinaryMimeType, fileOutput.MimeType)
				}
			},
		},
		{
			name: "InvalidFormatValue",
			arguments: []string{
				appTypes.CommandContent,
				"a.txt",
				"--format",
				"yaml",
			},
			prepare:     func(t *testing.T) string { return setupTestDirectory(t, map[string]string{"a.txt": "A"}) },
			expectError: true,
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "Invalid format value 'yaml'") {
					t.Errorf("expected error about invalid format value, got:\n%s", output)
				}
			},
		},
		{
			name: "CallChainMainDoc",
			arguments: []string{
				appTypes.CommandCallChain,
				"main.main",
				documentationFlag,
				"--format",
				appTypes.FormatRaw,
			},
			prepare: func(t *testing.T) string { return getModuleRoot(t) },
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "main.main") {
					t.Fatalf("expected main.main documentation in output, got:\n%s", output)
				}

				for _, line := range strings.Split(output, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "Error:") {
						t.Fatalf("unexpected error message in output:\n%s", output)
					}
				}
			},
		},
		{
			name: "GitDirectoryExcludedByDefault",
			arguments: []string{
				appTypes.CommandTree,
			},
			prepare: func(t *testing.T) string {
				layout := map[string]string{
					filepath.Join(utils.GitDirectoryName, hiddenFileName): hiddenFileContent,
					visibleFileName: visibleFileContent,
				}
				return setupTestDirectory(t, layout)
			},
			validate: func(t *testing.T, output string) {
				var nodes []appTypes.TreeOutputNode
				if err := json.Unmarshal([]byte(output), &nodes); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(nodes) != 1 {
					t.Fatalf("expected one root node, got %d", len(nodes))
				}
				children := nodes[0].Children
				if len(children) != 1 || children[0].Name != visibleFileName {
					t.Fatalf("expected only %s, got %#v", visibleFileName, children)
				}
				if children[0].MimeType != expectedTextMimeType {
					t.Fatalf("expected MIME type %s", expectedTextMimeType)
				}
			},
		},
		{
			name: "GitDirectoryIncludedWithFlag",
			arguments: []string{
				appTypes.CommandTree,
				includeGitFlag,
			},
			prepare: func(t *testing.T) string {
				layout := map[string]string{
					filepath.Join(utils.GitDirectoryName, hiddenFileName): hiddenFileContent,
					visibleFileName: visibleFileContent,
				}
				return setupTestDirectory(t, layout)
			},
			validate: func(t *testing.T, output string) {
				var nodes []appTypes.TreeOutputNode
				if err := json.Unmarshal([]byte(output), &nodes); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(nodes) != 1 {
					t.Fatalf("expected one root node, got %d", len(nodes))
				}
				names := make(map[string]*appTypes.TreeOutputNode)
				for _, child := range nodes[0].Children {
					names[child.Name] = child
				}
				if _, ok := names[utils.GitDirectoryName]; !ok {
					t.Fatalf("expected %s in output", utils.GitDirectoryName)
				}
				visibleNode, ok := names[visibleFileName]
				if !ok {
					t.Fatalf("expected %s in output", visibleFileName)
				}
				if visibleNode.MimeType != expectedTextMimeType {
					t.Fatalf("expected MIME type %s", expectedTextMimeType)
				}
			},
		},
		{
			name:      "GoogleSheetsAddonTreeOmitsIgnoredEntries",
			arguments: []string{appTypes.CommandTree},
			prepare:   setupGoogleSheetsAddonFixture,
			validate: func(testingHandle *testing.T, output string) {
				if strings.Contains(output, nodeModulesDirName) {
					testingHandle.Fatalf("expected tree output to exclude %s\n%s", nodeModulesDirName, output)
				}
				if strings.Contains(output, claspConfigurationFileName) {
					testingHandle.Fatalf("expected tree output to exclude %s\n%s", claspConfigurationFileName, output)
				}
				if !strings.Contains(output, visibleFileName) {
					testingHandle.Fatalf("expected tree output to include %s\n%s", visibleFileName, output)
				}
				if !strings.Contains(output, expectedTextMimeType) {
					testingHandle.Fatalf("expected MIME type %s in output\n%s", expectedTextMimeType, output)
				}
			},
		},
		{
			name:      "GoogleSheetsAddonContentOmitsIgnoredEntries",
			arguments: []string{appTypes.CommandContent},
			prepare:   setupGoogleSheetsAddonFixture,
			validate: func(testingHandle *testing.T, output string) {
				if strings.Contains(output, dependencyFileName) || strings.Contains(output, dependencyFileContent) {
					testingHandle.Fatalf("expected content output to exclude %s\n%s", dependencyFileName, output)
				}
				if strings.Contains(output, claspConfigurationFileName) || strings.Contains(output, claspConfigurationContent) {
					testingHandle.Fatalf("expected content output to exclude %s\n%s", claspConfigurationFileName, output)
				}
				if !strings.Contains(output, visibleFileName) || !strings.Contains(output, visibleFileContent) {
					testingHandle.Fatalf("expected content output to include %s and its content\n%s", visibleFileName, output)
				}
				if !strings.Contains(output, expectedTextMimeType) {
					testingHandle.Fatalf("expected MIME type %s in output\n%s", expectedTextMimeType, output)
				}
			},
		},
		{
			name:      "NestedGitignoreExcluded",
			arguments: []string{treeAlias},
			prepare: func(t *testing.T) string {
				layout := map[string]string{
					filepath.Join(subDirectoryName, gitignoreFileName):                      nodeModulesPattern,
					filepath.Join(subDirectoryName, nodeModulesDirName, dependencyFileName): dependencyFileContent,
					filepath.Join(subDirectoryName, visibleFileName):                        visibleFileContent,
				}
				return setupTestDirectory(t, layout)
			},
			validate: func(t *testing.T, output string) {
				var nodes []appTypes.TreeOutputNode
				if err := json.Unmarshal([]byte(output), &nodes); err != nil {
					t.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(nodes) != 1 {
					t.Fatalf("expected one root node, got %d", len(nodes))
				}
				rootChildren := nodes[0].Children
				if len(rootChildren) != 1 || rootChildren[0].Name != subDirectoryName {
					t.Fatalf("expected root child %s, got %#v", subDirectoryName, rootChildren)
				}
				subNode := rootChildren[0]
				if len(subNode.Children) != 1 || subNode.Children[0].Name != visibleFileName {
					t.Fatalf("expected only %s in %s, got %#v", visibleFileName, subDirectoryName, subNode.Children)
				}
				if subNode.Children[0].MimeType != expectedTextMimeType {
					t.Fatalf("expected MIME type %s", expectedTextMimeType)
				}
			},
		},
		{
			name:      "NestedGitignoreContentExcluded",
			arguments: []string{contentAlias},
			prepare: func(testingHandle *testing.T) string {
				directoryLayout := map[string]string{
					filepath.Join(subDirectoryName, gitignoreFileName):                      nodeModulesPattern + ignoredFileName + "\n",
					filepath.Join(subDirectoryName, nodeModulesDirName, dependencyFileName): dependencyFileContent,
					filepath.Join(subDirectoryName, ignoredFileName):                        ignoredFileContent,
					filepath.Join(subDirectoryName, visibleFileName):                        visibleFileContent,
				}
				return setupTestDirectory(testingHandle, directoryLayout)
			},
			validate: func(testingHandle *testing.T, output string) {
				if strings.Contains(output, nodeModulesDirName) {
					testingHandle.Fatalf("expected content output to exclude %s\n%s", nodeModulesDirName, output)
				}
				if strings.Contains(output, ignoredFileName) {
					testingHandle.Fatalf("expected content output to exclude %s\n%s", ignoredFileName, output)
				}
				var fileOutputs []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &fileOutputs); err != nil {
					testingHandle.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(fileOutputs) != 1 {
					testingHandle.Fatalf("expected one file, got %d", len(fileOutputs))
				}
				expectedSuffix := filepath.Join(subDirectoryName, visibleFileName)
				fileOutput := fileOutputs[0]
				if !strings.HasSuffix(fileOutput.Path, expectedSuffix) {
					testingHandle.Fatalf("expected path suffix %s, got %s", expectedSuffix, fileOutput.Path)
				}
				if fileOutput.Content != visibleFileContent {
					testingHandle.Fatalf("expected content %s, got %s", visibleFileContent, fileOutput.Content)
				}
				if fileOutput.MimeType != expectedTextMimeType {
					testingHandle.Fatalf("expected MIME type %s", expectedTextMimeType)
				}
			},
		},
		{
			name:      "ExcludePatternsTree",
			arguments: []string{appTypes.CommandTree, "-e", toolsDirectoryName, "-e", githubDirectoryName, "-e", yamlPattern},
			prepare:   setupExclusionPatternFixture,
			validate: func(testingHandle *testing.T, output string) {
				var outputNodes []appTypes.TreeOutputNode
				if err := json.Unmarshal([]byte(output), &outputNodes); err != nil {
					testingHandle.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(outputNodes) != 1 {
					testingHandle.Fatalf("expected one root node, got %d", len(outputNodes))
				}
				rootChildren := outputNodes[0].Children
				if len(rootChildren) != 2 {
					testingHandle.Fatalf("expected two root children, got %d", len(rootChildren))
				}
				for _, childNode := range rootChildren {
					if childNode.Name == toolsDirectoryName || childNode.Name == githubDirectoryName || childNode.Name == yamlRootFileName {
						testingHandle.Fatalf("unexpected child %s", childNode.Name)
					}
					if childNode.Name == nestedDirectoryName {
						if len(childNode.Children) != 1 || childNode.Children[0].Name != nestedTextFileName {
							testingHandle.Fatalf("expected only %s inside %s", nestedTextFileName, nestedDirectoryName)
						}
						if childNode.Children[0].MimeType != expectedTextMimeType {
							testingHandle.Fatalf("expected MIME type %s", expectedTextMimeType)
						}
					}
				}
			},
		},
		{
			name:      "ExcludePatternsContent",
			arguments: []string{appTypes.CommandContent, "-e", toolsDirectoryName, "-e", githubDirectoryName, "-e", yamlPattern},
			prepare:   setupExclusionPatternFixture,
			validate: func(testingHandle *testing.T, output string) {
				var fileOutputs []appTypes.FileOutput
				if err := json.Unmarshal([]byte(output), &fileOutputs); err != nil {
					testingHandle.Fatalf("invalid JSON: %v\n%s", err, output)
				}
				if len(fileOutputs) != 2 {
					testingHandle.Fatalf("expected two files, got %d", len(fileOutputs))
				}
				for _, fileOutput := range fileOutputs {
					if strings.HasSuffix(fileOutput.Path, yamlRootFileName) || strings.HasSuffix(fileOutput.Path, nestedYamlFileName) {
						testingHandle.Fatalf("unexpected YAML file %s", fileOutput.Path)
					}
					if strings.Contains(fileOutput.Path, toolsDirectoryName) || strings.Contains(fileOutput.Path, githubDirectoryName) {
						testingHandle.Fatalf("unexpected excluded path %s", fileOutput.Path)
					}
					if fileOutput.MimeType != expectedTextMimeType {
						testingHandle.Fatalf("expected MIME type %s", expectedTextMimeType)
					}
				}
			},
		},
	}

	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(t *testing.T) {
			workingDir := testCase.prepare(t)

			var output string
			if testCase.expectError {
				output = runCommandExpectError(t, binary, testCase.arguments, workingDir)
			} else if testCase.expectWarning {
				output = runCommandWithWarnings(t, binary, testCase.arguments, workingDir)
			} else {
				output = runCommand(t, binary, testCase.arguments, workingDir)
			}

			testCase.validate(t, output)
		})
	}
}
