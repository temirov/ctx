package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	"go/types"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

// CallChainOutput represents the call chain details of a target function.
type CallChainOutput struct {
	TargetFunction string            `json:"targetFunction"`
	Callers        []string          `json:"callers"`
	Callees        *[]string         `json:"callees"`
	Functions      map[string]string `json:"functions"`
}

// GetCallChainData returns the call chain details for the specified target function name.
// It loads all repository packages, builds the SSA program and static call graph,
// and extracts the callers, callees and corresponding function source codes.
func GetCallChainData(targetFunctionName string) (*CallChainOutput, error) {
	packageConfiguration := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  ".",
	}
	loadedPackages, loadError := packages.Load(packageConfiguration, "./...")
	if loadError != nil {
		return nil, fmt.Errorf("failed to load packages: %v", loadError)
	}
	if packages.PrintErrors(loadedPackages) > 0 {
		return nil, fmt.Errorf("errors encountered while loading packages")
	}
	fileSet := loadedPackages[0].Fset
	ssaProgram := ssa.NewProgram(fileSet, ssa.BuilderMode(0))
	for _, currentPackage := range loadedPackages {
		ssaProgram.CreatePackage(currentPackage.Types, currentPackage.Syntax, currentPackage.TypesInfo, true)
	}
	visitedPackages := make(map[string]bool)
	for _, currentPackage := range loadedPackages {
		importRecursively(ssaProgram, currentPackage.Types, visitedPackages)
	}
	ssaProgram.Build()
	callGraphGraph := static.CallGraph(ssaProgram)
	callGraphGraph.DeleteSyntheticNodes()
	targetNode := findFunctionNode(callGraphGraph, targetFunctionName)
	if targetNode == nil {
		return nil, fmt.Errorf("target function not found in call graph: %s", targetFunctionName)
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
	if len(calleeList) == 0 {
		calleesPointer = nil
	} else {
		calleesPointer = &calleeList
	}
	functionsMapping := make(map[string]string)
	for _, currentPackage := range loadedPackages {
		for _, syntaxFile := range currentPackage.Syntax {
			for _, declaration := range syntaxFile.Decls {
				functionDeclaration, isFunction := declaration.(*ast.FuncDecl)
				if !isFunction {
					continue
				}
				qualifiedFunctionName := getQualifiedFunctionName(currentPackage, functionDeclaration)
				if qualifiedFunctionName == targetNode.Func.String() || containsString(callerList, qualifiedFunctionName) || containsString(calleeList, qualifiedFunctionName) {
					sourceText, printingError := nodeToString(currentPackage.Fset, functionDeclaration)
					if printingError == nil {
						functionsMapping[qualifiedFunctionName] = sourceText
					}
				}
			}
		}
	}
	return &CallChainOutput{
		TargetFunction: targetFunctionName,
		Callers:        callerList,
		Callees:        calleesPointer,
		Functions:      functionsMapping,
	}, nil
}

func importRecursively(ssaProgram *ssa.Program, typesPackage *types.Package, visited map[string]bool) {
	if visited[typesPackage.Path()] {
		return
	}
	visited[typesPackage.Path()] = true
	if ssaProgram.Package(typesPackage) == nil {
		ssaProgram.CreatePackage(typesPackage, nil, nil, true)
	}
	for _, importedPackage := range typesPackage.Imports() {
		importRecursively(ssaProgram, importedPackage, visited)
	}
}

func nodeToString(fileSet *token.FileSet, astNode ast.Node) (string, error) {
	var outputBuffer bytes.Buffer
	if err := printer.Fprint(&outputBuffer, fileSet, astNode); err != nil {
		return "", err
	}
	return outputBuffer.String(), nil
}

func getQualifiedFunctionName(pkg *packages.Package, functionDeclaration *ast.FuncDecl) string {
	functionName := functionDeclaration.Name.Name
	if functionDeclaration.Recv != nil && len(functionDeclaration.Recv.List) > 0 {
		var receiverBuffer bytes.Buffer
		if err := printer.Fprint(&receiverBuffer, pkg.Fset, functionDeclaration.Recv.List[0].Type); err != nil {
			receiverBuffer.WriteString("unknown")
		}
		receiverRepresentation := strings.TrimSpace(receiverBuffer.String())
		return pkg.PkgPath + "." + receiverRepresentation + "." + functionName
	}
	return pkg.PkgPath + "." + functionName
}

func findFunctionNode(graph *callgraph.Graph, inputFunctionName string) *callgraph.Node {
	for _, node := range graph.Nodes {
		if node.Func != nil {
			fullFunctionName := node.Func.String()
			if fullFunctionName == inputFunctionName {
				return node
			}
			if strings.HasSuffix(fullFunctionName, inputFunctionName) {
				return node
			}
		}
	}
	return nil
}

func containsString(stringSlice []string, targetString string) bool {
	for _, currentString := range stringSlice {
		if currentString == targetString {
			return true
		}
	}
	return false
}

// RenderCallChainJSON returns the call chain output as an indented JSON string.
func RenderCallChainJSON(callChainOutput *CallChainOutput) (string, error) {
	jsonBytes, err := json.MarshalIndent(callChainOutput, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal call chain output to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// RenderCallChainRaw returns the call chain output as a detailed human-readable string.
func RenderCallChainRaw(callChainOutput *CallChainOutput) string {
	var builder strings.Builder
	builder.WriteString("----- CALLCHAIN METADATA -----\n")
	builder.WriteString(fmt.Sprintf("Target Function: %s\n", callChainOutput.TargetFunction))
	builder.WriteString("Callers:\n")
	if len(callChainOutput.Callers) == 0 {
		builder.WriteString("  (none)\n")
	} else {
		for _, callerName := range callChainOutput.Callers {
			builder.WriteString(fmt.Sprintf("  %s\n", callerName))
		}
	}
	builder.WriteString("Callees:\n")
	if callChainOutput.Callees == nil {
		builder.WriteString("  (none)\n")
	} else {
		for _, calleeName := range *callChainOutput.Callees {
			builder.WriteString(fmt.Sprintf("  %s\n", calleeName))
		}
	}
	builder.WriteString("\n----- FUNCTIONS -----\n")
	if len(callChainOutput.Functions) == 0 {
		builder.WriteString("  (none)\n")
	} else {
		for functionName, sourceCode := range callChainOutput.Functions {
			builder.WriteString(fmt.Sprintf("Function: %s\n", functionName))
			builder.WriteString("--------------------------------------------------\n")
			builder.WriteString(sourceCode)
			builder.WriteString("\n--------------------------------------------------\n\n")
		}
	}
	return builder.String()
}
