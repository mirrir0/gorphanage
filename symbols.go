package main

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// findSymbols discovers all symbols in the project
func (a *Analyzer) findSymbols() error {
	for _, pkg := range a.packages {
		for i, file := range pkg.Syntax {
			if i < len(pkg.CompiledGoFiles) {
				a.findSymbolsInFile(pkg, file, pkg.CompiledGoFiles[i])
			}
		}
	}
	return nil
}

// findSymbolsInFile extracts symbols from a single file
func (a *Analyzer) findSymbolsInFile(pkg *packages.Package, file *ast.File, filename string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			a.processFunctionDecl(pkg, node, filename)
		case *ast.GenDecl:
			a.processGenDecl(pkg, node, filename)
		}
		return true
	})
}

// processFunctionDecl processes function declarations
func (a *Analyzer) processFunctionDecl(pkg *packages.Package, node *ast.FuncDecl, filename string) {
	if node.Name == nil || node.Name.Name == "_" {
		return
	}

	startPos := a.fileSet.Position(node.Pos())
	endPos := a.fileSet.Position(node.End())

	symbol := &Symbol{
		Name:     node.Name.Name,
		Kind:     "function",
		File:     filename,
		Position: startPos,
		Start: Position{
			Line:   startPos.Line,
			Column: startPos.Column,
		},
		End: Position{
			Line:   endPos.Line,
			Column: endPos.Column,
		},
		Exported: ast.IsExported(node.Name.Name),
		Package:  pkg.PkgPath,
	}

	key := a.getSymbolKey(pkg.PkgPath, node.Name.Name, "function")
	a.symbols[key] = symbol
}

// processGenDecl processes general declarations (types, variables, constants)
func (a *Analyzer) processGenDecl(pkg *packages.Package, node *ast.GenDecl, filename string) {
	for _, spec := range node.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			a.processTypeSpec(pkg, s, filename)
		case *ast.ValueSpec:
			a.processValueSpec(pkg, s, node.Tok, filename)
		}
	}
}

// processTypeSpec processes type specifications
func (a *Analyzer) processTypeSpec(pkg *packages.Package, spec *ast.TypeSpec, filename string) {
	if spec.Name == nil || spec.Name.Name == "_" {
		return
	}

	startPos := a.fileSet.Position(spec.Pos())
	endPos := a.fileSet.Position(spec.End())

	symbol := &Symbol{
		Name:     spec.Name.Name,
		Kind:     "type",
		File:     filename,
		Position: startPos,
		Start: Position{
			Line:   startPos.Line,
			Column: startPos.Column,
		},
		End: Position{
			Line:   endPos.Line,
			Column: endPos.Column,
		},
		Exported: ast.IsExported(spec.Name.Name),
		Package:  pkg.PkgPath,
	}

	key := a.getSymbolKey(pkg.PkgPath, spec.Name.Name, "type")
	a.symbols[key] = symbol
}

// processValueSpec processes variable and constant specifications
func (a *Analyzer) processValueSpec(pkg *packages.Package, spec *ast.ValueSpec, tok token.Token, filename string) {
	for _, name := range spec.Names {
		if name == nil || name.Name == "_" {
			continue
		}

		startPos := a.fileSet.Position(name.Pos())
		endPos := a.fileSet.Position(name.End())

		kind := "variable"
		if tok == token.CONST {
			kind = "constant"
		}

		symbol := &Symbol{
			Name:     name.Name,
			Kind:     kind,
			File:     filename,
			Position: startPos,
			Start: Position{
				Line:   startPos.Line,
				Column: startPos.Column,
			},
			End: Position{
				Line:   endPos.Line,
				Column: endPos.Column,
			},
			Exported: ast.IsExported(name.Name),
			Package:  pkg.PkgPath,
		}

		key := a.getSymbolKey(pkg.PkgPath, name.Name, kind)
		a.symbols[key] = symbol
	}
}
