// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package output

import (
	"fmt"
	"strings"

	"github.com/boone-studios/tukey/internal/models"
)

// ConsoleFormatter handles console output formatting
type ConsoleFormatter struct{}

// NewConsoleFormatter creates a new console formatter
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{}
}

// PrintSummary displays a human-readable summary of the analysis results
func (cf *ConsoleFormatter) PrintSummary(result *models.AnalysisResult, verbose bool) {
	graph := result.Graph

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("DEPENDENCY ANALYSIS SUMMARY")
	fmt.Println(strings.Repeat("=", 70))

	fmt.Printf("📊 Graph Statistics:\n")
	fmt.Printf("   • Total Nodes: %d\n", graph.TotalNodes)
	fmt.Printf("   • Total Dependencies: %d\n", graph.TotalEdges)
	fmt.Printf("   • Orphaned Elements: %d\n", len(graph.Orphans))

	// Determine how many items to show
	maxHighlyDepended := 5
	maxComplexNodes := 5
	maxOrphans := 10
	maxDependentsToShow := 3

	if verbose {
		maxHighlyDepended = len(graph.HighlyDepended)
		maxComplexNodes = len(graph.ComplexNodes)
		maxOrphans = len(graph.Orphans)
		maxDependentsToShow = -1 // Show all
		fmt.Printf("\n🔍 VERBOSE MODE: Showing complete dependency lists\n")
	}

	fmt.Printf("\n🔥 Most Depended Upon Elements:\n")
	for i, node := range graph.HighlyDepended {
		if i >= maxHighlyDepended {
			if !verbose {
				fmt.Printf("   ... and %d more (use -v for full list)\n", len(graph.HighlyDepended)-maxHighlyDepended)
			}
			break
		}

		relativePath := strings.TrimPrefix(node.File, "/")
		if strings.HasPrefix(relativePath, "/") {
			relativePath = relativePath[1:] // Remove leading slash if still present
		}

		fmt.Printf("   %d. %s (%s) - %d dependents\n",
			i+1, node.Name, relativePath, len(node.Dependents))

		// Show dependents
		dependentCount := 0
		for _, dep := range node.Dependents {
			if maxDependentsToShow > 0 && dependentCount >= maxDependentsToShow {
				fmt.Printf("      ... and %d more dependents\n", len(node.Dependents)-maxDependentsToShow)
				break
			}
			fmt.Printf("      ← %s (%s)\n", dep.TargetName, dep.Type)
			dependentCount++
		}

		if verbose && i < len(graph.HighlyDepended)-1 {
			fmt.Println() // Add spacing between entries in verbose mode
		}
	}

	fmt.Printf("\n🧠 Most Complex Elements:\n")
	for i, node := range graph.ComplexNodes {
		if i >= maxComplexNodes {
			if !verbose {
				fmt.Printf("   ... and %d more (use -v for full list)\n", len(graph.ComplexNodes)-maxComplexNodes)
			}
			break
		}

		relativePath := strings.TrimPrefix(node.File, "/")
		if strings.HasPrefix(relativePath, "/") {
			relativePath = relativePath[1:]
		}

		fmt.Printf("   %d. %s (%s) - Score: %d\n",
			i+1, node.Name, relativePath, node.Score)
		fmt.Printf("      Dependencies: %d, Dependents: %d\n",
			len(node.Dependencies), len(node.Dependents))

		if verbose {
			// Show what this node depends on
			if len(node.Dependencies) > 0 {
				fmt.Printf("      Depends on:\n")
				for _, dep := range node.Dependencies {
					fmt.Printf("        → %s (%s, %d times)\n", dep.TargetName, dep.Type, dep.Count)
				}
			}

			// Show what depends on this node
			if len(node.Dependents) > 0 {
				fmt.Printf("      Depended upon by:\n")
				depCount := 0
				for _, dep := range node.Dependents {
					if depCount >= 10 { // Limit even in verbose mode for readability
						fmt.Printf("        ... and %d more\n", len(node.Dependents)-10)
						break
					}
					fmt.Printf("        ← %s (%s, %d times)\n", dep.TargetName, dep.Type, dep.Count)
					depCount++
				}
			}

			if i < len(graph.ComplexNodes)-1 {
				fmt.Println() // Add spacing between entries
			}
		}
	}

	if len(graph.Orphans) > 0 {
		fmt.Printf("\n👻 Orphaned Elements (%d total):\n", len(graph.Orphans))
		for i, node := range graph.Orphans {
			if i >= maxOrphans {
				if !verbose {
					fmt.Printf("   ... and %d more (use -v for full list)\n", len(graph.Orphans)-maxOrphans)
				}
				break
			}

			relativePath := strings.TrimPrefix(node.File, "/")
			if strings.HasPrefix(relativePath, "/") {
				relativePath = relativePath[1:]
			}

			if verbose {
				fmt.Printf("   • %s (%s) in %s (line %d)\n", node.Name, node.Type, relativePath, node.Line)
			} else {
				fmt.Printf("   • %s (%s) in %s\n", node.Name, node.Type, relativePath)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 70))

	// Add a function usage report in verbose mode
	if verbose {
		cf.PrintFunctionUsageReport(result)
	}

	if !verbose {
		fmt.Printf("💡 Tip: Use -v or --verbose flag to see complete dependency lists and function usage report\n")
		fmt.Println(strings.Repeat("=", 70))
	}
}

// PrintFunctionUsageReport shows detailed function usage across the codebase
func (cf *ConsoleFormatter) PrintFunctionUsageReport(result *models.AnalysisResult) {
	fmt.Printf("\n📋 FUNCTION USAGE REPORT\n")
	fmt.Println(strings.Repeat("=", 70))

	// We will need to get this data from the result
	// For now, let's create a placeholder
	fmt.Printf("   Function usage report not yet implemented in output package.\n")
	fmt.Printf("   This functionality will be moved here from the analyzer.\n")

	fmt.Println(strings.Repeat("=", 70))
}
