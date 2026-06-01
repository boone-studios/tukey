// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package compare

import (
	"fmt"
	"strings"

	"github.com/boone-studios/tukey/internal/models"
)

// ComplexityDiff represents a change in complexity score for a code element
type ComplexityDiff struct {
	Name     string
	Type     string
	File     string
	Line     int
	OldScore int
	NewScore int
}

// ComparisonReport captures structural and metric differences between two graphs
type ComparisonReport struct {
	OldNodesCount int
	NewNodesCount int
	OldEdgesCount int
	NewEdgesCount int

	AddedNodes   []*models.DependencyNode
	DeletedNodes []*models.DependencyNode

	AddedOrphans   []*models.DependencyNode
	DeletedOrphans []*models.DependencyNode

	ComplexityRegressions  []ComplexityDiff
	ComplexityImprovements []ComplexityDiff
}

// GetStableSignature constructs a line-number independent, unique signature for a node
func GetStableSignature(node *models.DependencyNode) string {
	if node.ClassName != "" {
		fqcn := node.Namespace
		if fqcn != "" {
			fqcn += "\\"
		}
		fqcn += node.ClassName
		return fmt.Sprintf("%s:%s::%s", node.Type, fqcn, node.Name)
	}

	fqn := node.Namespace
	if fqn != "" {
		fqn += "\\"
	}
	fqn += node.Name
	return fmt.Sprintf("%s:%s", node.Type, fqn)
}

// CompareGraphs parses the baseline and active graphs to produce a delta comparison
func CompareGraphs(oldGraph, newGraph *models.DependencyGraph) *ComparisonReport {
	report := &ComparisonReport{
		OldNodesCount: oldGraph.TotalNodes,
		NewNodesCount: newGraph.TotalNodes,
		OldEdgesCount: oldGraph.TotalEdges,
		NewEdgesCount: newGraph.TotalEdges,
	}

	// Index nodes by stable signature
	oldIndex := make(map[string]*models.DependencyNode)
	for _, node := range oldGraph.Nodes {
		sig := GetStableSignature(node)
		oldIndex[sig] = node
	}

	newIndex := make(map[string]*models.DependencyNode)
	for _, node := range newGraph.Nodes {
		sig := GetStableSignature(node)
		newIndex[sig] = node
	}

	// Find added and modified nodes
	for sig, newNode := range newIndex {
		oldNode, exists := oldIndex[sig]
		if !exists {
			report.AddedNodes = append(report.AddedNodes, newNode)
		} else {
			// Compare complexity
			if newNode.Score > oldNode.Score {
				report.ComplexityRegressions = append(report.ComplexityRegressions, ComplexityDiff{
					Name:     newNode.Name,
					Type:     newNode.Type,
					File:     newNode.File,
					Line:     newNode.Line,
					OldScore: oldNode.Score,
					NewScore: newNode.Score,
				})
			} else if newNode.Score < oldNode.Score {
				report.ComplexityImprovements = append(report.ComplexityImprovements, ComplexityDiff{
					Name:     newNode.Name,
					Type:     newNode.Type,
					File:     newNode.File,
					Line:     newNode.Line,
					OldScore: oldNode.Score,
					NewScore: newNode.Score,
				})
			}
		}
	}

	// Find deleted nodes
	for sig, oldNode := range oldIndex {
		if _, exists := newIndex[sig]; !exists {
			report.DeletedNodes = append(report.DeletedNodes, oldNode)
		}
	}

	// Map old orphans and new orphans by stable signature
	oldOrphansIndex := make(map[string]*models.DependencyNode)
	for _, orphan := range oldGraph.Orphans {
		sig := GetStableSignature(orphan)
		oldOrphansIndex[sig] = orphan
	}

	newOrphansIndex := make(map[string]*models.DependencyNode)
	for _, orphan := range newGraph.Orphans {
		sig := GetStableSignature(orphan)
		newOrphansIndex[sig] = orphan
	}

	// Find new orphans
	for sig, newOrphan := range newOrphansIndex {
		if _, exists := oldOrphansIndex[sig]; !exists {
			report.AddedOrphans = append(report.AddedOrphans, newOrphan)
		}
	}

	// Find resolved orphans
	for sig, oldOrphan := range oldOrphansIndex {
		if _, exists := newOrphansIndex[sig]; !exists {
			report.DeletedOrphans = append(report.DeletedOrphans, oldOrphan)
		}
	}

	return report
}

// PrintComparisonReport writes a beautiful console analysis comparing two graphs
func PrintComparisonReport(report *ComparisonReport, newOrphansCount int) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🕵️  TUKEY ARCHITECTURAL REGRESSION & CHANGE REPORT")
	fmt.Println(strings.Repeat("=", 70))

	oldOrphansCount := newOrphansCount - len(report.AddedOrphans) + len(report.DeletedOrphans)
	orphanDiff := newOrphansCount - oldOrphansCount

	// Section 1: Stats summary
	fmt.Printf("📊 Codebase Structural Changes:\n")
	fmt.Printf("   • Code Elements:     %d -> %d (%+d)\n", report.OldNodesCount, report.NewNodesCount, report.NewNodesCount-report.OldNodesCount)
	fmt.Printf("   • Dependencies:      %d -> %d (%+d)\n", report.OldEdgesCount, report.NewEdgesCount, report.NewEdgesCount-report.OldEdgesCount)
	fmt.Printf("   • Orphaned Elements: %d -> %d (%+d)\n", oldOrphansCount, newOrphansCount, orphanDiff)

	// Section 2: Added / Removed Elements
	if len(report.AddedNodes) > 0 {
		fmt.Printf("\n➕ Added Elements (%d):\n", len(report.AddedNodes))
		limit := len(report.AddedNodes)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			node := report.AddedNodes[i]
			fmt.Printf("   • [%s] %s (%s:%d)\n", node.Type, node.Name, node.File, node.Line)
		}
		if len(report.AddedNodes) > 10 {
			fmt.Printf("   ... and %d more elements\n", len(report.AddedNodes)-10)
		}
	}

	if len(report.DeletedNodes) > 0 {
		fmt.Printf("\n➖ Deleted Elements (%d):\n", len(report.DeletedNodes))
		limit := len(report.DeletedNodes)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			node := report.DeletedNodes[i]
			fmt.Printf("   • [%s] %s (%s:%d)\n", node.Type, node.Name, node.File, node.Line)
		}
		if len(report.DeletedNodes) > 10 {
			fmt.Printf("   ... and %d more elements\n", len(report.DeletedNodes)-10)
		}
	}

	// Section 3: Orphan / Dead Code Candidates
	if len(report.AddedOrphans) > 0 {
		fmt.Printf("\n💀 New Orphans / Dead Code Candidates (%d):\n", len(report.AddedOrphans))
		fmt.Printf("   ⚠️  These elements are unused and have no dependents. Consider cleaning them up.\n")
		for _, node := range report.AddedOrphans {
			fmt.Printf("   • [%s] %s (%s:%d)\n", node.Type, node.Name, node.File, node.Line)
		}
	}

	if len(report.DeletedOrphans) > 0 {
		fmt.Printf("\n🎉 Cleaned / Integrated Orphans (%d):\n", len(report.DeletedOrphans))
		for _, node := range report.DeletedOrphans {
			fmt.Printf("   • [%s] %s (%s)\n", node.Type, node.Name, node.File)
		}
	}

	// Section 4: Complexity Regressions and Improvements
	if len(report.ComplexityRegressions) > 0 {
		fmt.Printf("\n⚠️  Complexity Regressions (%d):\n", len(report.ComplexityRegressions))
		fmt.Printf("   ⚠️  These elements have become more complex. Consider refactoring.\n")
		limit := len(report.ComplexityRegressions)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			diff := report.ComplexityRegressions[i]
			fmt.Printf("   • [%s] %s: Score %d -> %d (+%d) (%s:%d)\n",
				diff.Type, diff.Name, diff.OldScore, diff.NewScore, diff.NewScore-diff.OldScore, diff.File, diff.Line)
		}
		if len(report.ComplexityRegressions) > 10 {
			fmt.Printf("   ... and %d more regressions\n", len(report.ComplexityRegressions)-10)
		}
	}

	if len(report.ComplexityImprovements) > 0 {
		fmt.Printf("\n✨ Complexity Improvements (%d):\n", len(report.ComplexityImprovements))
		limit := len(report.ComplexityImprovements)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			diff := report.ComplexityImprovements[i]
			fmt.Printf("   • [%s] %s: Score %d -> %d (%+d) (%s:%d)\n",
				diff.Type, diff.Name, diff.OldScore, diff.NewScore, diff.NewScore-diff.OldScore, diff.File, diff.Line)
		}
		if len(report.ComplexityImprovements) > 10 {
			fmt.Printf("   ... and %d more improvements\n", len(report.ComplexityImprovements)-10)
		}
	}

	fmt.Println(strings.Repeat("=", 70))
}
