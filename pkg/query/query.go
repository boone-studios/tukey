// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package query

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/boone-studios/tukey/internal/models"
)

// NodeResult is a single entry returned by all query types.
type NodeResult struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	File            string `json:"file"`
	Namespace       string `json:"namespace,omitempty"`
	ClassName       string `json:"className,omitempty"`
	Line            int    `json:"line"`
	Score           int    `json:"score"`
	DependencyCount int    `json:"dependencyCount"`
	DependentCount  int    `json:"dependentCount"`
	// Populated for --callers and --dependents to show how nodes are connected.
	RefType  string `json:"refType,omitempty"`
	RefLines []int  `json:"refLines,omitempty"`
	RefCount int    `json:"refCount,omitempty"`
}

// QueryResult is the JSON envelope returned by every query operation.
type QueryResult struct {
	Query   string       `json:"query"`
	Term    string       `json:"term,omitempty"`
	Count   int          `json:"count"`
	Results []NodeResult `json:"results"`
}

// Engine queries a pre-built Tukey analysis file without re-running analysis.
type Engine struct {
	graph *models.DependencyGraph
}

type analysisEnvelope struct {
	Graph *models.DependencyGraph `json:"graph"`
}

// Load reads a Tukey analysis JSON file (e.g. produced by tukey -o analysis.json)
// and returns a ready Engine.
func Load(path string) (*Engine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var env analysisEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", path, err)
	}
	if env.Graph == nil {
		return nil, fmt.Errorf("%s: missing 'graph' field – was this file produced by tukey -o?", path)
	}
	return &Engine{graph: env.Graph}, nil
}

// Find returns all nodes whose name contains term (case-insensitive substring).
func (e *Engine) Find(term string) QueryResult {
	// Try direct ID lookup first
	if node, exists := e.graph.Nodes[term]; exists {
		results := []NodeResult{nodeToResult(node)}
		return QueryResult{Query: "find", Term: term, Count: 1, Results: results}
	}

	lower := strings.ToLower(term)
	var results []NodeResult
	for _, node := range e.graph.Nodes {
		matched := false
		for _, candidateName := range e.getNodeNames(node) {
			if strings.Contains(strings.ToLower(candidateName), lower) {
				matched = true
				break
			}
		}
		if matched {
			results = append(results, nodeToResult(node))
		}
	}
	sortResults(results)
	return QueryResult{Query: "find", Term: term, Count: len(results), Results: coalesce(results)}
}

// Callers returns all nodes that call or reference the named symbol (incoming edges).
// When multiple nodes share the same name, results are merged across all matches.
func (e *Engine) Callers(term string) QueryResult {
	targets := e.findNodes(term)
	seen := make(map[string]bool)
	var results []NodeResult
	for _, target := range targets {
		for _, ref := range target.Dependents {
			if seen[ref.TargetID] {
				continue
			}
			seen[ref.TargetID] = true
			caller := e.graph.Nodes[ref.TargetID]
			if caller == nil {
				continue
			}
			r := nodeToResult(caller)
			r.RefType = ref.Type
			r.RefLines = ref.Lines
			r.RefCount = ref.Count
			results = append(results, r)
		}
	}
	sortResults(results)
	return QueryResult{Query: "callers", Term: term, Count: len(results), Results: coalesce(results)}
}

// Dependents returns all nodes that the named symbol directly depends on (outgoing edges).
// When multiple nodes share the same name, results are merged across all matches.
func (e *Engine) Dependents(term string) QueryResult {
	targets := e.findNodes(term)
	seen := make(map[string]bool)
	var results []NodeResult
	for _, target := range targets {
		for _, ref := range target.Dependencies {
			if seen[ref.TargetID] {
				continue
			}
			seen[ref.TargetID] = true
			dep := e.graph.Nodes[ref.TargetID]
			if dep == nil {
				continue
			}
			r := nodeToResult(dep)
			r.RefType = ref.Type
			r.RefLines = ref.Lines
			r.RefCount = ref.Count
			results = append(results, r)
		}
	}
	sortResults(results)
	return QueryResult{Query: "dependents", Term: term, Count: len(results), Results: coalesce(results)}
}

// Orphans returns all nodes with no dependencies and no dependents (dead code candidates).
func (e *Engine) Orphans() QueryResult {
	var results []NodeResult
	for _, node := range e.graph.Orphans {
		results = append(results, nodeToResult(node))
	}
	sortResults(results)
	return QueryResult{Query: "orphans", Count: len(results), Results: coalesce(results)}
}

// getNodeNames returns all possible candidate names for a node to match against.
func (e *Engine) getNodeNames(node *models.DependencyNode) []string {
	names := []string{node.Name}

	if node.Namespace != "" {
		names = append(names, node.Namespace+"\\"+node.Name)
	}

	if node.ClassName != "" {
		classScoped := node.ClassName + "::" + node.Name
		names = append(names, classScoped)
		if node.Namespace != "" {
			names = append(names, node.Namespace+"\\"+classScoped)
		}
	}

	names = append(names, node.ID)

	return names
}

// findNodes returns all nodes whose name exactly matches term (case-insensitive),
// falling back to a substring match when there are no exact matches.
func (e *Engine) findNodes(term string) []*models.DependencyNode {
	// Try direct ID lookup first (exact match)
	if node, exists := e.graph.Nodes[term]; exists {
		return []*models.DependencyNode{node}
	}

	lower := strings.ToLower(term)
	var exact, partial []*models.DependencyNode

	for _, node := range e.graph.Nodes {
		isExact := false
		isPartial := false

		for _, candidateName := range e.getNodeNames(node) {
			candLower := strings.ToLower(candidateName)
			if candLower == lower {
				isExact = true
				break
			} else if strings.Contains(candLower, lower) {
				isPartial = true
			}
		}

		if isExact {
			exact = append(exact, node)
		} else if isPartial {
			partial = append(partial, node)
		}
	}

	if len(exact) > 0 {
		return exact
	}
	return partial
}

func nodeToResult(node *models.DependencyNode) NodeResult {
	return NodeResult{
		ID:              node.ID,
		Name:            node.Name,
		Type:            node.Type,
		File:            node.File,
		Namespace:       node.Namespace,
		ClassName:       node.ClassName,
		Line:            node.Line,
		Score:           node.Score,
		DependencyCount: len(node.Dependencies),
		DependentCount:  len(node.Dependents),
	}
}

func sortResults(results []NodeResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].File != results[j].File {
			return results[i].File < results[j].File
		}
		return results[i].Line < results[j].Line
	})
}

// coalesce ensures callers never get a JSON null instead of an empty array.
func coalesce(results []NodeResult) []NodeResult {
	if results == nil {
		return []NodeResult{}
	}
	return results
}
