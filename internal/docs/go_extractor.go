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

type goExtractor struct {
	currentModulePath string
	packageCache      map[string]*doc.Package
	textCache         map[string]string
	remote            remoteDocumentationProvider
}

const (
	goPackageEntryPrefix = "pkg:"
	goSymbolEntryPrefix  = "sym:"
	goRemoteEntryPrefix  = "remote:"
)

func newGoExtractor(repositoryRoot string, remote remoteDocumentationProvider) (*goExtractor, error) {
	goModBytes, readError := os.ReadFile(filepath.Join(repositoryRoot, "go.mod"))
	if readError != nil {
		return nil, readError
	}
	moduleFile, parseError := modfile.Parse("go.mod", goModBytes, nil)
	if parseError != nil {
		return nil, parseError
	}
	return &goExtractor{
		currentModulePath: moduleFile.Module.Mod.Path,
		packageCache:      map[string]*doc.Package{},
		textCache:         map[string]string{},
		remote:            remote,
	}, nil
}

func (extractor *goExtractor) SupportedExtensions() []string {
	return []string{goFileExtension}
}

func (extractor *goExtractor) RequiresSource() bool {
	return false
}

func (extractor *goExtractor) CollectDocumentation(filePath string, _ []byte) ([]types.DocumentationEntry, error) {
	fileSet := token.NewFileSet()
	fileAST, parseError := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
	if parseError != nil {
		return nil, parseError
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
		if strings.HasPrefix(importPath, extractor.currentModulePath) {
			return
		}
		if _, exists := seenEntries[goPackageEntryPrefix+importPath]; exists {
			return
		}
		if _, cached := extractor.packageCache[importPath]; !cached {
			extractor.packageCache[importPath] = loadGoPackageDocumentation(importPath)
		}
		if packageDoc := extractor.packageCache[importPath]; packageDoc != nil && strings.TrimSpace(packageDoc.Doc) != "" {
			documentationEntries = append(documentationEntries, types.DocumentationEntry{
				Kind: documentationKindPackage,
				Name: importPath,
				Doc:  strings.TrimSpace(packageDoc.Doc),
			})
		}
		seenEntries[goPackageEntryPrefix+importPath] = struct{}{}
	}

	addSymbolEntry := func(importPath, symbolName string) {
		if strings.HasPrefix(importPath, extractor.currentModulePath) {
			return
		}
		symbolIdentifier := importPath + "." + symbolName
		if _, exists := seenEntries[goSymbolEntryPrefix+symbolIdentifier]; exists {
			return
		}
		if text, cached := extractor.textCache[symbolIdentifier]; cached {
			if text != "" {
				documentationEntries = append(documentationEntries, types.DocumentationEntry{
					Kind: documentationKindSymbol,
					Name: symbolIdentifier,
					Doc:  text,
				})
			}
			seenEntries[goSymbolEntryPrefix+symbolIdentifier] = struct{}{}
			return
		}
		packageDoc := extractor.packageCache[importPath]
		if packageDoc == nil {
			packageDoc = loadGoPackageDocumentation(importPath)
			extractor.packageCache[importPath] = packageDoc
		}
		documentationText := ""
		if packageDoc != nil {
			documentationText = strings.TrimSpace(findGoSymbolDocumentation(packageDoc, symbolName))
		}
		extractor.textCache[symbolIdentifier] = documentationText
		if documentationText != "" {
			documentationEntries = append(documentationEntries, types.DocumentationEntry{
				Kind: documentationKindSymbol,
				Name: symbolIdentifier,
				Doc:  documentationText,
			})
		}
		seenEntries[goSymbolEntryPrefix+symbolIdentifier] = struct{}{}
	}

	for _, importPath := range aliasToImport {
		addPackageEntry(importPath)
		if extractor.remote != nil {
			for _, entry := range extractor.remote.DocumentationForImport(importPath) {
				entryKey := goRemoteEntryPrefix + entry.Kind + ":" + entry.Name
				if _, exists := seenEntries[entryKey]; exists {
					continue
				}
				documentationEntries = append(documentationEntries, entry)
				seenEntries[entryKey] = struct{}{}
			}
		}
	}

	ast.Inspect(fileAST, func(node ast.Node) bool {
		selector, isSelector := node.(*ast.SelectorExpr)
		if !isSelector {
			return true
		}
		identifier, isIdentifier := selector.X.(*ast.Ident)
		if !isIdentifier {
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

func loadGoPackageDocumentation(importPath string) *doc.Package {
	packagesConfig := &packages.Config{Mode: packages.NeedSyntax | packages.NeedFiles}
	if loadedPackages, loadError := packages.Load(packagesConfig, importPath); loadError == nil && len(loadedPackages) > 0 && len(loadedPackages[0].Syntax) > 0 {
		packageDocumentation, _ := doc.NewFromFiles(loadedPackages[0].Fset, loadedPackages[0].Syntax, importPath)
		return packageDocumentation
	}
	buildPackage, importError := build.Default.Import(importPath, "", build.FindOnly)
	if importError != nil {
		return nil
	}
	fileSet := token.NewFileSet()
	directoryASTMap, parseError := parser.ParseDir(fileSet, buildPackage.Dir, nil, parser.ParseComments)
	if parseError != nil {
		return nil
	}
	var files []*ast.File
	for _, astPackage := range directoryASTMap {
		for _, fileAST := range astPackage.Files {
			files = append(files, fileAST)
		}
	}
	packageDocumentation, _ := doc.NewFromFiles(fileSet, files, importPath)
	return packageDocumentation
}

func findGoSymbolDocumentation(packageDoc *doc.Package, symbol string) string {
	for _, functionDocumentation := range packageDoc.Funcs {
		if functionDocumentation.Name == symbol {
			return functionDocumentation.Doc
		}
	}
	for _, typeDocumentation := range packageDoc.Types {
		for _, methodDocumentation := range typeDocumentation.Methods {
			if methodDocumentation.Name == symbol {
				return methodDocumentation.Doc
			}
		}
		for _, functionDocumentation := range typeDocumentation.Funcs {
			if functionDocumentation.Name == symbol {
				return functionDocumentation.Doc
			}
		}
	}
	return ""
}
