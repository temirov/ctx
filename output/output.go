// Package output handles rendering the collected data into different formats (JSON, Raw).
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/temirov/content/types"
)

const (
	jsonIndentPrefix = ""
	jsonIndentSpacer = "  "

	rawTreePrefixInitial = ""
	rawTreePrefixBranch  = "│   "
	rawTreePrefixLeaf    = "    "
	rawTreeConnectorMid  = "├── "
	rawTreeConnectorLast = "└── "

	rawContentFileHeaderPrefix = "File: "
	rawContentEndOfFilePrefix  = "End of file: "
	rawContentSeparator        = "----------------------------------------"

	rawTreeFileMarkerPrefix = "[File] "
	rawTreeDirHeaderPrefix  = "--- Directory Tree: "
	rawTreeDirHeaderSuffix  = " ---"

	rawCallChainMetaHeader    = "----- CALLCHAIN METADATA -----"
	rawCallChainFuncHeader    = "----- FUNCTIONS -----"
	rawCallChainFuncSeparator = "--------------------------------------------------"
	rawCallChainFuncPrefix    = "Function: "
	rawCallChainTargetPrefix  = "Target Function: "
	rawCallChainCallersLabel  = "Callers:"
	rawCallChainCalleesLabel  = "Callees:"
	rawCallChainIndent        = "  "
	rawCallChainNone          = "(none)"
)

// RenderJSON marshals the collected results into an indented JSON string.
func RenderJSON(results []interface{}) (string, error) {
	if len(results) == 0 {
		return "[]", nil
	}
	jsonData, err := json.MarshalIndent(results, jsonIndentPrefix, jsonIndentSpacer)
	if err != nil {
		return "", fmt.Errorf("failed to marshal results to JSON: %w", err)
	}
	return string(jsonData), nil
}

// RenderRaw iterates through results and prints them to stdout in the original text format.
func RenderRaw(commandName string, results []interface{}) error {
	isFirstOutput := true
	for _, result := range results {
		switch item := result.(type) {
		case *types.FileOutput:
			if commandName == types.CommandContent {
				fmt.Printf("%s%s\n", rawContentFileHeaderPrefix, item.Path)
				fmt.Println(item.Content)
				fmt.Printf("%s%s\n", rawContentEndOfFilePrefix, item.Path)
				fmt.Println(rawContentSeparator)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Unexpected FileOutput during raw '%s' render for path %s\n", commandName, item.Path)
			}
		case *types.TreeOutputNode:
			if commandName == types.CommandTree {
				if item.Type == types.NodeTypeFile {
					fmt.Printf("%s%s\n", rawTreeFileMarkerPrefix, item.Path)
				} else if item.Type == types.NodeTypeDirectory {
					if !isFirstOutput {
						fmt.Println()
					}
					fmt.Printf("%s%s%s\n", rawTreeDirHeaderPrefix, item.Path, rawTreeDirHeaderSuffix)
					printRawTreeNode(item, rawTreePrefixInitial)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Unexpected TreeOutputNode during raw '%s' render for path %s\n", commandName, item.Path)
			}
		default:
			fmt.Fprintf(os.Stderr, "Warning: Skipping unexpected result type during raw render: %T\n", item)
		}
		isFirstOutput = false
	}
	return nil
}

func printRawTreeNode(node *types.TreeOutputNode, prefix string) {
	if node == nil || node.Type != types.NodeTypeDirectory || len(node.Children) == 0 {
		return
	}

	numberOfChildren := len(node.Children)
	for index, child := range node.Children {
		isLastChild := index == numberOfChildren-1

		connector := rawTreeConnectorMid
		newPrefix := prefix + rawTreePrefixBranch
		if isLastChild {
			connector = rawTreeConnectorLast
			newPrefix = prefix + rawTreePrefixLeaf
		}

		fmt.Printf("%s%s%s\n", prefix, connector, child.Name)

		if child.Type == types.NodeTypeDirectory {
			printRawTreeNode(child, newPrefix)
		}
	}
}

// RenderCallChainJSON marshals the call chain output into an indented JSON string.
func RenderCallChainJSON(callChainOutput *types.CallChainOutput) (string, error) {
	jsonBytes, err := json.MarshalIndent(callChainOutput, jsonIndentPrefix, jsonIndentSpacer)
	if err != nil {
		return "", fmt.Errorf("failed to marshal call chain output to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// RenderCallChainRaw formats the call chain output into a detailed human-readable string.
func RenderCallChainRaw(callChainOutput *types.CallChainOutput) string {
	var builder strings.Builder

	builder.WriteString(rawCallChainMetaHeader)
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("%s%s\n", rawCallChainTargetPrefix, callChainOutput.TargetFunction))

	builder.WriteString(rawCallChainCallersLabel)
	builder.WriteString("\n")
	if len(callChainOutput.Callers) == 0 {
		builder.WriteString(rawCallChainIndent)
		builder.WriteString(rawCallChainNone)
		builder.WriteString("\n")
	} else {
		for _, callerName := range callChainOutput.Callers {
			builder.WriteString(rawCallChainIndent)
			builder.WriteString(callerName)
			builder.WriteString("\n")
		}
	}

	builder.WriteString(rawCallChainCalleesLabel)
	builder.WriteString("\n")
	if callChainOutput.Callees == nil || len(*callChainOutput.Callees) == 0 {
		builder.WriteString(rawCallChainIndent)
		builder.WriteString(rawCallChainNone)
		builder.WriteString("\n")
	} else {
		for _, calleeName := range *callChainOutput.Callees {
			builder.WriteString(rawCallChainIndent)
			builder.WriteString(calleeName)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n")
	builder.WriteString(rawCallChainFuncHeader)
	builder.WriteString("\n")
	if len(callChainOutput.Functions) == 0 {
		builder.WriteString(rawCallChainIndent)
		builder.WriteString(rawCallChainNone)
		builder.WriteString("\n")
	} else {
		for functionName, sourceCode := range callChainOutput.Functions {
			builder.WriteString(fmt.Sprintf("%s%s\n", rawCallChainFuncPrefix, functionName))
			builder.WriteString(rawCallChainFuncSeparator)
			builder.WriteString("\n")
			builder.WriteString(sourceCode)
			if !strings.HasSuffix(sourceCode, "\n") {
				builder.WriteString("\n")
			}
			builder.WriteString(rawCallChainFuncSeparator)
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}
