package output_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/types"
)

const (
	callChainTarget       = "alpha"
	callChainCaller       = "beta"
	callChainCallee       = "gamma"
	callChainFunctionBody = "body"
	documentationKind     = "func"
	documentationName     = "alpha"
	documentationDoc      = "doc"
	filePath              = "file.txt"
	fileContent           = "content"
	jsonExpected          = "{\n  \"documentation\": [\n    {\n      \"type\": \"var\",\n      \"name\": \"x\",\n      \"documentation\": \"d\"\n    }\n  ],\n  \"code\": [\n    {\n      \"path\": \"file.txt\",\n      \"type\": \"file\",\n      \"content\": \"content\"\n    }\n  ]\n}"
	xmlExpected           = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<result>\n  <documentation>\n    <entry>\n      <type>var</type>\n      <name>x</name>\n      <documentation>d</documentation>\n    </entry>\n  </documentation>\n  <code>\n    <item>\n      <path>file.txt</path>\n      <type>file</type>\n      <content>content</content>\n      <documentation></documentation>\n    </item>\n  </code>\n</result>"
	rawExpected           = "--- Documentation ---\nfunc alpha\ndoc\n\nFile: file.txt\ncontent\nEnd of file: file.txt\n----------------------------------------\n"
)

// TestRenderCallChainRaw verifies raw callchain rendering.
func TestRenderCallChainRaw(testingHandle *testing.T) {
	callees := []string{callChainCallee}
	callChain := &types.CallChainOutput{
		TargetFunction: callChainTarget,
		Callers:        []string{callChainCaller},
		Callees:        &callees,
		Functions: map[string]string{
			callChainTarget: callChainFunctionBody,
			callChainCaller: callChainFunctionBody,
			callChainCallee: callChainFunctionBody,
		},
		Documentation: []types.DocumentationEntry{
			{Kind: documentationKind, Name: documentationName, Doc: documentationDoc},
		},
	}
	rendered := output.RenderCallChainRaw(callChain)
	expected := "----- CALLCHAIN METADATA -----\n" +
		"Target Function: " + callChainTarget + "\n" +
		"Callers:\n " + callChainCaller + "\n" +
		"Callees:\n " + callChainCallee + "\n\n" +
		"----- FUNCTIONS -----\n" +
		"Function: " + callChainTarget + "\n" +
		"----------------------------------------\n" + callChainFunctionBody + "\n" +
		"----------------------------------------\n\n" +
		"Function: " + callChainCaller + "\n" +
		"----------------------------------------\n" + callChainFunctionBody + "\n" +
		"----------------------------------------\n\n" +
		"Function: " + callChainCallee + "\n" +
		"----------------------------------------\n" + callChainFunctionBody + "\n" +
		"----------------------------------------\n\n" +
		"--- DOCS ---\n" +
		documentationKind + " " + documentationName + "\n" +
		documentationDoc + "\n\n"
	if rendered != expected {
		testingHandle.Fatalf("unexpected output: %q", rendered)
	}
}

// TestRenderJSON verifies JSON rendering and deduplication.
func TestRenderJSON(testingHandle *testing.T) {
	documentationEntries := []types.DocumentationEntry{
		{Kind: "var", Name: "x", Doc: "d"},
		{Kind: "var", Name: "x", Doc: "d"},
	}
	items := []interface{}{
		&types.FileOutput{Path: filePath, Type: types.NodeTypeFile, Content: fileContent},
		&types.FileOutput{Path: filePath, Type: types.NodeTypeFile, Content: fileContent},
	}
	rendered, renderError := output.RenderJSON(documentationEntries, items)
	if renderError != nil {
		testingHandle.Fatalf("RenderJSON error: %v", renderError)
	}
	if rendered != jsonExpected {
		testingHandle.Fatalf("unexpected JSON: %s", rendered)
	}
}

// TestRenderXML verifies XML rendering and deduplication.
func TestRenderXML(testingHandle *testing.T) {
	documentationEntries := []types.DocumentationEntry{
		{Kind: "var", Name: "x", Doc: "d"},
		{Kind: "var", Name: "x", Doc: "d"},
	}
	items := []interface{}{
		&types.FileOutput{Path: filePath, Type: types.NodeTypeFile, Content: fileContent},
		&types.FileOutput{Path: filePath, Type: types.NodeTypeFile, Content: fileContent},
	}
	rendered, renderError := output.RenderXML(documentationEntries, items)
	if renderError != nil {
		testingHandle.Fatalf("RenderXML error: %v", renderError)
	}
	if rendered != xmlExpected {
		testingHandle.Fatalf("unexpected XML: %s", rendered)
	}
}

// TestRenderRaw verifies raw output rendering.
func TestRenderRaw(testingHandle *testing.T) {
	documentationEntries := []types.DocumentationEntry{
		{Kind: documentationKind, Name: documentationName, Doc: documentationDoc},
		{Kind: documentationKind, Name: documentationName, Doc: documentationDoc},
	}
	items := []interface{}{
		&types.FileOutput{Path: filePath, Type: types.NodeTypeFile, Content: fileContent},
		&types.FileOutput{Path: filePath, Type: types.NodeTypeFile, Content: fileContent},
	}
	originalStdout := os.Stdout
	readPipe, writePipe, pipeError := os.Pipe()
	if pipeError != nil {
		testingHandle.Fatalf("pipe error: %v", pipeError)
	}
	os.Stdout = writePipe
	renderError := output.RenderRaw(types.CommandContent, documentationEntries, items)
	writePipe.Close()
	os.Stdout = originalStdout
	if renderError != nil {
		testingHandle.Fatalf("RenderRaw error: %v", renderError)
	}
	captured, readError := io.ReadAll(readPipe)
	if readError != nil {
		testingHandle.Fatalf("read error: %v", readError)
	}
	trimmed := strings.ReplaceAll(string(captured), "\r", "")
	if trimmed != rawExpected {
		testingHandle.Fatalf("unexpected raw output: %q", trimmed)
	}
}
