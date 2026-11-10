// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package query

import (
	"sort"

	"github.com/boone-studios/tukey/internal/models"
)

type SymbolResolution struct {
	MatchType string       `json:"matchType"`
	Ambiguous bool         `json:"ambiguous"`
	Targets   []NodeResult `json:"targets"`
}

type EdgeResult struct {
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	Lines    []int  `json:"lines,omitempty"`
	Count    int    `json:"count"`
	Depth    int    `json:"depth,omitempty"`
}

// LocalizedContextResult holds categorized nodes and the edges connecting them.
type LocalizedContextResult struct {
	Symbol        string           `json:"symbol"`
	Resolution    SymbolResolution `json:"resolution"`
	TargetNodes   []NodeResult     `json:"targets"`
	Dependencies  []NodeResult     `json:"dependencies"`
	Dependents    []NodeResult     `json:"dependents"`
	Relationships []EdgeResult     `json:"relationships"`
}

func newSymbolResolution(targets []*models.DependencyNode, matchType string) SymbolResolution {
	results := make([]NodeResult, 0, len(targets))
	for _, target := range targets {
		results = append(results, nodeToResult(target))
	}
	sortResults(results)
	return SymbolResolution{
		MatchType: matchType,
		Ambiguous: len(targets) > 1,
		Targets:   results,
	}
}

func newEdge(sourceID, targetID string, ref *models.DependencyRef, depth int) EdgeResult {
	return EdgeResult{
		SourceID: sourceID,
		TargetID: targetID,
		Type:     ref.Type,
		Lines:    append([]int(nil), ref.Lines...),
		Count:    ref.Count,
		Depth:    depth,
	}
}

func sortEdges(edges []EdgeResult) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Depth != edges[j].Depth {
			return edges[i].Depth < edges[j].Depth
		}
		if edges[i].SourceID != edges[j].SourceID {
			return edges[i].SourceID < edges[j].SourceID
		}
		if edges[i].TargetID != edges[j].TargetID {
			return edges[i].TargetID < edges[j].TargetID
		}
		return edges[i].Type < edges[j].Type
	})
}

// Callers returns incoming nodes plus one relationship record per matched target.
func (e *Engine) Callers(term string) QueryResult {
	targets, matchType := e.findNodesWithResolution(term)
	resolution := newSymbolResolution(targets, matchType)
	seen := make(map[string]bool)
	var results []NodeResult
	relationships := []EdgeResult{}
	for _, target := range targets {
		for _, ref := range target.Dependents {
			caller := e.graph.Nodes[ref.TargetID]
			if caller == nil {
				continue
			}
			relationships = append(relationships, newEdge(caller.ID, target.ID, ref, 1))
			if seen[ref.TargetID] {
				continue
			}
			seen[ref.TargetID] = true
			r := nodeToResult(caller)
			r.RefType = ref.Type
			r.RefLines = ref.Lines
			r.RefCount = ref.Count
			results = append(results, r)
		}
	}
	sortResults(results)
	sortEdges(relationships)
	return QueryResult{
		Query: "callers", Term: term, Count: len(results), Results: coalesce(results),
		Resolution: &resolution, Relationships: relationships,
	}
}

// Dependents returns outgoing nodes plus one relationship record per matched target.
func (e *Engine) Dependents(term string) QueryResult {
	targets, matchType := e.findNodesWithResolution(term)
	resolution := newSymbolResolution(targets, matchType)
	seen := make(map[string]bool)
	var results []NodeResult
	relationships := []EdgeResult{}
	for _, target := range targets {
		for _, ref := range target.Dependencies {
			dep := e.graph.Nodes[ref.TargetID]
			if dep == nil {
				continue
			}
			relationships = append(relationships, newEdge(target.ID, dep.ID, ref, 1))
			if seen[ref.TargetID] {
				continue
			}
			seen[ref.TargetID] = true
			r := nodeToResult(dep)
			r.RefType = ref.Type
			r.RefLines = ref.Lines
			r.RefCount = ref.Count
			results = append(results, r)
		}
	}
	sortResults(results)
	sortEdges(relationships)
	return QueryResult{
		Query: "dependents", Term: term, Count: len(results), Results: coalesce(results),
		Resolution: &resolution, Relationships: relationships,
	}
}

// LocalizedContext finds dependencies and dependents up to a certain depth around a symbol.
func (e *Engine) LocalizedContext(term string, maxDepth int) LocalizedContextResult {
	if maxDepth < 1 {
		maxDepth = 1
	}

	targets, matchType := e.findNodesWithResolution(term)
	resolution := newSymbolResolution(targets, matchType)
	res := LocalizedContextResult{
		Symbol:        term,
		Resolution:    resolution,
		TargetNodes:   resolution.Targets,
		Dependencies:  []NodeResult{},
		Dependents:    []NodeResult{},
		Relationships: []EdgeResult{},
	}
	if len(targets) == 0 {
		return res
	}

	targetIDs := make(map[string]bool)
	type queueItem struct {
		node  *models.DependencyNode
		depth int
	}
	var depQueue []queueItem
	for _, target := range targets {
		targetIDs[target.ID] = true
		depQueue = append(depQueue, queueItem{node: target, depth: 0})
	}

	edgeIndex := make(map[string]int)
	recordEdge := func(sourceID, targetID string, ref *models.DependencyRef, depth int) {
		key := sourceID + "\x00" + targetID
		if index, exists := edgeIndex[key]; exists {
			if depth < res.Relationships[index].Depth {
				res.Relationships[index].Depth = depth
			}
			return
		}
		edgeIndex[key] = len(res.Relationships)
		res.Relationships = append(res.Relationships, newEdge(sourceID, targetID, ref, depth))
	}

	seenDeps := make(map[string]bool)
	for len(depQueue) > 0 {
		current := depQueue[0]
		depQueue = depQueue[1:]
		if current.depth >= maxDepth {
			continue
		}
		for _, ref := range current.node.Dependencies {
			depNode := e.graph.Nodes[ref.TargetID]
			if depNode == nil {
				continue
			}
			recordEdge(current.node.ID, depNode.ID, ref, current.depth+1)
			if targetIDs[depNode.ID] || seenDeps[depNode.ID] {
				continue
			}
			seenDeps[depNode.ID] = true
			r := nodeToResult(depNode)
			r.RefType = ref.Type
			r.RefLines = ref.Lines
			r.RefCount = ref.Count
			r.Depth = current.depth + 1
			res.Dependencies = append(res.Dependencies, r)
			depQueue = append(depQueue, queueItem{node: depNode, depth: current.depth + 1})
		}
	}

	seenCallers := make(map[string]bool)
	var callerQueue []queueItem
	for _, target := range targets {
		callerQueue = append(callerQueue, queueItem{node: target, depth: 0})
	}
	for len(callerQueue) > 0 {
		current := callerQueue[0]
		callerQueue = callerQueue[1:]
		if current.depth >= maxDepth {
			continue
		}
		for _, ref := range current.node.Dependents {
			callerNode := e.graph.Nodes[ref.TargetID]
			if callerNode == nil {
				continue
			}
			recordEdge(callerNode.ID, current.node.ID, ref, current.depth+1)
			if targetIDs[callerNode.ID] || seenCallers[callerNode.ID] {
				continue
			}
			seenCallers[callerNode.ID] = true
			r := nodeToResult(callerNode)
			r.RefType = ref.Type
			r.RefLines = ref.Lines
			r.RefCount = ref.Count
			r.Depth = current.depth + 1
			res.Dependents = append(res.Dependents, r)
			callerQueue = append(callerQueue, queueItem{node: callerNode, depth: current.depth + 1})
		}
	}

	sortResults(res.Dependencies)
	sortResults(res.Dependents)
	sortEdges(res.Relationships)
	return res
}
