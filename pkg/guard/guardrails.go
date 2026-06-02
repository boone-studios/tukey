// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package guard

import (
	"fmt"
	"strings"

	"github.com/boone-studios/tukey/internal/config"
	"github.com/boone-studios/tukey/internal/models"
)

// Violation represents a forbidden dependency transition between architectural layers
type Violation struct {
	SourceNode  *models.DependencyNode
	TargetNode  *models.DependencyNode
	SourceLayer string
	TargetLayer string
	Reason      string
}

// ValidateBoundaries checks a dependency graph against configured layer rules
func ValidateBoundaries(graph *models.DependencyGraph, archConfig config.ArchitectureConfig) []Violation {
	if graph == nil || len(archConfig.Layers) == 0 {
		return nil
	}

	graph.RLock()
	defer graph.RUnlock()

	// Map rules for fast lookup: sourceLayer -> forbiddenTargetLayers
	rulesMap := make(map[string]map[string]bool)
	for _, rule := range archConfig.Rules {
		if _, exists := rulesMap[rule.From]; !exists {
			rulesMap[rule.From] = make(map[string]bool)
		}
		for _, forbidden := range rule.CannotDependOn {
			rulesMap[rule.From][forbidden] = true
		}
	}

	var violations []Violation

	// Check each node in the graph
	for _, node := range graph.Nodes {
		srcLayer := getLayerForNode(node, archConfig.Layers)
		if srcLayer == "" {
			continue
		}

		forbiddenTargets, hasRules := rulesMap[srcLayer]
		if !hasRules {
			continue
		}

		// Check outbound dependencies
		for depID := range node.Dependencies {
			depNode := graph.Nodes[depID]
			if depNode == nil {
				continue
			}

			tgtLayer := getLayerForNode(depNode, archConfig.Layers)
			if tgtLayer == "" || tgtLayer == srcLayer {
				continue
			}

			if forbiddenTargets[tgtLayer] {
				violations = append(violations, Violation{
					SourceNode:  node,
					TargetNode:  depNode,
					SourceLayer: srcLayer,
					TargetLayer: tgtLayer,
					Reason:      fmt.Sprintf("layer '%s' is forbidden from depending on layer '%s'", srcLayer, tgtLayer),
				})
			}
		}
	}

	return violations
}

func getLayerForNode(node *models.DependencyNode, layers []config.LayerConfig) string {
	nodeFile := strings.ReplaceAll(node.File, "\\", "/")
	for _, layer := range layers {
		layerPath := strings.ReplaceAll(layer.Path, "\\", "/")
		// Match if the file path contains the layer directory path
		if strings.Contains(nodeFile, layerPath) {
			return layer.Name
		}
	}
	return ""
}

// PrintViolations prints architectural violations to the console
func PrintViolations(violations []Violation) {
	if len(violations) == 0 {
		fmt.Println("✅ No architectural boundary violations detected")
		return
	}

	fmt.Println("\n======================================================================")
	fmt.Printf("🚨 ARCHITECTURAL LAYER BOUNDARY VIOLATIONS (%d detected)\n", len(violations))
	fmt.Println("======================================================================")

	for i, v := range violations {
		fmt.Printf("   %d. [%s] %s (%s:%d)\n", i+1, v.SourceLayer, v.SourceNode.Name, v.SourceNode.File, v.SourceNode.Line)
		fmt.Printf("      → depends on [%s] %s (%s:%d)\n", v.TargetLayer, v.TargetNode.Name, v.TargetNode.File, v.TargetNode.Line)
		fmt.Printf("      Reason: %s\n\n", v.Reason)
	}
	fmt.Println("======================================================================")
}
