package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type OrphanedCodeFinder struct {
	projectPath  string
	fileSet      *token.FileSet
	packages     []*packages.Package
	symbols      map[string]*Symbol
	references   map[string][]Reference
	reachable    map[string]bool // Tracks symbols reachable from main packages
	mainPackages []*packages.Package
}

type Symbol struct {
	Name     string
	Kind     string // "function", "variable", "type", "constant"
	File     string
	Position token.Position
	Exported bool
	Package  string
}

type Reference struct {
	File     string
	Position token.Position
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <project-path>\n", os.Args[0])
		os.Exit(1)
	}

	projectPath := os.Args[1]
	finder := NewOrphanedCodeFinder(projectPath)

	fmt.Printf("Analyzing project at: %s\n", projectPath)

	if err := finder.LoadProject(); err != nil {
		log.Fatalf("Failed to load project: %v", err)
	}

	fmt.Printf("Loaded %d packages\n", len(finder.packages))

	if err := finder.FindSymbols(); err != nil {
		log.Fatalf("Failed to find symbols: %v", err)
	}

	fmt.Printf("Found %d symbols\n", len(finder.symbols))

	if err := finder.FindReferences(); err != nil {
		log.Fatalf("Failed to find references: %v", err)
	}

	if err := finder.IdentifyMainPackages(); err != nil {
		log.Fatalf("Failed to identify main packages: %v", err)
	}

	if err := finder.TraceReachability(); err != nil {
		log.Fatalf("Failed to trace reachability: %v", err)
	}

	orphans := finder.FindOrphans()
	finder.PrintOrphans(orphans)
}

func NewOrphanedCodeFinder(projectPath string) *OrphanedCodeFinder {
	return &OrphanedCodeFinder{
		projectPath: projectPath,
		fileSet:     token.NewFileSet(),
		symbols:     make(map[string]*Symbol),
		references:  make(map[string][]Reference),
		reachable:   make(map[string]bool),
	}
}

func (f *OrphanedCodeFinder) LoadProject() error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   f.projectPath,
		Fset:  f.fileSet,
		Tests: false, // Skip test files for now
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	// Filter out packages with errors
	var validPkgs []*packages.Package
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			fmt.Printf("Skipping package %s due to errors:\n", pkg.PkgPath)
			for _, err := range pkg.Errors {
				fmt.Printf("  %v\n", err)
			}
			continue
		}
		validPkgs = append(validPkgs, pkg)
	}

	f.packages = validPkgs
	return nil
}

func (f *OrphanedCodeFinder) FindSymbols() error {
	for _, pkg := range f.packages {
		for i, file := range pkg.Syntax {
			if i < len(pkg.CompiledGoFiles) {
				f.findSymbolsInFile(pkg, file, pkg.CompiledGoFiles[i])
			}
		}
	}
	return nil
}

func (f *OrphanedCodeFinder) findSymbolsInFile(pkg *packages.Package, file *ast.File, filename string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name != nil && node.Name.Name != "_" {
				pos := f.fileSet.Position(node.Pos())
				symbol := &Symbol{
					Name:     node.Name.Name,
					Kind:     "function",
					File:     filename,
					Position: pos,
					Exported: ast.IsExported(node.Name.Name),
					Package:  pkg.PkgPath,
				}
				key := f.getSymbolKey(pkg.PkgPath, node.Name.Name, "function")
				f.symbols[key] = symbol
			}

		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name != nil && s.Name.Name != "_" {
						pos := f.fileSet.Position(s.Pos())
						symbol := &Symbol{
							Name:     s.Name.Name,
							Kind:     "type",
							File:     filename,
							Position: pos,
							Exported: ast.IsExported(s.Name.Name),
							Package:  pkg.PkgPath,
						}
						key := f.getSymbolKey(pkg.PkgPath, s.Name.Name, "type")
						f.symbols[key] = symbol
					}

				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name != nil && name.Name != "_" {
							pos := f.fileSet.Position(name.Pos())
							kind := "variable"
							if node.Tok == token.CONST {
								kind = "constant"
							}
							symbol := &Symbol{
								Name:     name.Name,
								Kind:     kind,
								File:     filename,
								Position: pos,
								Exported: ast.IsExported(name.Name),
								Package:  pkg.PkgPath,
							}
							key := f.getSymbolKey(pkg.PkgPath, name.Name, kind)
							f.symbols[key] = symbol
						}
					}
				}
			}
		}
		return true
	})
}

func (f *OrphanedCodeFinder) FindReferences() error {
	for _, pkg := range f.packages {
		for _, file := range pkg.Syntax {
			f.findReferencesInFile(pkg, file)
		}
	}
	return nil
}

func (f *OrphanedCodeFinder) findReferencesInFile(pkg *packages.Package, file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			// Check if this identifier is being used (not declared)
			if obj := pkg.TypesInfo.Uses[node]; obj != nil {
				pos := f.fileSet.Position(node.Pos())

				// Determine the kind of symbol being referenced based on types.Object
				kind := f.getObjectKind(obj)

				// Get package path, handling nil package
				pkgPath := ""
				if obj.Pkg() != nil {
					pkgPath = obj.Pkg().Path()
				}

				key := f.getSymbolKey(pkgPath, obj.Name(), kind)

				f.references[key] = append(f.references[key], Reference{
					File:     pos.Filename,
					Position: pos,
				})
			}

		case *ast.SelectorExpr:
			// Handle qualified identifiers (pkg.Symbol)
			if obj := pkg.TypesInfo.Uses[node.Sel]; obj != nil {
				pos := f.fileSet.Position(node.Sel.Pos())

				kind := f.getObjectKind(obj)

				// Get package path, handling nil package
				pkgPath := ""
				if obj.Pkg() != nil {
					pkgPath = obj.Pkg().Path()
				}

				key := f.getSymbolKey(pkgPath, obj.Name(), kind)

				f.references[key] = append(f.references[key], Reference{
					File:     pos.Filename,
					Position: pos,
				})
			}
		}
		return true
	})
}

func (f *OrphanedCodeFinder) IdentifyMainPackages() error {
	for _, pkg := range f.packages {
		if pkg.Name == "main" {
			f.mainPackages = append(f.mainPackages, pkg)
		}
	}

	if len(f.mainPackages) == 0 {
		fmt.Println("⚠️  No main packages found - analyzing all packages for internal usage")
		// If no main packages, treat all packages as potentially reachable
		for _, pkg := range f.packages {
			f.mainPackages = append(f.mainPackages, pkg)
		}
	} else {
		fmt.Printf("📦 Found %d main package(s)\n", len(f.mainPackages))
	}

	return nil
}

func (f *OrphanedCodeFinder) TraceReachability() error {
	fmt.Println("🔍 Tracing reachability from main packages...")

	// Start from all entry points in main packages
	queue := []string{}

	// Add main functions and init functions as entry points
	for _, pkg := range f.mainPackages {
		mainKey := f.getSymbolKey(pkg.PkgPath, "main", "function")
		if _, exists := f.symbols[mainKey]; exists {
			queue = append(queue, mainKey)
			f.reachable[mainKey] = true
		}

		// Also add init functions as entry points
		initKey := f.getSymbolKey(pkg.PkgPath, "init", "function")
		if _, exists := f.symbols[initKey]; exists {
			queue = append(queue, initKey)
			f.reachable[initKey] = true
		}

		// Add all exported symbols from main packages as potentially reachable
		// (they might be called by tests or external tools)
		for symbolKey, symbol := range f.symbols {
			if symbol.Package == pkg.PkgPath && symbol.Exported {
				if !f.reachable[symbolKey] {
					queue = append(queue, symbolKey)
					f.reachable[symbolKey] = true
				}
			}
		}
	}

	// BFS to find all reachable symbols
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		// Find all symbols referenced by the current symbol
		referencedSymbols := f.findReferencedSymbols(current)

		for _, refSymbol := range referencedSymbols {
			if !f.reachable[refSymbol] {
				f.reachable[refSymbol] = true
				queue = append(queue, refSymbol)
			}
		}
	}

	reachableCount := len(f.reachable)
	totalCount := len(f.symbols)
	fmt.Printf("📊 Reachability analysis: %d/%d symbols reachable from main packages\n",
		reachableCount, totalCount)

	return nil
}

func (f *OrphanedCodeFinder) findReferencedSymbols(symbolKey string) []string {
	var referenced []string

	// Get the symbol to find its file(s)
	symbol, exists := f.symbols[symbolKey]
	if !exists {
		return referenced
	}

	// Find all packages that might contain this symbol's definition
	var relevantPackages []*packages.Package
	for _, pkg := range f.packages {
		if pkg.PkgPath == symbol.Package {
			relevantPackages = append(relevantPackages, pkg)
		}
	}

	// Look through the symbol's definition files for references to other symbols
	for _, pkg := range relevantPackages {
		for _, file := range pkg.Syntax {
			// Check if this file contains our symbol
			if f.fileContainsSymbol(file, symbol) {
				// Find all references in this file
				ast.Inspect(file, func(n ast.Node) bool {
					switch node := n.(type) {
					case *ast.Ident:
						if obj := pkg.TypesInfo.Uses[node]; obj != nil {
							kind := f.getObjectKind(obj)
							pkgPath := ""
							if obj.Pkg() != nil {
								pkgPath = obj.Pkg().Path()
							}
							refKey := f.getSymbolKey(pkgPath, obj.Name(), kind)

							// Only add if it's a different symbol
							if refKey != symbolKey {
								referenced = append(referenced, refKey)
							}
						}
					case *ast.SelectorExpr:
						if obj := pkg.TypesInfo.Uses[node.Sel]; obj != nil {
							kind := f.getObjectKind(obj)
							pkgPath := ""
							if obj.Pkg() != nil {
								pkgPath = obj.Pkg().Path()
							}
							refKey := f.getSymbolKey(pkgPath, obj.Name(), kind)

							if refKey != symbolKey {
								referenced = append(referenced, refKey)
							}
						}
					}
					return true
				})
			}
		}
	}

	return referenced
}

func (f *OrphanedCodeFinder) fileContainsSymbol(file *ast.File, symbol *Symbol) bool {
	// Simple check: if the file contains a declaration with the symbol's name
	contains := false
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name != nil && node.Name.Name == symbol.Name && symbol.Kind == "function" {
				contains = true
				return false
			}
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name != nil && s.Name.Name == symbol.Name && symbol.Kind == "type" {
						contains = true
						return false
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name != nil && name.Name == symbol.Name &&
							(symbol.Kind == "variable" || symbol.Kind == "constant") {
							contains = true
							return false
						}
					}
				}
			}
		}
		return true
	})
	return contains
}

func (f *OrphanedCodeFinder) FindOrphans() []*Symbol {
	var orphans []*Symbol

	for key, symbol := range f.symbols {
		// Skip test functions as they have their own entry points
		if strings.HasPrefix(symbol.Name, "Test") ||
			strings.HasPrefix(symbol.Name, "Benchmark") ||
			strings.HasPrefix(symbol.Name, "Example") {
			continue
		}

		// If the symbol is not reachable from any main package, it's orphaned
		if !f.reachable[key] {
			orphans = append(orphans, symbol)
		}
	}

	return orphans
}

func (f *OrphanedCodeFinder) getSymbolKey(pkgPath, name, kind string) string {
	return fmt.Sprintf("%s.%s.%s", pkgPath, name, kind)
}

func (f *OrphanedCodeFinder) getObjectKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		return "function"
	case *types.TypeName:
		return "type"
	case *types.Const:
		return "constant"
	case *types.Var:
		return "variable"
	default:
		return "unknown"
	}
}

func (f *OrphanedCodeFinder) isMainPackage(pkgPath string) bool {
	for _, pkg := range f.packages {
		if pkg.PkgPath == pkgPath {
			return pkg.Name == "main"
		}
	}
	return false
}

func (f *OrphanedCodeFinder) PrintOrphans(orphans []*Symbol) {
	if len(orphans) == 0 {
		fmt.Println("\n✅ No orphaned code found!")
		fmt.Println("All symbols are reachable from main package entry points.")
		return
	}

	fmt.Printf("\n🗑️  ORPHANED CODE ANALYSIS\n")
	fmt.Printf("Found %d symbols that are NOT reachable from any main package:\n\n", len(orphans))

	// Group by kind
	kindGroups := make(map[string][]*Symbol)
	for _, orphan := range orphans {
		kindGroups[orphan.Kind] = append(kindGroups[orphan.Kind], orphan)
	}

	for kind, symbols := range kindGroups {
		fmt.Printf("=== %s%s ===\n", strings.ToUpper(kind[:1]), kind[1:]+"s")
		for _, symbol := range symbols {
			relPath, err := filepath.Rel(f.projectPath, symbol.File)
			if err != nil {
				relPath = symbol.File
			}

			exportStatus := "private"
			if symbol.Exported {
				exportStatus = "exported"
			}

			fmt.Printf("  📍 %s (%s) - %s\n",
				symbol.Name,
				exportStatus,
				formatPosition(relPath, symbol.Position))
		}
		fmt.Println()
	}

	fmt.Println("💡 These symbols are not reachable from any main() or init() function.")
	fmt.Println("💡 Test functions are excluded as they have separate entry points.")

	if len(f.mainPackages) > 0 {
		fmt.Printf("💡 Analysis based on %d main package(s) found in the project.\n", len(f.mainPackages))
	}
}

func formatPosition(file string, pos token.Position) string {
	return fmt.Sprintf("%s:%d:%d", file, pos.Line, pos.Column)
}
