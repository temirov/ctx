package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/temirov/ctx/types"
)

const (
	indentPrefix     = ""
	indentSpacer     = "  "
	separatorLine    = "----------------------------------------"
	documentationHdr = "--- Documentation ---"
	callchainMetaHdr = "----- CALLCHAIN METADATA -----"
	callchainFuncHdr = "----- FUNCTIONS -----"
	callchainDocsHdr = "--- DOCS ---"
)

// RenderCallChainRaw returns the call‑chain output in raw text format.
func RenderCallChainRaw(data *types.CallChainOutput) string {
	var buf bytes.Buffer

	buf.WriteString(callchainMetaHdr + "\n")
	buf.WriteString("Target Function: " + data.TargetFunction + "\n")
	buf.WriteString("Callers:\n")
	if len(data.Callers) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, c := range data.Callers {
			buf.WriteString(" " + c + "\n")
		}
	}
	buf.WriteString("Callees:\n")
	if data.Callees == nil || len(*data.Callees) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, c := range *data.Callees {
			buf.WriteString(" " + c + "\n")
		}
	}
	buf.WriteString("\n")
	buf.WriteString(callchainFuncHdr + "\n")
	for _, name := range orderedFunctionNames(data) {
		buf.WriteString("Function: " + name + "\n")
		buf.WriteString(separatorLine + "\n")
		buf.WriteString(data.Functions[name] + "\n")
		buf.WriteString(separatorLine + "\n\n")
	}
	if len(data.Documentation) > 0 {
		buf.WriteString(callchainDocsHdr + "\n")
		for _, e := range data.Documentation {
			buf.WriteString(e.Kind + " " + e.Name + "\n")
			if e.Doc != "" {
				buf.WriteString(e.Doc + "\n")
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

// RenderRaw deduplicates and prints documentation and results in raw text format.
func RenderRaw(commandName string, documentationEntries []types.DocumentationEntry, collected []interface{}) error {
	dedupedDocs := dedupeDocumentationEntries(documentationEntries)
	dedupedItems := dedupeCollectedItems(collected)

	if len(dedupedDocs) > 0 {
		fmt.Println(documentationHdr)
		for _, entry := range dedupedDocs {
			fmt.Println(entry.Kind, entry.Name)
			if entry.Doc != "" {
				fmt.Println(entry.Doc)
			}
			fmt.Println()
		}
	}

	for _, item := range dedupedItems {
		switch v := item.(type) {
		case *types.FileOutput:
			if commandName == types.CommandContent {
				fmt.Printf("File: %s\n", v.Path)
				fmt.Println(v.Content)
				fmt.Printf("End of file: %s\n", v.Path)
				fmt.Println(separatorLine)
			}
		case *types.TreeOutputNode:
			if commandName == types.CommandTree {
				switch v.Type {
				case types.NodeTypeFile:
					fmt.Printf("[File] %s\n", v.Path)
				case types.NodeTypeBinary:
					fmt.Printf("[Binary] %s\n", v.Path)
				default:
					fmt.Printf("\n--- Directory Tree: %s ---\n", v.Path)
					printTree(v, "")
				}
			}
		case *types.CallChainOutput:
			if commandName == types.CommandCallChain {
				fmt.Print(RenderCallChainRaw(v))
			}
		}
	}

	return nil
}

func dedupeDocumentationEntries(entries []types.DocumentationEntry) []types.DocumentationEntry {
	seen := make(map[string]struct{}, len(entries))
	var out []types.DocumentationEntry
	for _, e := range entries {
		key := e.Kind + ":" + e.Name
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			out = append(out, e)
		}
	}
	return out
}

func dedupeCollectedItems(items []interface{}) []interface{} {
	seen := make(map[string]struct{}, len(items))
	var out []interface{}
	for _, item := range items {
		var key string
		switch v := item.(type) {
		case *types.FileOutput:
			key = "file:" + v.Path
		case *types.TreeOutputNode:
			key = "node:" + v.Path
		case *types.CallChainOutput:
			key = "callchain:" + v.TargetFunction
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

func printTree(node *types.TreeOutputNode, prefix string) {
	switch node.Type {
	case types.NodeTypeFile:
		fmt.Printf("%s[File] %s\n", prefix, node.Path)
		return
	case types.NodeTypeBinary:
		fmt.Printf("%s[Binary] %s\n", prefix, node.Path)
		return
	}
	fmt.Printf("%s%s\n", prefix, node.Path)
	for _, child := range node.Children {
		printTree(child, prefix+"  ")
	}
}

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
	for _, c := range data.Callers {
		add(c)
	}
	if data.Callees != nil {
		for _, c := range *data.Callees {
			add(c)
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
