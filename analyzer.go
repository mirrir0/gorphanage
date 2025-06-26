package main

import (
	"fmt"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer(config *Config) *Analyzer {
	return &Analyzer{
		config:     config,
		fileSet:    token.NewFileSet(),
		symbols:    make(map[string]*Symbol),
		references: make(map[string][]Reference),
		reachable:  make(map[string]bool),
	}
}

// Analyze performs the complete orphaned code analysis
func (a *Analyzer) Analyze() (*AnalysisResult, error) {
	if err := a.loadProject(); err != nil {
		return nil, fmt.Errorf("loading project: %w", err)
	}

	if a.config.Verbose && !a.config.OutputJSON {
		fmt.Printf("📦 Loaded %d packages\n", len(a.packages))
	}

	if err := a.findSymbols(); err != nil {
		return nil, fmt.Errorf("finding symbols: %w", err)
	}

	if a.config.Verbose && !a.config.OutputJSON {
		fmt.Printf("🔍 Found %d symbols\n", len(a.symbols))
	}

	if err := a.findReferences(); err != nil {
		return nil, fmt.Errorf("finding references: %w", err)
	}

	if err := a.identifyMainPackages(); err != nil {
		return nil, fmt.Errorf("identifying main packages: %w", err)
	}

	if err := a.traceReachability(); err != nil {
		return nil, fmt.Errorf("tracing reachability: %w", err)
	}

	orphans := a.findOrphans()

	result := &AnalysisResult{
		ProjectPath:      a.config.ProjectPath,
		TotalSymbols:     len(a.symbols),
		ReachableSymbols: len(a.reachable),
		MainPackages:     len(a.mainPackages),
		OrphanedSymbols:  orphans,
		ExcludedPackages: a.config.Exclude,
		IncludedTests:    a.config.IncludeTests,
	}

	return result, nil
}

// loadProject loads all packages in the project
func (a *Analyzer) loadProject() error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   a.config.ProjectPath,
		Fset:  a.fileSet,
		Tests: a.config.IncludeTests,
	}

	if a.config.Verbose && !a.config.OutputJSON {
		fmt.Printf("🔍 Loading packages from %s...\n", a.config.ProjectPath)
		if a.config.IncludeTests {
			fmt.Println("🧪 Including test files")
		}
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	// Filter out packages with errors and excluded packages
	var validPkgs []*packages.Package
	for _, pkg := range pkgs {
		// Skip packages with errors
		if len(pkg.Errors) > 0 {
			if a.config.Verbose && !a.config.OutputJSON {
				fmt.Printf("⚠️  Skipping package %s due to errors:\n", pkg.PkgPath)
				for _, err := range pkg.Errors {
					fmt.Printf("    %v\n", err)
				}
			}
			continue
		}

		// Skip excluded packages
		if a.isPackageExcluded(pkg.PkgPath) {
			if a.config.Verbose && !a.config.OutputJSON {
				fmt.Printf("📋 Excluding package %s (matches exclude pattern)\n", pkg.PkgPath)
			}
			continue
		}

		validPkgs = append(validPkgs, pkg)
	}

	a.packages = validPkgs
	return nil
}

// isPackageExcluded checks if a package should be excluded based on patterns
func (a *Analyzer) isPackageExcluded(pkgPath string) bool {
	for _, pattern := range a.config.Exclude {
		if matched, _ := filepath.Match(pattern, pkgPath); matched {
			return true
		}
		// Also check if the pattern matches any part of the path
		if strings.Contains(pkgPath, strings.Trim(pattern, "*")) {
			return true
		}
	}
	return false
}

// identifyMainPackages finds all main packages in the project
func (a *Analyzer) identifyMainPackages() error {
	for _, pkg := range a.packages {
		if pkg.Name == "main" {
			a.mainPackages = append(a.mainPackages, pkg)
		}
	}

	if len(a.mainPackages) == 0 {
		if a.config.Verbose && !a.config.OutputJSON {
			fmt.Println("⚠️  No main packages found - analyzing all packages for internal usage")
		}
		// If no main packages, treat all packages as potentially reachable
		for _, pkg := range a.packages {
			a.mainPackages = append(a.mainPackages, pkg)
		}
	} else {
		if a.config.Verbose && !a.config.OutputJSON {
			fmt.Printf("📦 Found %d main package(s)\n", len(a.mainPackages))
			for _, pkg := range a.mainPackages {
				fmt.Printf("    %s\n", pkg.PkgPath)
			}
		}
	}

	return nil
}

// getSymbolKey generates a unique key for a symbol
func (a *Analyzer) getSymbolKey(pkgPath, name, kind string) string {
	return fmt.Sprintf("%s.%s.%s", pkgPath, name, kind)
}

// isMainPackage checks if a package path represents a main package
func (a *Analyzer) isMainPackage(pkgPath string) bool {
	for _, pkg := range a.packages {
		if pkg.PkgPath == pkgPath {
			return pkg.Name == "main"
		}
	}
	return false
}
