package output_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/types"
)

// textMimeTypeExpected defines the MIME type expected for text files.
const textMimeTypeExpected = "text/plain; charset=utf-8"

// callChainRawExpected defines the expected raw rendering of a call chain.
const callChainRawExpected = "----- CALLCHAIN METADATA -----\n" +
	"Target Function: target\n" +
	"Callers:\n" +
	" caller\n" +
	"Callees:\n" +
	" callee\n\n" +
	"----- FUNCTIONS -----\n" +
	"Function: target\n" +
	"----------------------------------------\n" +
	"body\n" +
	"----------------------------------------\n\n" +
	"Function: caller\n" +
	"----------------------------------------\n" +
	"\n" +
	"----------------------------------------\n\n" +
	"Function: callee\n" +
	"----------------------------------------\n" +
	"\n" +
	"----------------------------------------\n\n" +
	"--- DOCS ---\n" +
	"kind target\n" +
	"documentation\n\n"

// TestRenderCallChainRaw verifies RenderCallChainRaw output.
func TestRenderCallChainRaw(testingInstance *testing.T) {
	callees := []string{"callee"}
	data := &types.CallChainOutput{
		TargetFunction: "target",
		Callers:        []string{"caller"},
		Callees:        &callees,
		Functions:      map[string]string{"target": "body"},
		Documentation:  []types.DocumentationEntry{{Kind: "kind", Name: "target", Doc: "documentation"}},
	}
	actual := output.RenderCallChainRaw(data)
	if actual != callChainRawExpected {
		testingInstance.Errorf("unexpected output: %q", actual)
	}
}

// jsonExpected defines the expected JSON rendering.
const jsonExpected = "{\n" +
	"  \"documentation\": [\n" +
	"    {\n" +
	"      \"type\": \"kind\",\n" +
	"      \"name\": \"name\",\n" +
	"      \"documentation\": \"doc\"\n" +
	"    }\n" +
	"  ],\n" +
	"  \"code\": [\n" +
	"    {\n" +
	"      \"path\": \"file.txt\",\n" +
	"      \"type\": \"file\",\n" +
	"      \"content\": \"data\",\n" +
	"      \"mimeType\": \"" + textMimeTypeExpected + "\"\n" +
	"    }\n" +
	"  ]\n" +
	"}"

// TestRenderJSON verifies RenderJSON output and deduplication.
func TestRenderJSON(testingInstance *testing.T) {
	docs := []types.DocumentationEntry{{Kind: "kind", Name: "name", Doc: "doc"}, {Kind: "kind", Name: "name", Doc: "doc"}}
	items := []interface{}{&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", MimeType: textMimeTypeExpected}, &types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", MimeType: textMimeTypeExpected}}
	actual, err := output.RenderJSON(docs, items)
	if err != nil {
		testingInstance.Fatalf("render json error: %v", err)
	}
	if actual != jsonExpected {
		testingInstance.Errorf("unexpected output: %q", actual)
	}
}

// xmlExpected defines the expected XML rendering.
const xmlExpected = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
	"<result>\n" +
	"  <documentation>\n" +
	"    <entry>\n" +
	"      <type>kind</type>\n" +
	"      <name>name</name>\n" +
	"      <documentation>doc</documentation>\n" +
	"    </entry>\n" +
	"  </documentation>\n" +
	"  <code>\n" +
	"    <item>\n" +
	"      <path>file.txt</path>\n" +
	"      <type>file</type>\n" +
	"      <content>data</content>\n" +
	"      <mimeType>" + textMimeTypeExpected + "</mimeType>\n" +
	"      <documentation></documentation>\n" +
	"    </item>\n" +
	"  </code>\n" +
	"</result>"

// TestRenderXML verifies RenderXML output and deduplication.
func TestRenderXML(testingInstance *testing.T) {
	docs := []types.DocumentationEntry{{Kind: "kind", Name: "name", Doc: "doc"}, {Kind: "kind", Name: "name", Doc: "doc"}}
	items := []interface{}{&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", MimeType: textMimeTypeExpected}, &types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", MimeType: textMimeTypeExpected}}
	actual, err := output.RenderXML(docs, items)
	if err != nil {
		testingInstance.Fatalf("render xml error: %v", err)
	}
	if actual != xmlExpected {
		testingInstance.Errorf("unexpected output: %q", actual)
	}
}

// rawExpected defines the expected raw output for documentation and content.
const rawExpected = "--- Documentation ---\n" +
	"kind name\n" +
	"doc\n\n" +
	"File: file.txt\n" +
	"data\n" +
	"End of file: file.txt\n" +
	"----------------------------------------\n"

// TestRenderRaw verifies RenderRaw printing.
func TestRenderRaw(testingInstance *testing.T) {
	docs := []types.DocumentationEntry{{Kind: "kind", Name: "name", Doc: "doc"}}
	items := []interface{}{&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", MimeType: textMimeTypeExpected}}
	reader, writer, pipeError := os.Pipe()
	if pipeError != nil {
		testingInstance.Fatalf("pipe error: %v", pipeError)
	}
	originalStdout := os.Stdout
	os.Stdout = writer
	err := output.RenderRaw(types.CommandContent, docs, items)
	writer.Close()
	os.Stdout = originalStdout
	if err != nil {
		testingInstance.Fatalf("render raw error: %v", err)
	}
	var buffer bytes.Buffer
	_, readError := buffer.ReadFrom(reader)
	if readError != nil {
		testingInstance.Fatalf("read error: %v", readError)
	}
	actual := buffer.String()
	if actual != rawExpected {
		testingInstance.Errorf("unexpected output: %q", actual)
	}
}
