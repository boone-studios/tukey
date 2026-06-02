// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package analyzer

import (
	"sort"

	"github.com/boone-studios/tukey/internal/models"
)

// FindCycles traces dependency loops in the graph using DFS recursion
func FindCycles(graph *models.DependencyGraph) [][]string {
	if graph == nil {
		return nil
	}

	graph.RLock()
	defer graph.RUnlock()

	// 0 = unvisited, 1 = visiting, 2 = visited
	state := make(map[string]int)
	var cycles [][]string
	var stack []string

	var dfs func(nodeID string)
	dfs = func(nodeID string) {
		state[nodeID] = 1
		stack = append(stack, nodeID)

		node := graph.Nodes[nodeID]
		if node != nil {
			var depIDs []string
			for depID := range node.Dependencies {
				depIDs = append(depIDs, depID)
			}
			sort.Strings(depIDs)

			for _, depID := range depIDs {
				if state[depID] == 1 {
					// Loop detected
					cycleStartIdx := -1
					for idx, sID := range stack {
						if sID == depID {
							cycleStartIdx = idx
							break
						}
					}
					if cycleStartIdx != -1 {
						cyclePath := append([]string{}, stack[cycleStartIdx:]...)
						cyclePath = append(cyclePath, depID)
						cycles = append(cycles, cyclePath)
					}
				} else if state[depID] == 0 {
					dfs(depID)
				}
			}
		}

		stack = stack[:len(stack)-1]
		state[nodeID] = 2
	}

	var nodeIDs []string
	for nodeID := range graph.Nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	for _, nodeID := range nodeIDs {
		if state[nodeID] == 0 {
			dfs(nodeID)
		}
	}

	return cycles
}
