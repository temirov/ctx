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

	"github.com/temirov/ctx/types"
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
func (c *Collector) CollectFromFile(filePath string) ([]types.DocumentationEntry, error) {
	fileSet := token.NewFileSet()
	fileAST, parseErr := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
	if parseErr != nil {
		return nil, parseErr
	}

	aliasToImport := map[string]string{}
	for _, imp := range fileAST.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil && imp.Name.Name != "_" && imp.Name.Name != "." {
			aliasToImport[imp.Name.Name] = importPath
		} else {
			aliasToImport[filepath.Base(importPath)] = importPath
		}
	}

	var entries []types.DocumentationEntry
	seen := map[string]struct{}{}

	addPackageEntry := func(importPath string) {
		if strings.HasPrefix(importPath, c.currentModulePath) {
			return
		}
		if _, ok := seen["pkg:"+importPath]; ok {
			return
		}
		if _, cached := c.packageCache[importPath]; !cached {
			c.packageCache[importPath] = loadPackageDoc(importPath)
		}
		if pkg := c.packageCache[importPath]; pkg != nil && strings.TrimSpace(pkg.Doc) != "" {
			entries = append(entries, types.DocumentationEntry{
				Kind: "package",
				Name: importPath,
				Doc:  strings.TrimSpace(pkg.Doc),
			})
		}
		seen["pkg:"+importPath] = struct{}{}
	}

	addSymbolEntry := func(importPath, symbol string) {
		if strings.HasPrefix(importPath, c.currentModulePath) {
			return
		}
		key := importPath + "." + symbol
		if _, ok := seen["sym:"+key]; ok {
			return
		}
		if text, cached := c.textCache[key]; cached {
			if text != "" {
				entries = append(entries, types.DocumentationEntry{
					Kind: "function",
					Name: key,
					Doc:  text,
				})
			}
			seen["sym:"+key] = struct{}{}
			return
		}
		pkg := c.packageCache[importPath]
		if pkg == nil {
			pkg = loadPackageDoc(importPath)
			c.packageCache[importPath] = pkg
		}
		text := ""
		if pkg != nil {
			text = strings.TrimSpace(findSymbolDoc(pkg, symbol))
		}
		c.textCache[key] = text
		if text != "" {
			entries = append(entries, types.DocumentationEntry{
				Kind: "function",
				Name: key,
				Doc:  text,
			})
		}
		seen["sym:"+key] = struct{}{}
	}

	for _, importPath := range aliasToImport {
		addPackageEntry(importPath)
	}

	ast.Inspect(fileAST, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		importPath, exists := aliasToImport[ident.Name]
		if !exists {
			return true
		}
		addSymbolEntry(importPath, sel.Sel.Name)
		return true
	})

	return entries, nil
}

func loadPackageDoc(importPath string) *doc.Package {
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedFiles}
	if pkgs, err := packages.Load(cfg, importPath); err == nil && len(pkgs) > 0 && len(pkgs[0].Syntax) > 0 {
		pkg, _ := doc.NewFromFiles(pkgs[0].Fset, pkgs[0].Syntax, importPath)
		return pkg
	}
	buildPkg, err := build.Default.Import(importPath, "", build.FindOnly)
	if err != nil {
		return nil
	}
	fs := token.NewFileSet()
	dirMap, err := parser.ParseDir(fs, buildPkg.Dir, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	var files []*ast.File
	for _, p := range dirMap {
		for _, f := range p.Files {
			files = append(files, f)
		}
	}
	pkg, _ := doc.NewFromFiles(fs, files, importPath)
	return pkg
}

func findSymbolDoc(pkg *doc.Package, symbol string) string {
	for _, f := range pkg.Funcs {
		if f.Name == symbol {
			return f.Doc
		}
	}
	for _, t := range pkg.Types {
		for _, m := range t.Methods {
			if m.Name == symbol {
				return m.Doc
			}
		}
		for _, f := range t.Funcs {
			if f.Name == symbol {
				return f.Doc
			}
		}
	}
	return ""
}
