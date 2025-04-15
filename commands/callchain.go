// Package commands contains the core logic for data collection for each command.
package commands

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil" // Import the ssautil package

	apptypes "github.com/temirov/ctx/types"
)

// GetCallChainData returns the call chain details for the specified target function name.
// It loads all repository packages, builds the SSA program and static call graph,
// and extracts the callers, callees and corresponding function source codes.
func GetCallChainData(targetFunctionName string) (*apptypes.CallChainOutput, error) {
	packageConfiguration := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  ".",
		Fset: token.NewFileSet(),
	}
	loadedPackages, loadError := packages.Load(packageConfiguration, "./...")
	if loadError != nil {
		return nil, fmt.Errorf("failed to load packages: %w", loadError)
	}
	if packages.PrintErrors(loadedPackages) > 0 {
		return nil, fmt.Errorf("errors encountered while loading packages")
	}

	// Use ssautil.Packages to create the SSA program and packages.
	// BuilderMode 0 provides the default SSA construction behavior.
	program, _ := ssautil.Packages(loadedPackages, ssa.BuilderMode(0))
	if program == nil {
		// This indicates an error during SSA construction not caught earlier.
		return nil, fmt.Errorf("failed to build SSA program (ssautil.Packages returned nil program)")
	}

	// Build the SSA code for all function bodies in the program.
	program.Build()

	// Build the static call graph from the fully constructed SSA program.
	callGraphGraph := static.CallGraph(program)
	callGraphGraph.DeleteSyntheticNodes()

	targetNode := findFunctionNode(callGraphGraph, targetFunctionName)
	if targetNode == nil {
		return nil, fmt.Errorf("target function '%s' not found in call graph", targetFunctionName)
	}

	var callerList []string
	for _, incomingEdge := range targetNode.In {
		if incomingEdge.Caller != nil && incomingEdge.Caller.Func != nil {
			callerList = append(callerList, incomingEdge.Caller.Func.String())
		}
	}

	var calleeList []string
	for _, outgoingEdge := range targetNode.Out {
		if outgoingEdge.Callee != nil && outgoingEdge.Callee.Func != nil {
			calleeList = append(calleeList, outgoingEdge.Callee.Func.String())
		}
	}

	var calleesPointer *[]string
	if len(calleeList) > 0 {
		calleesPointer = &calleeList
	}

	functionsMapping := make(map[string]string)
	relevantFunctions := make(map[string]bool)
	resolvedTargetName := targetNode.Func.String()
	relevantFunctions[resolvedTargetName] = true
	for _, caller := range callerList {
		relevantFunctions[caller] = true
	}
	for _, callee := range calleeList {
		relevantFunctions[callee] = true
	}

	for _, currentPackage := range loadedPackages {
		for _, syntaxFile := range currentPackage.Syntax {
			ast.Inspect(syntaxFile, func(node ast.Node) bool {
				functionDeclaration, isFunction := node.(*ast.FuncDecl)
				if !isFunction || functionDeclaration.Name == nil {
					return true
				}

				qualifiedFunctionName := getQualifiedFunctionName(currentPackage, functionDeclaration)

				if relevantFunctions[qualifiedFunctionName] {
					sourceText, printingError := nodeToString(packageConfiguration.Fset, functionDeclaration)
					if printingError == nil {
						functionsMapping[qualifiedFunctionName] = sourceText
					} else {
						fmt.Fprintf(os.Stderr, "Warning: could not get source for %s: %v\n", qualifiedFunctionName, printingError)
					}
				}
				return true
			})
		}
	}

	return &apptypes.CallChainOutput{
		TargetFunction: resolvedTargetName,
		Callers:        callerList,
		Callees:        calleesPointer,
		Functions:      functionsMapping,
	}, nil
}

func nodeToString(fileSet *token.FileSet, astNode ast.Node) (string, error) {
	var outputBuffer bytes.Buffer
	config := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 4}
	err := config.Fprint(&outputBuffer, fileSet, astNode)
	if err != nil {
		return "", err
	}
	return outputBuffer.String(), nil
}

func getQualifiedFunctionName(pkg *packages.Package, functionDeclaration *ast.FuncDecl) string {
	functionName := functionDeclaration.Name.Name
	if functionDeclaration.Recv != nil && len(functionDeclaration.Recv.List) > 0 {
		receiverType := functionDeclaration.Recv.List[0].Type
		var receiverBuffer bytes.Buffer
		if err := printer.Fprint(&receiverBuffer, pkg.Fset, receiverType); err != nil {
			return pkg.PkgPath + ".(*unknown)." + functionName
		}
		receiverRepresentation := strings.TrimSpace(receiverBuffer.String())
		return pkg.PkgPath + "." + receiverRepresentation + "." + functionName
	}
	return pkg.PkgPath + "." + functionName
}

func findFunctionNode(graph *callgraph.Graph, inputFunctionName string) *callgraph.Node {
	var bestMatch *callgraph.Node
	for _, node := range graph.Nodes {
		if node.Func != nil {
			fullFunctionName := node.Func.String()
			if fullFunctionName == inputFunctionName {
				return node
			}
			if strings.HasSuffix(fullFunctionName, "."+inputFunctionName) {
				if bestMatch == nil || len(fullFunctionName) > len(bestMatch.Func.String()) {
					bestMatch = node
				}
			}
		}
	}
	return bestMatch
}
