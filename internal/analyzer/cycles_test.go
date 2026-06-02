// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package analyzer

import (
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func TestFindCycles_DetectsLoop(t *testing.T) {
	// Construct a synthetic cycle: A -> B -> C -> A
	graph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{
			"class:App\\A:5": {
				ID:   "class:App\\A:5",
				Name: "A",
				Dependencies: map[string]*models.DependencyRef{
					"class:App\\B:10": {TargetID: "class:App\\B:10"},
				},
			},
			"class:App\\B:10": {
				ID:   "class:App\\B:10",
				Name: "B",
				Dependencies: map[string]*models.DependencyRef{
					"class:App\\C:15": {TargetID: "class:App\\C:15"},
				},
			},
			"class:App\\C:15": {
				ID:   "class:App\\C:15",
				Name: "C",
				Dependencies: map[string]*models.DependencyRef{
					"class:App\\A:5": {TargetID: "class:App\\A:5"},
				},
			},
		},
	}

	cycles := FindCycles(graph)

	if len(cycles) != 1 {
		t.Fatalf("expected exactly 1 cycle, got %d", len(cycles))
	}

	cycle := cycles[0]
	// Expected loop length is 4: A -> B -> C -> A
	if len(cycle) != 4 {
		t.Errorf("expected cycle path length of 4, got %d", len(cycle))
	}

	if cycle[0] != "class:App\\A:5" || cycle[3] != "class:App\\A:5" {
		t.Errorf("expected cycle to start and end at App\\A, got: %v", cycle)
	}
}
