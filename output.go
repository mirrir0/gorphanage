package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PrintResults outputs the analysis results in human-readable format
func (a *Analyzer) PrintResults(result *AnalysisResult) {
	if len(result.OrphanedSymbols) == 0 {
		fmt.Println("\n✅ No orphaned code found!")
		fmt.Println("All symbols are reachable from main package entry points.")
		return
	}

	fmt.Printf("\n🗑️  ORPHANED CODE ANALYSIS\n")
	fmt.Printf("Found %d symbols that are NOT reachable from any main package:\n\n", len(result.OrphanedSymbols))

	// Group by kind
	kindGroups := make(map[string][]*Symbol)
	for _, orphan := range result.OrphanedSymbols {
		kindGroups[orphan.Kind] = append(kindGroups[orphan.Kind], orphan)
	}

	for kind, symbols := range kindGroups {
		fmt.Printf("=== %s%s ===\n", strings.ToUpper(kind[:1]), kind[1:]+"s")
		for _, symbol := range symbols {
			relPath, err := filepath.Rel(a.config.ProjectPath, symbol.File)
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
				formatPosition(relPath, symbol.Start))
		}
		fmt.Println()
	}

	a.printSummary(result)
}

// printSummary prints analysis summary and helpful tips
func (a *Analyzer) printSummary(result *AnalysisResult) {
	fmt.Println("💡 These symbols are not reachable from any main() or init() function.")
	fmt.Println("💡 Test functions are excluded as they have separate entry points.")

	if result.MainPackages > 0 {
		fmt.Printf("💡 Analysis based on %d main package(s) found in the project.\n", result.MainPackages)
	}

	// Additional statistics
	fmt.Printf("\n📊 Analysis Summary:\n")
	fmt.Printf("  • Total symbols: %d\n", result.TotalSymbols)
	fmt.Printf("  • Reachable symbols: %d\n", result.ReachableSymbols)
	fmt.Printf("  • Orphaned symbols: %d\n", len(result.OrphanedSymbols))

	if result.TotalSymbols > 0 {
		orphanPercentage := float64(len(result.OrphanedSymbols)) / float64(result.TotalSymbols) * 100
		fmt.Printf("  • Orphan rate: %.1f%%\n", orphanPercentage)
	}
}

// formatPosition formats a position for display
func formatPosition(file string, pos Position) string {
	return fmt.Sprintf("%s:%d:%d", file, pos.Line, pos.Column)
}
