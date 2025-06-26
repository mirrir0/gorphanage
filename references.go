package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// findReferences discovers all symbol references in the project
func (a *Analyzer) findReferences() error {
	for _, pkg := range a.packages {
		for _, file := range pkg.Syntax {
			a.findReferencesInFile(pkg, file)
		}
	}
	return nil
}

// findReferencesInFile finds all symbol references in a single file
func (a *Analyzer) findReferencesInFile(pkg *packages.Package, file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			a.processIdentReference(pkg, node)
		case *ast.SelectorExpr:
			a.processSelectorReference(pkg, node)
		}
		return true
	})
}

// processIdentReference processes identifier references
func (a *Analyzer) processIdentReference(pkg *packages.Package, node *ast.Ident) {
	// Check if this identifier is being used (not declared)
	obj := pkg.TypesInfo.Uses[node]
	if obj == nil {
		return
	}

	pos := a.fileSet.Position(node.Pos())
	kind := a.getObjectKind(obj)

	// Get package path, handling nil package
	pkgPath := ""
	if obj.Pkg() != nil {
		pkgPath = obj.Pkg().Path()
	}

	key := a.getSymbolKey(pkgPath, obj.Name(), kind)

	a.references[key] = append(a.references[key], Reference{
		File:     pos.Filename,
		Position: pos,
	})
}

// processSelectorReference processes selector expression references (pkg.Symbol)
func (a *Analyzer) processSelectorReference(pkg *packages.Package, node *ast.SelectorExpr) {
	obj := pkg.TypesInfo.Uses[node.Sel]
	if obj == nil {
		return
	}

	pos := a.fileSet.Position(node.Sel.Pos())
	kind := a.getObjectKind(obj)

	// Get package path, handling nil package
	pkgPath := ""
	if obj.Pkg() != nil {
		pkgPath = obj.Pkg().Path()
	}

	key := a.getSymbolKey(pkgPath, obj.Name(), kind)

	a.references[key] = append(a.references[key], Reference{
		File:     pos.Filename,
		Position: pos,
	})
}

// getObjectKind determines the kind of a types.Object
func (a *Analyzer) getObjectKind(obj types.Object) string {
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
