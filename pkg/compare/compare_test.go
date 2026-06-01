// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package compare

import (
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func TestCompareGraphs_AddedAndDeletedNodes(t *testing.T) {
	// Baseline graph with 2 nodes: A (class) and B (method)
	oldGraph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{
			"class:App\\Models\\User:8": {
				ID:        "class:App\\Models\\User:8",
				Name:      "User",
				Type:      "class",
				Namespace: "App\\Models",
				Score:     10,
			},
			"method:App\\Models\\User::save:20": {
				ID:        "method:App\\Models\\User::save:20",
				Name:      "save",
				Type:      "method",
				Namespace: "App\\Models",
				ClassName: "User",
				Score:     5,
			},
		},
		TotalNodes: 2,
		TotalEdges: 0,
	}

	// New graph:
	// - Class User is retained (but line number changed: 8 -> 10). Stable matching should match it.
	// - Method save is deleted.
	// - New method load is added.
	// - Class User complexity score increases from 10 to 12.
	newGraph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{
			"class:App\\Models\\User:10": {
				ID:        "class:App\\Models\\User:10",
				Name:      "User",
				Type:      "class",
				Namespace: "App\\Models",
				Score:     12, // Score increased!
			},
			"method:App\\Models\\User::load:30": {
				ID:        "method:App\\Models\\User::load:30",
				Name:      "load",
				Type:      "method",
				Namespace: "App\\Models",
				ClassName: "User",
				Score:     4,
			},
		},
		TotalNodes: 2,
		TotalEdges: 0,
	}

	report := CompareGraphs(oldGraph, newGraph)

	if len(report.AddedNodes) != 1 {
		t.Errorf("expected 1 added node, got %d", len(report.AddedNodes))
	} else if report.AddedNodes[0].Name != "load" {
		t.Errorf("expected added node to be 'load', got %s", report.AddedNodes[0].Name)
	}

	if len(report.DeletedNodes) != 1 {
		t.Errorf("expected 1 deleted node, got %d", len(report.DeletedNodes))
	} else if report.DeletedNodes[0].Name != "save" {
		t.Errorf("expected deleted node to be 'save', got %s", report.DeletedNodes[0].Name)
	}

	if len(report.ComplexityRegressions) != 1 {
		t.Errorf("expected 1 complexity regression, got %d", len(report.ComplexityRegressions))
	} else {
		diff := report.ComplexityRegressions[0]
		if diff.Name != "User" || diff.OldScore != 10 || diff.NewScore != 12 {
			t.Errorf("expected User complexity change from 10 -> 12, got %s: %d -> %d", diff.Name, diff.OldScore, diff.NewScore)
		}
	}
}

func TestCompareGraphs_Orphans(t *testing.T) {
	oldGraph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{},
		Orphans: []*models.DependencyNode{
			{
				ID:        "class:App\\OldOrphan:5",
				Name:      "OldOrphan",
				Type:      "class",
				Namespace: "App",
			},
		},
	}

	newGraph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{},
		Orphans: []*models.DependencyNode{
			{
				ID:        "class:App\\NewOrphan:15",
				Name:      "NewOrphan",
				Type:      "class",
				Namespace: "App",
			},
		},
	}

	report := CompareGraphs(oldGraph, newGraph)

	if len(report.AddedOrphans) != 1 {
		t.Errorf("expected 1 added orphan, got %d", len(report.AddedOrphans))
	} else if report.AddedOrphans[0].Name != "NewOrphan" {
		t.Errorf("expected added orphan to be 'NewOrphan', got %s", report.AddedOrphans[0].Name)
	}

	if len(report.DeletedOrphans) != 1 {
		t.Errorf("expected 1 deleted orphan, got %d", len(report.DeletedOrphans))
	} else if report.DeletedOrphans[0].Name != "OldOrphan" {
		t.Errorf("expected deleted orphan to be 'OldOrphan', got %s", report.DeletedOrphans[0].Name)
	}
}
