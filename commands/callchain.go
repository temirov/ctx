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

	"github.com/temirov/ctx/docs"
	apptypes "github.com/temirov/ctx/types"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// GetCallChainData returns the call chain information for the requested function.
func GetCallChainData(
	targetFunctionQualifiedName string,
	includeDocumentation bool,
	documentationCollector *docs.Collector,
	repositoryRootDirectory string,
) (*apptypes.CallChainOutput, error) {
	packageLoadConfiguration := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  ".",
		Fset: token.NewFileSet(),
	}
	loadedPackages, loadError := packages.Load(packageLoadConfiguration, "./...")
	if loadError != nil {
		return nil, fmt.Errorf("failed to load packages: %w", loadError)
	}
	if packages.PrintErrors(loadedPackages) > 0 {
		return nil, fmt.Errorf("errors encountered while loading packages")
	}

	ssaProgram, _ := ssautil.Packages(loadedPackages, ssa.BuilderMode(0))
	ssaProgram.Build()

	callGraphRoot := static.CallGraph(ssaProgram)
	callGraphRoot.DeleteSyntheticNodes()

	targetNode := selectFunctionNode(callGraphRoot, targetFunctionQualifiedName)
	if targetNode == nil {
		return nil, fmt.Errorf("target function %q not found in call graph", targetFunctionQualifiedName)
	}

	var directCallerNames []string
	for _, inEdge := range targetNode.In {
		if inEdge.Caller != nil && inEdge.Caller.Func != nil {
			directCallerNames = append(directCallerNames, inEdge.Caller.Func.String())
		}
	}

	var directCalleeNames []string
	for _, outEdge := range targetNode.Out {
		if outEdge.Callee != nil && outEdge.Callee.Func != nil {
			directCalleeNames = append(directCalleeNames, outEdge.Callee.Func.String())
		}
	}

	relevantFunctionNames := map[string]struct{}{
		targetNode.Func.String(): {},
	}
	for _, name := range directCallerNames {
		relevantFunctionNames[name] = struct{}{}
	}
	for _, name := range directCalleeNames {
		relevantFunctionNames[name] = struct{}{}
	}

	functionSources := make(map[string]string)
	extractedFilePaths := make(map[string]struct{})

	for _, pkg := range loadedPackages {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				funcDecl, ok := node.(*ast.FuncDecl)
				if !ok || funcDecl.Name == nil {
					return true
				}
				qualifiedName := composeQualifiedName(pkg, funcDecl)
				if _, needed := relevantFunctionNames[qualifiedName]; !needed {
					return true
				}
				var buf bytes.Buffer
				(&printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 4}).
					Fprint(&buf, packageLoadConfiguration.Fset, funcDecl)
				functionSources[qualifiedName] = buf.String()
				if includeDocumentation {
					if pos := packageLoadConfiguration.Fset.File(funcDecl.Pos()); pos != nil {
						extractedFilePaths[pos.Name()] = struct{}{}
					}
				}
				return true
			})
		}
	}

	output := &apptypes.CallChainOutput{
		TargetFunction: targetFunctionQualifiedName,
		Callers:        directCallerNames,
		Functions:      functionSources,
	}
	if len(directCalleeNames) > 0 {
		output.Callees = &directCalleeNames
	}

	if includeDocumentation && documentationCollector != nil {
		repoPrefix := repositoryRootDirectory + string(os.PathSeparator)
		for filePath := range extractedFilePaths {
			if filePath != repositoryRootDirectory && !strings.HasPrefix(filePath, repoPrefix) {
				continue
			}
			entries, err := documentationCollector.CollectFromFile(filePath)
			if err == nil && len(entries) > 0 {
				output.Documentation = append(output.Documentation, entries...)
			}
		}
	}

	return output, nil
}

func composeQualifiedName(pkg *packages.Package, decl *ast.FuncDecl) string {
	name := decl.Name.Name
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		var buf bytes.Buffer
		printer.Fprint(&buf, pkg.Fset, decl.Recv.List[0].Type)
		return pkg.PkgPath + "." + strings.TrimSpace(buf.String()) + "." + name
	}
	return pkg.PkgPath + "." + name
}

func selectFunctionNode(graph *callgraph.Graph, candidate string) *callgraph.Node {
	short := candidate
	if i := strings.LastIndex(candidate, "."); i >= 0 && i < len(candidate)-1 {
		short = candidate[i+1:]
	}
	var best *callgraph.Node
	for _, node := range graph.Nodes {
		if node.Func == nil {
			continue
		}
		full := node.Func.String()
		if full == candidate {
			return node
		}
		if strings.HasSuffix(full, "."+candidate) || strings.HasSuffix(full, "."+short) {
			if best == nil || len(full) > len(best.Func.String()) {
				best = node
			}
		}
	}
	return best
}
