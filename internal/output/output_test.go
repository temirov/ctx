package output_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

// textMimeTypeExpected defines the MIME type expected for text files.
const textMimeTypeExpected = "text/plain; charset=utf-8"

var sampleLastModified = time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

const sampleFileSize int64 = 123

const sampleFileSizeHuman = "123b"

var sampleLastModifiedFormatted = utils.FormatTimestamp(sampleLastModified)

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
var jsonExpected = "{\n" +
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
	"      \"size\": \"" + sampleFileSizeHuman + "\",\n" +
	"      \"lastModified\": \"" + sampleLastModifiedFormatted + "\",\n" +
	"      \"mimeType\": \"" + textMimeTypeExpected + "\"\n" +
	"    }\n" +
	"  ]\n" +
	"}"

// renderJSONErrorMessage defines the error message for JSON rendering failures.
const renderJSONErrorMessage = "render json error"

// TestRenderJSON verifies RenderJSON output and deduplication.
func TestRenderJSON(testingInstance *testing.T) {
	docs := []types.DocumentationEntry{{Kind: "kind", Name: "name", Doc: "doc"}, {Kind: "kind", Name: "name", Doc: "doc"}}
	items := []interface{}{
		&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", Size: sampleFileSizeHuman, LastModified: sampleLastModifiedFormatted, MimeType: textMimeTypeExpected},
		&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", Size: sampleFileSizeHuman, LastModified: sampleLastModifiedFormatted, MimeType: textMimeTypeExpected},
	}
	actual, renderJSONError := output.RenderJSON(docs, items)
	if renderJSONError != nil {
		testingInstance.Fatalf("%s: %v", renderJSONErrorMessage, renderJSONError)
	}
	if actual != jsonExpected {
		testingInstance.Errorf("unexpected output: %q", actual)
	}
}

// xmlExpected defines the expected XML rendering.
var xmlExpected = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
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
	"      <size>" + sampleFileSizeHuman + "</size>\n" +
	"      <lastModified>" + sampleLastModifiedFormatted + "</lastModified>\n" +
	"      <mimeType>" + textMimeTypeExpected + "</mimeType>\n" +
	"      <documentation></documentation>\n" +
	"    </item>\n" +
	"  </code>\n" +
	"</result>"

// renderXMLErrorMessage defines the error message for XML rendering failures.
const renderXMLErrorMessage = "render xml error"

// TestRenderXML verifies RenderXML output and deduplication.
func TestRenderXML(testingInstance *testing.T) {
	docs := []types.DocumentationEntry{{Kind: "kind", Name: "name", Doc: "doc"}, {Kind: "kind", Name: "name", Doc: "doc"}}
	items := []interface{}{
		&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", Size: sampleFileSizeHuman, LastModified: sampleLastModifiedFormatted, MimeType: textMimeTypeExpected},
		&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", Size: sampleFileSizeHuman, LastModified: sampleLastModifiedFormatted, MimeType: textMimeTypeExpected},
	}
	actual, renderXMLError := output.RenderXML(docs, items)
	if renderXMLError != nil {
		testingInstance.Fatalf("%s: %v", renderXMLErrorMessage, renderXMLError)
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

// pipeCreationErrorMessage defines the error message for pipe creation failures.
const pipeCreationErrorMessage = "pipe creation error"

// renderRawErrorMessage defines the error message for raw rendering failures.
const renderRawErrorMessage = "render raw error"

// bufferReadErrorMessage defines the error message for buffer read failures.
const bufferReadErrorMessage = "buffer read error"

// TestRenderRaw verifies RenderRaw printing.
func TestRenderRaw(testingInstance *testing.T) {
	docs := []types.DocumentationEntry{{Kind: "kind", Name: "name", Doc: "doc"}}
	items := []interface{}{
		&types.FileOutput{Path: "file.txt", Type: types.NodeTypeFile, Content: "data", Size: sampleFileSizeHuman, LastModified: sampleLastModifiedFormatted, MimeType: textMimeTypeExpected},
	}
	reader, writer, pipeCreationError := os.Pipe()
	if pipeCreationError != nil {
		testingInstance.Fatalf("%s: %v", pipeCreationErrorMessage, pipeCreationError)
	}
	originalStdout := os.Stdout
	os.Stdout = writer
	renderRawError := output.RenderRaw(types.CommandContent, docs, items)
	writer.Close()
	os.Stdout = originalStdout
	if renderRawError != nil {
		testingInstance.Fatalf("%s: %v", renderRawErrorMessage, renderRawError)
	}
	var buffer bytes.Buffer
	_, bufferReadError := buffer.ReadFrom(reader)
	if bufferReadError != nil {
		testingInstance.Fatalf("%s: %v", bufferReadErrorMessage, bufferReadError)
	}
	actual := buffer.String()
	if actual != rawExpected {
		testingInstance.Errorf("unexpected output: %q", actual)
	}
}
