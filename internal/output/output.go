package output

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/temirov/ctx/internal/types"
)

const (
	indentPrefix = ""
	indentSpacer = "  "

	separatorLine         = "----------------------------------------"
	documentationHeader   = "--- Documentation ---"
	callchainMetaHeader   = "----- CALLCHAIN METADATA -----"
	callchainFunctionsHdr = "----- FUNCTIONS -----"
	callchainDocsHeader   = "--- DOCS ---"

	xmlHeader         = xml.Header
	xmlRootElement    = "result"
	xmlCallchainsName = "callchains"

	binaryContentOmitted = "(binary content omitted)"
	mimeTypeLabel        = "Mime Type: "
	binaryNodeFormat     = "[Binary] %s (%s%s)\n"
	binaryTreeFormat     = "%s[Binary] %s (%s%s)\n"

	// filePrefix is the key prefix used when deduplicating file outputs.
	filePrefix = "file:"
	// nodePrefix is the key prefix used when deduplicating tree nodes.
	nodePrefix = "node:"
	// callChainPrefix is the key prefix used when deduplicating call chain outputs.
	callChainPrefix = "callchain:"
)

// RenderCallChainRaw returns the call‑chain output in raw text format.
func RenderCallChainRaw(data *types.CallChainOutput) string {
	var buf bytes.Buffer

	buf.WriteString(callchainMetaHeader + "\n")
	buf.WriteString("Target Function: " + data.TargetFunction + "\n")
	buf.WriteString("Callers:\n")
	if len(data.Callers) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, callerName := range data.Callers {
			buf.WriteString(" " + callerName + "\n")
		}
	}
	buf.WriteString("Callees:\n")
	if data.Callees == nil || len(*data.Callees) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, calleeName := range *data.Callees {
			buf.WriteString(" " + calleeName + "\n")
		}
	}
	buf.WriteString("\n")
	buf.WriteString(callchainFunctionsHdr + "\n")
	for _, name := range orderedFunctionNames(data) {
		buf.WriteString("Function: " + name + "\n")
		buf.WriteString(separatorLine + "\n")
		buf.WriteString(data.Functions[name] + "\n")
		buf.WriteString(separatorLine + "\n\n")
	}
	if len(data.Documentation) > 0 {
		buf.WriteString(callchainDocsHeader + "\n")
		for _, documentationEntry := range data.Documentation {
			buf.WriteString(documentationEntry.Kind + " " + documentationEntry.Name + "\n")
			if documentationEntry.Doc != "" {
				buf.WriteString(documentationEntry.Doc + "\n")
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// RenderCallChainJSON marshals the call‑chain output as a JSON array.
func RenderCallChainJSON(data *types.CallChainOutput) (string, error) {
	array := []types.CallChainOutput{*data}
	encoded, err := json.MarshalIndent(array, indentPrefix, indentSpacer)
	return string(encoded), err
}

// RenderCallChainXML marshals the call-chain output as an XML document.
func RenderCallChainXML(data *types.CallChainOutput) (string, error) {
	var callees []string
	if data.Callees != nil {
		callees = *data.Callees
	}
	type xmlFunction struct {
		Name string `xml:"name"`
		Body string `xml:"body"`
	}
	var functionList []xmlFunction
	names := make([]string, 0, len(data.Functions))
	for name := range data.Functions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		functionList = append(functionList, xmlFunction{Name: name, Body: data.Functions[name]})
	}
	type xmlCallChain struct {
		TargetFunction string                     `xml:"targetFunction"`
		Callers        []string                   `xml:"callers>caller"`
		Callees        []string                   `xml:"callees>callee,omitempty"`
		Functions      []xmlFunction              `xml:"functions>function"`
		Documentation  []types.DocumentationEntry `xml:"documentation>entry,omitempty"`
	}
	wrapper := struct {
		XMLName xml.Name       `xml:""`
		Chains  []xmlCallChain `xml:"callchain"`
	}{
		XMLName: xml.Name{Local: xmlCallchainsName},
		Chains: []xmlCallChain{
			{
				TargetFunction: data.TargetFunction,
				Callers:        data.Callers,
				Callees:        callees,
				Functions:      functionList,
				Documentation:  data.Documentation,
			},
		},
	}
	encoded, err := xml.MarshalIndent(wrapper, indentPrefix, indentSpacer)
	if err != nil {
		return "", err
	}
	return xmlHeader + string(encoded), nil
}

// RenderJSON deduplicates and marshals documentation and results to JSON.
func RenderJSON(documentationEntries []types.DocumentationEntry, collected []interface{}) (string, error) {
	dedupedDocs := dedupeDocumentationEntries(documentationEntries)
	dedupedItems := dedupeCollectedItems(collected)

	if len(dedupedDocs) == 0 {
		if len(dedupedItems) == 0 {
			return "[]", nil
		}
		encoded, err := json.MarshalIndent(dedupedItems, indentPrefix, indentSpacer)
		return string(encoded), err
	}

	bundle := struct {
		Documentation []types.DocumentationEntry `json:"documentation"`
		Code          []interface{}              `json:"code"`
	}{
		Documentation: dedupedDocs,
		Code:          dedupedItems,
	}
	encoded, err := json.MarshalIndent(bundle, indentPrefix, indentSpacer)
	return string(encoded), err
}

// RenderXML deduplicates and marshals documentation and results to XML.
func RenderXML(documentationEntries []types.DocumentationEntry, collected []interface{}) (string, error) {
	dedupedDocs := dedupeDocumentationEntries(documentationEntries)
	dedupedItems := dedupeCollectedItems(collected)
	bundle := struct {
		XMLName       xml.Name                   `xml:""`
		Documentation []types.DocumentationEntry `xml:"documentation>entry,omitempty"`
		Code          []interface{}              `xml:"code>item,omitempty"`
	}{
		XMLName:       xml.Name{Local: xmlRootElement},
		Documentation: dedupedDocs,
		Code:          dedupedItems,
	}
	encoded, err := xml.MarshalIndent(bundle, indentPrefix, indentSpacer)
	if err != nil {
		return "", err
	}
	return xmlHeader + string(encoded), nil
}

// RenderRaw deduplicates and prints documentation and results in raw text format.
func RenderRaw(commandName string, documentationEntries []types.DocumentationEntry, collected []interface{}) error {
	dedupedDocs := dedupeDocumentationEntries(documentationEntries)
	dedupedItems := dedupeCollectedItems(collected)

	if len(dedupedDocs) > 0 {
		fmt.Println(documentationHeader)
		for _, entry := range dedupedDocs {
			fmt.Println(entry.Kind, entry.Name)
			if entry.Doc != "" {
				fmt.Println(entry.Doc)
			}
			fmt.Println()
		}
	}

	for _, item := range dedupedItems {
		switch outputItem := item.(type) {
		case *types.FileOutput:
			if commandName == types.CommandContent {
				fmt.Printf("File: %s\n", outputItem.Path)
				if outputItem.Type == types.NodeTypeBinary {
					fmt.Printf("%s%s\n", mimeTypeLabel, outputItem.MimeType)
					if outputItem.Content == "" {
						fmt.Println(binaryContentOmitted)
					} else {
						fmt.Println(outputItem.Content)
					}
				} else {
					fmt.Println(outputItem.Content)
				}
				fmt.Printf("End of file: %s\n", outputItem.Path)
				fmt.Println(separatorLine)
			}
		case *types.TreeOutputNode:
			if commandName == types.CommandTree {
				if outputItem.Type == types.NodeTypeFile {
					fmt.Printf("[File] %s\n", outputItem.Path)
				} else if outputItem.Type == types.NodeTypeBinary {
					fmt.Printf(binaryNodeFormat, outputItem.Path, mimeTypeLabel, outputItem.MimeType)
				} else {
					fmt.Printf("\n--- Directory Tree: %s ---\n", outputItem.Path)
					printTree(outputItem, "")
				}
			}
		case *types.CallChainOutput:
			if commandName == types.CommandCallChain {
				fmt.Print(RenderCallChainRaw(outputItem))
			}
		}
	}

	return nil
}

// dedupeDocumentationEntries returns a slice with duplicate documentation entries removed.
func dedupeDocumentationEntries(entries []types.DocumentationEntry) []types.DocumentationEntry {
	seen := make(map[string]struct{}, len(entries))
	var out []types.DocumentationEntry
	for _, documentationEntry := range entries {
		key := documentationEntry.Kind + ":" + documentationEntry.Name
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			out = append(out, documentationEntry)
		}
	}
	return out
}

// dedupeCollectedItems returns a slice with duplicate collected items removed.
func dedupeCollectedItems(items []interface{}) []interface{} {
	seen := make(map[string]struct{}, len(items))
	var out []interface{}
	for _, item := range items {
		var key string
		switch outputItem := item.(type) {
		case *types.FileOutput:
			key = filePrefix + outputItem.Path
		case *types.TreeOutputNode:
			key = nodePrefix + outputItem.Path
		case *types.CallChainOutput:
			key = callChainPrefix + outputItem.TargetFunction
		default:
			continue
		}
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			out = append(out, item)
		}
	}
	return out
}

// printTree recursively prints a directory tree to stdout.
func printTree(node *types.TreeOutputNode, prefix string) {
	switch node.Type {
	case types.NodeTypeFile:
		fmt.Printf("%s[File] %s\n", prefix, node.Path)
		return
	case types.NodeTypeBinary:
		fmt.Printf(binaryTreeFormat, prefix, node.Path, mimeTypeLabel, node.MimeType)
		return
	}
	fmt.Printf("%s%s\n", prefix, node.Path)
	for _, child := range node.Children {
		printTree(child, prefix+"  ")
	}
}

// orderedFunctionNames returns a deterministic ordering of function names.
func orderedFunctionNames(data *types.CallChainOutput) []string {
	seen := map[string]struct{}{}
	var order []string
	add := func(name string) {
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			order = append(order, name)
		}
	}
	add(data.TargetFunction)
	for _, callerName := range data.Callers {
		add(callerName)
	}
	if data.Callees != nil {
		for _, calleeName := range *data.Callees {
			add(calleeName)
		}
	}
	var others []string
	for name := range data.Functions {
		if _, ok := seen[name]; !ok {
			others = append(others, name)
		}
	}
	sort.Strings(others)
	order = append(order, others...)
	return order
}
