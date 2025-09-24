// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

//go:build !testcover

package models

import "sync"

// FileInfo holds information about discovered PHP files
type FileInfo struct {
	Path         string
	RelativePath string
	Size         int64
}

// CodeElement represents any parseable element in PHP code
type CodeElement struct {
	Type       string   // "class", "function", "method", "property", "constant"
	Name       string   // Element name
	Namespace  string   // Namespace (if any)
	ClassName  string   // Parent class (for methods/properties)
	Visibility string   // "public", "private", "protected"
	IsStatic   bool     // For methods and properties
	IsAbstract bool     // For classes and methods
	Line       int      // Line number where defined
	File       string   // File path
	Parameters []string // For functions/methods
	ReturnType string   // Return type hint (if any)
}

// ParsedFile contains all elements found in a PHP file
type ParsedFile struct {
	Path      string
	Namespace string
	Uses      []string       // Import statements
	Elements  []CodeElement  // All defined elements
	Usage     []UsageElement // References to other elements
}

// UsageElement represents usage of external code elements
type UsageElement struct {
	Type     string // "class", "function", "method", "property"
	Name     string
	Context  string // Where it's used (function name, class name, etc.)
	Line     int
	IsStatic bool
}

// DependencyNode represents a node in the dependency tree
type DependencyNode struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Type         string                    `json:"type"`
	File         string                    `json:"file"`
	Namespace    string                    `json:"namespace"`
	ClassName    string                    `json:"className,omitempty"`
	Line         int                       `json:"line"`
	Dependencies map[string]*DependencyRef `json:"dependencies"`
	Dependents   map[string]*DependencyRef `json:"dependents"`
	Score        int                       `json:"score"`
}

// DependencyRef represents a reference between nodes
type DependencyRef struct {
	TargetID   string `json:"targetId"`
	TargetName string `json:"targetName"`
	Type       string `json:"type"` // "uses", "extends", "implements", "calls", "instantiates"
	Count      int    `json:"count"`
	Lines      []int  `json:"lines"`
	Context    string `json:"context"`
}

// DependencyGraph holds the complete dependency analysis
type DependencyGraph struct {
	Nodes          map[string]*DependencyNode `json:"nodes"`
	TotalNodes     int                        `json:"totalNodes"`
	TotalEdges     int                        `json:"totalEdges"`
	Orphans        []*DependencyNode          `json:"orphans"`
	HighlyDepended []*DependencyNode          `json:"highlyDepended"`
	ComplexNodes   []*DependencyNode          `json:"complexNodes"`
	mu             sync.RWMutex
}

// AnalysisResult holds the complete analysis results
type AnalysisResult struct {
	Graph          *DependencyGraph
	ParsedFiles    []*ParsedFile
	TotalFiles     int
	TotalElements  int
	ProcessingTime string
}

// Lock Concurrency helpers (exported so other packages can coordinate safely)
func (g *DependencyGraph) Lock()    { g.mu.Lock() }
func (g *DependencyGraph) Unlock()  { g.mu.Unlock() }
func (g *DependencyGraph) RLock()   { g.mu.RLock() }
func (g *DependencyGraph) RUnlock() { g.mu.RUnlock() }
