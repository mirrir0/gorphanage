package main

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

// traceReachability performs BFS from main package entry points to find reachable symbols
func (a *Analyzer) traceReachability() error {
	if a.config.Verbose && !a.config.OutputJSON {
		fmt.Println("🔍 Tracing reachability from main packages...")
	}

	// Start from all entry points in main packages
	queue := a.findEntryPoints()

	if a.config.Verbose && !a.config.OutputJSON {
		fmt.Printf("🎯 Starting with %d entry points\n", len(queue))
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
		referencedSymbols := a.findReferencedSymbols(current)

		for _, refSymbol := range referencedSymbols {
			if !a.reachable[refSymbol] {
				a.reachable[refSymbol] = true
				queue = append(queue, refSymbol)
			}
		}
	}

	reachableCount := len(a.reachable)
	totalCount := len(a.symbols)
	if a.config.Verbose && !a.config.OutputJSON {
		fmt.Printf("📊 Reachability analysis: %d/%d symbols reachable from main packages\n",
			reachableCount, totalCount)
	}

	return nil
}

// findEntryPoints identifies all entry points for reachability analysis
func (a *Analyzer) findEntryPoints() []string {
	var queue []string

	// Add main functions and init functions as entry points
	for _, pkg := range a.mainPackages {
		mainKey := a.getSymbolKey(pkg.PkgPath, "main", "function")
		if _, exists := a.symbols[mainKey]; exists {
			queue = append(queue, mainKey)
			a.reachable[mainKey] = true
		}

		// Also add init functions as entry points
		initKey := a.getSymbolKey(pkg.PkgPath, "init", "function")
		if _, exists := a.symbols[initKey]; exists {
			queue = append(queue, initKey)
			a.reachable[initKey] = true
		}

		// Add all exported symbols from main packages as potentially reachable
		// (they might be called by tests or external tools)
		for symbolKey, symbol := range a.symbols {
			if symbol.Package == pkg.PkgPath && symbol.Exported {
				if !a.reachable[symbolKey] {
					queue = append(queue, symbolKey)
					a.reachable[symbolKey] = true
				}
			}
		}
	}

	return queue
}

// findReferencedSymbols finds all symbols referenced by a given symbol
func (a *Analyzer) findReferencedSymbols(symbolKey string) []string {
	var referenced []string

	// Get the symbol to find its file(s)
	symbol, exists := a.symbols[symbolKey]
	if !exists {
		return referenced
	}

	// Find all packages that might contain this symbol's definition
	var relevantPackages []*packages.Package
	for _, pkg := range a.packages {
		if pkg.PkgPath == symbol.Package {
			relevantPackages = append(relevantPackages, pkg)
		}
	}

	// Look through the symbol's definition files for references to other symbols
	for _, pkg := range relevantPackages {
		for _, file := range pkg.Syntax {
			// Check if this file contains our symbol
			if a.fileContainsSymbol(file, symbol) {
				// Find all references in this file
				ast.Inspect(file, func(n ast.Node) bool {
					switch node := n.(type) {
					case *ast.Ident:
						if obj := pkg.TypesInfo.Uses[node]; obj != nil {
							kind := a.getObjectKind(obj)
							pkgPath := ""
							if obj.Pkg() != nil {
								pkgPath = obj.Pkg().Path()
							}
							refKey := a.getSymbolKey(pkgPath, obj.Name(), kind)

							// Only add if it's a different symbol
							if refKey != symbolKey {
								referenced = append(referenced, refKey)
							}
						}
					case *ast.SelectorExpr:
						if obj := pkg.TypesInfo.Uses[node.Sel]; obj != nil {
							kind := a.getObjectKind(obj)
							pkgPath := ""
							if obj.Pkg() != nil {
								pkgPath = obj.Pkg().Path()
							}
							refKey := a.getSymbolKey(pkgPath, obj.Name(), kind)

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

// fileContainsSymbol checks if a file contains the declaration of a given symbol
func (a *Analyzer) fileContainsSymbol(file *ast.File, symbol *Symbol) bool {
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

// findOrphans identifies symbols that are not reachable from main packages
func (a *Analyzer) findOrphans() []*Symbol {
	var orphans []*Symbol

	for key, symbol := range a.symbols {
		// Skip test functions as they have their own entry points
		if a.isTestFunction(symbol.Name) {
			continue
		}

		// If the symbol is not reachable from any main package, it's orphaned
		if !a.reachable[key] {
			orphans = append(orphans, symbol)
		}
	}

	return orphans
}

// isTestFunction checks if a function name indicates it's a test function
func (a *Analyzer) isTestFunction(name string) bool {
	return strings.HasPrefix(name, "Test") ||
		strings.HasPrefix(name, "Benchmark") ||
		strings.HasPrefix(name, "Example")
}
