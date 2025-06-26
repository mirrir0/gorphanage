package main

import (
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Config holds the configuration for the analysis
type Config struct {
	ProjectPath  string
	OutputJSON   bool
	Verbose      bool
	Exclude      []string
	IncludeTests bool
}

// Symbol represents a code symbol (function, type, variable, constant)
type Symbol struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"` // "function", "variable", "type", "constant"
	File     string   `json:"file"`
	Start    Position `json:"start"`
	End      Position `json:"end"`
	Exported bool     `json:"exported"`
	Package  string   `json:"package"`

	// Internal fields (not serialized)
	Position token.Position `json:"-"`
}

// Position represents a line:column position in a file
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Reference represents a usage of a symbol
type Reference struct {
	File     string
	Position token.Position
}

// AnalysisResult contains the complete analysis results
type AnalysisResult struct {
	ProjectPath      string    `json:"project_path"`
	TotalSymbols     int       `json:"total_symbols"`
	ReachableSymbols int       `json:"reachable_symbols"`
	MainPackages     int       `json:"main_packages"`
	OrphanedSymbols  []*Symbol `json:"orphaned_symbols"`
	ExcludedPackages []string  `json:"excluded_packages,omitempty"`
	IncludedTests    bool      `json:"included_tests"`
}

// Analyzer performs the orphaned code analysis
type Analyzer struct {
	config       *Config
	fileSet      *token.FileSet
	packages     []*packages.Package
	symbols      map[string]*Symbol
	references   map[string][]Reference
	reachable    map[string]bool
	mainPackages []*packages.Package
}
