// Package docs provides documentation extraction for external packages and
// symbols referenced by ctx when the --doc flag is used.
package docs

import (
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"

	"github.com/temirov/ctx/internal/types"
)

type Collector struct {
	currentModulePath string
	packageCache      map[string]*doc.Package
	textCache         map[string]string
}

// NewCollector creates a Collector using the repository root that contains go.mod.
func NewCollector(repositoryRoot string) (*Collector, error) {
	goModBytes, readErr := os.ReadFile(filepath.Join(repositoryRoot, "go.mod"))
	if readErr != nil {
		return nil, readErr
	}
	mod, parseErr := modfile.Parse("go.mod", goModBytes, nil)
	if parseErr != nil {
		return nil, parseErr
	}
	return &Collector{
		currentModulePath: mod.Module.Mod.Path,
		packageCache:      map[string]*doc.Package{},
		textCache:         map[string]string{},
	}, nil
}

// CollectFromFile returns documentation entries for every imported package and
// every selector expression that refers to an external package in the source file.
func (collector *Collector) CollectFromFile(filePath string) ([]types.DocumentationEntry, error) {
	fileSet := token.NewFileSet()
	fileAST, parseErr := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
	if parseErr != nil {
		return nil, parseErr
	}

	aliasToImport := map[string]string{}
	for _, importSpec := range fileAST.Imports {
		importPath := strings.Trim(importSpec.Path.Value, `"`)
		if importSpec.Name != nil && importSpec.Name.Name != "_" && importSpec.Name.Name != "." {
			aliasToImport[importSpec.Name.Name] = importPath
		} else {
			aliasToImport[filepath.Base(importPath)] = importPath
		}
	}

	var documentationEntries []types.DocumentationEntry
	seenEntries := map[string]struct{}{}

	addPackageEntry := func(importPath string) {
		if strings.HasPrefix(importPath, collector.currentModulePath) {
			return
		}
		if _, exists := seenEntries["pkg:"+importPath]; exists {
			return
		}
		if _, cached := collector.packageCache[importPath]; !cached {
			collector.packageCache[importPath] = loadPackageDoc(importPath)
		}
		if packageDoc := collector.packageCache[importPath]; packageDoc != nil && strings.TrimSpace(packageDoc.Doc) != "" {
			documentationEntries = append(documentationEntries, types.DocumentationEntry{
				Kind: "package",
				Name: importPath,
				Doc:  strings.TrimSpace(packageDoc.Doc),
			})
		}
		seenEntries["pkg:"+importPath] = struct{}{}
	}

	addSymbolEntry := func(importPath, symbol string) {
		if strings.HasPrefix(importPath, collector.currentModulePath) {
			return
		}
		key := importPath + "." + symbol
		if _, exists := seenEntries["sym:"+key]; exists {
			return
		}
		if text, cached := collector.textCache[key]; cached {
			if text != "" {
				documentationEntries = append(documentationEntries, types.DocumentationEntry{
					Kind: "function",
					Name: key,
					Doc:  text,
				})
			}
			seenEntries["sym:"+key] = struct{}{}
			return
		}
		packageDoc := collector.packageCache[importPath]
		if packageDoc == nil {
			packageDoc = loadPackageDoc(importPath)
			collector.packageCache[importPath] = packageDoc
		}
		text := ""
		if packageDoc != nil {
			text = strings.TrimSpace(findSymbolDoc(packageDoc, symbol))
		}
		collector.textCache[key] = text
		if text != "" {
			documentationEntries = append(documentationEntries, types.DocumentationEntry{
				Kind: "function",
				Name: key,
				Doc:  text,
			})
		}
		seenEntries["sym:"+key] = struct{}{}
	}

	for _, importPath := range aliasToImport {
		addPackageEntry(importPath)
	}

	ast.Inspect(fileAST, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		importPath, exists := aliasToImport[identifier.Name]
		if !exists {
			return true
		}
		addSymbolEntry(importPath, selector.Sel.Name)
		return true
	})

	return documentationEntries, nil
}

// loadPackageDoc loads documentation for the specified import path.
func loadPackageDoc(importPath string) *doc.Package {
	packagesConfig := &packages.Config{Mode: packages.NeedSyntax | packages.NeedFiles}
	if loadedPackages, loadErr := packages.Load(packagesConfig, importPath); loadErr == nil && len(loadedPackages) > 0 && len(loadedPackages[0].Syntax) > 0 {
		packageDoc, _ := doc.NewFromFiles(loadedPackages[0].Fset, loadedPackages[0].Syntax, importPath)
		return packageDoc
	}
	buildPackage, importErr := build.Default.Import(importPath, "", build.FindOnly)
	if importErr != nil {
		return nil
	}
	fileSet := token.NewFileSet()
	directoryASTMap, parseErr := parser.ParseDir(fileSet, buildPackage.Dir, nil, parser.ParseComments)
	if parseErr != nil {
		return nil
	}
	var files []*ast.File
	for _, astPackage := range directoryASTMap {
		for _, fileAST := range astPackage.Files {
			files = append(files, fileAST)
		}
	}
	packageDoc, _ := doc.NewFromFiles(fileSet, files, importPath)
	return packageDoc
}

// findSymbolDoc retrieves documentation text for the named symbol.
func findSymbolDoc(packageDoc *doc.Package, symbol string) string {
	for _, functionDoc := range packageDoc.Funcs {
		if functionDoc.Name == symbol {
			return functionDoc.Doc
		}
	}
	for _, typeDoc := range packageDoc.Types {
		for _, methodDoc := range typeDoc.Methods {
			if methodDoc.Name == symbol {
				return methodDoc.Doc
			}
		}
		for _, functionDoc := range typeDoc.Funcs {
			if functionDoc.Name == symbol {
				return functionDoc.Doc
			}
		}
	}
	return ""
}
