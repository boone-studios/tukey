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

// NodeResult is a single entry returned by symbol query types.
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
	Depth           int    `json:"depth,omitempty"`
	// Populated for --callers and --dependents to show how nodes are connected
	RefType  string `json:"refType,omitempty"`
	RefLines []int  `json:"refLines,omitempty"`
	RefCount int    `json:"refCount,omitempty"`
}

// QueryResult is the JSON envelope returned by symbol query operations.
type QueryResult struct {
	Query         string            `json:"query"`
	Term          string            `json:"term,omitempty"`
	Count         int               `json:"count"`
	Results       []NodeResult      `json:"results"`
	Resolution    *SymbolResolution `json:"resolution,omitempty"`
	Relationships []EdgeResult      `json:"relationships,omitempty"`
}

// DiagnosticsResult is the JSON envelope returned for analysis diagnostics.
type DiagnosticsResult struct {
	Query   string                      `json:"query"`
	Count   int                         `json:"count"`
	Results []models.AnalysisDiagnostic `json:"results"`
}

// Engine queries a pre-built Tukey analysis file without re-running analysis
type Engine struct {
	graph    *models.DependencyGraph
	metadata models.AnalysisMetadata
}

type analysisEnvelope struct {
	models.AnalysisMetadata
	Graph *models.DependencyGraph `json:"graph"`
}

// Load reads a Tukey analysis JSON file (e.g. produced by tukey -o analysis.json)
// and returns a ready Engine
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
	return &Engine{graph: env.Graph, metadata: env.AnalysisMetadata}, nil
}

// Graph returns the underlying dependency graph
func (e *Engine) Graph() *models.DependencyGraph {
	return e.graph
}

// Find returns all nodes whose name contains term (case-insensitive substring)
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

// Orphans returns all nodes with no dependencies and no dependents (dead code candidates)
func (e *Engine) Orphans() QueryResult {
	var results []NodeResult
	for _, node := range e.graph.Orphans {
		results = append(results, nodeToResult(node))
	}
	sortResults(results)
	return QueryResult{Query: "orphans", Count: len(results), Results: coalesce(results)}
}

// Diagnostics returns parser and resolution diagnostics for the loaded analysis.
func (e *Engine) Diagnostics() DiagnosticsResult {
	results := append([]models.AnalysisDiagnostic(nil), e.graph.Diagnostics...)
	sort.Slice(results, func(i, j int) bool {
		if results[i].File != results[j].File {
			return results[i].File < results[j].File
		}
		if results[i].Code != results[j].Code {
			return results[i].Code < results[j].Code
		}
		if results[i].Symbol != results[j].Symbol {
			return results[i].Symbol < results[j].Symbol
		}
		return results[i].ReferenceType < results[j].ReferenceType
	})
	if results == nil {
		results = []models.AnalysisDiagnostic{}
	}
	return DiagnosticsResult{Query: "diagnostics", Count: len(results), Results: results}
}

// getNodeNames returns all possible candidate names for a node to match against
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
	nodes, _ := e.findNodesWithResolution(term)
	return nodes
}

func (e *Engine) findNodesWithResolution(term string) ([]*models.DependencyNode, string) {
	if node, exists := e.graph.Nodes[term]; exists {
		return []*models.DependencyNode{node}, "id"
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
		sortNodes(exact)
		return exact, "exact"
	}
	if len(partial) == 0 {
		return []*models.DependencyNode{}, "not_found"
	}
	sortNodes(partial)
	return partial, "substring"
}

func sortNodes(nodes []*models.DependencyNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
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
		if results[i].Line != results[j].Line {
			return results[i].Line < results[j].Line
		}
		return results[i].ID < results[j].ID
	})
}

// coalesce ensures callers never get a JSON null instead of an empty array
func coalesce(results []NodeResult) []NodeResult {
	if results == nil {
		return []NodeResult{}
	}
	return results
}
