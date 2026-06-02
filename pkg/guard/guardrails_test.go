// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package guard

import (
	"testing"

	"github.com/boone-studios/tukey/internal/config"
	"github.com/boone-studios/tukey/internal/models"
)

func TestValidateBoundaries(t *testing.T) {
	// Setup architecture configuration
	archConfig := config.ArchitectureConfig{
		Layers: []config.LayerConfig{
			{Name: "Domain", Path: "src/Domain"},
			{Name: "Infrastructure", Path: "src/Infra"},
		},
		Rules: []config.RuleConfig{
			{
				From:           "Domain",
				CannotDependOn: []string{"Infrastructure"},
			},
		},
	}

	// Case 1: Forbidden dependency (Domain node -> Infra node)
	domainNode := &models.DependencyNode{
		ID:        "class:App\\Domain\\User:5",
		Name:      "User",
		Type:      "class",
		File:      "/myproject/src/Domain/User.php",
		Namespace: "App\\Domain",
		Dependencies: map[string]*models.DependencyRef{
			"class:App\\Infra\\Database:10": {
				TargetID:   "class:App\\Infra\\Database:10",
				TargetName: "Database",
				Type:       "instantiation",
			},
		},
	}

	infraNode := &models.DependencyNode{
		ID:        "class:App\\Infra\\Database:10",
		Name:      "Database",
		Type:      "class",
		File:      "/myproject/src/Infra/Database.php",
		Namespace: "App\\Infra",
	}

	// Case 2: Allowed dependency (Domain node -> Domain node)
	allowedDomainNode := &models.DependencyNode{
		ID:        "class:App\\Domain\\UserRepository:8",
		Name:      "UserRepository",
		Type:      "class",
		File:      "/myproject/src/Domain/UserRepository.php",
		Namespace: "App\\Domain",
		Dependencies: map[string]*models.DependencyRef{
			"class:App\\Domain\\User:5": {
				TargetID:   "class:App\\Domain\\User:5",
				TargetName: "User",
				Type:       "calls",
			},
		},
	}

	graph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{
			domainNode.ID:        domainNode,
			infraNode.ID:         infraNode,
			allowedDomainNode.ID: allowedDomainNode,
		},
	}

	violations := ValidateBoundaries(graph, archConfig)

	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 architectural boundary violation, got %d", len(violations))
	}

	v := violations[0]
	if v.SourceLayer != "Domain" || v.TargetLayer != "Infrastructure" {
		t.Errorf("expected violation from Domain to Infrastructure, got: srcLayer=%s tgtLayer=%s", v.SourceLayer, v.TargetLayer)
	}

	if v.SourceNode.Name != "User" || v.TargetNode.Name != "Database" {
		t.Errorf("expected User -> Database violation, got: srcNode=%s tgtNode=%s", v.SourceNode.Name, v.TargetNode.Name)
	}
}
