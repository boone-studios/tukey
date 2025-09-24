// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/boone-studios/tukey/internal/models"
)

// DependencyTracker builds dependency relationships
type DependencyTracker struct {
	graph        *models.DependencyGraph
	nodeIndex    map[string]string     // Maps element names to node IDs
	namespaceMap map[string]string     // Maps class names to full-namespaced names
	allUsage     []models.UsageElement // Store all usage for function reporting
}

// NewDependencyTracker creates a new dependency tracker
func NewDependencyTracker() *DependencyTracker {
	return &DependencyTracker{
		graph: &models.DependencyGraph{
			Nodes:          make(map[string]*models.DependencyNode),
			Orphans:        []*models.DependencyNode{},
			HighlyDepended: []*models.DependencyNode{},
			ComplexNodes:   []*models.DependencyNode{},
		},
		nodeIndex:    make(map[string]string),
		namespaceMap: make(map[string]string),
		allUsage:     []models.UsageElement{},
	}
}

// BuildDependencyGraph creates the complete dependency graph from parsed files
func (dt *DependencyTracker) BuildDependencyGraph(parsedFiles []*models.ParsedFile) *models.DependencyGraph {
	// Phase 1: Create all nodes and build indexes
	dt.createNodes(parsedFiles)

	// Phase 2: Build dependency relationships
	dt.buildRelationships(parsedFiles)

	// Phase 3: Calculate metrics and analyze patterns
	dt.calculateMetrics()
	dt.identifyPatterns()

	return dt.graph
}

// createNodes builds all nodes and indexes from parsed files
func (dt *DependencyTracker) createNodes(parsedFiles []*models.ParsedFile) {
	dt.graph.Lock()
	defer dt.graph.Unlock()

	for _, file := range parsedFiles {
		// Build namespace mapping for this file
		for _, element := range file.Elements {
			fullName := dt.getFullName(element.Namespace, element.Name)

			// Create unique node ID
			nodeID := fmt.Sprintf("%s:%s:%d", element.Type, fullName, element.Line)

			node := &models.DependencyNode{
				ID:           nodeID,
				Name:         element.Name,
				Type:         element.Type,
				File:         file.Path,
				Namespace:    element.Namespace,
				ClassName:    element.ClassName,
				Line:         element.Line,
				Dependencies: make(map[string]*models.DependencyRef),
				Dependents:   make(map[string]*models.DependencyRef),
				Score:        dt.calculateComplexityScore(&element),
			}

			dt.graph.Nodes[nodeID] = node

			// Build search indexes - be more careful about conflicts
			// Always index by full name (with namespace)
			dt.nodeIndex[fullName] = nodeID

			// Only index by short name if there's no namespace conflict
			if element.Namespace == "" {
				// Global namespace - safe to index by short name
				dt.nodeIndex[element.Name] = nodeID
			} else {
				// Check if this short name already exists
				if _, exists := dt.nodeIndex[element.Name]; exists {
					// There's a conflict - remove the short name index
					// This forces resolution to use full namespaced names
					delete(dt.nodeIndex, element.Name)

					// Also remove it from the namespace map if it was a class
					if element.Type == "class" {
						delete(dt.namespaceMap, element.Name)
					}
				} else {
					// No conflict yet - add a short name index
					dt.nodeIndex[element.Name] = nodeID

					if element.Type == "class" {
						dt.namespaceMap[element.Name] = fullName
					}
				}
			}
		}
	}

	dt.graph.TotalNodes = len(dt.graph.Nodes)
}

// buildRelationships creates dependency links between nodes
func (dt *DependencyTracker) buildRelationships(parsedFiles []*models.ParsedFile) {
	for _, file := range parsedFiles {
		dt.processFileUsage(file)
		dt.processImports(file)
	}
}

// processFileUsage analyzes usage patterns in a file
func (dt *DependencyTracker) processFileUsage(file *models.ParsedFile) {
	for _, usage := range file.Usage {
		// Store usage for function reporting
		dt.allUsage = append(dt.allUsage, usage)
		dt.createDependency(usage, file)
	}
}

// processImports handles use statements and namespace imports
func (dt *DependencyTracker) processImports(file *models.ParsedFile) {
	for _, use := range file.Uses {
		// Find classes in current file that might use these imports
		for _, element := range file.Elements {
			if element.Type == "class" {
				dt.createImportDependency(element, use, file)
			}
		}
	}
}

// createDependency establishes a dependency relationship
func (dt *DependencyTracker) createDependency(usage models.UsageElement, file *models.ParsedFile) {
	// Find the source node (where the usage occurs)
	var sourceNode *models.DependencyNode
	for _, node := range dt.graph.Nodes {
		if node.File == file.Path {
			if usage.Context == node.Name ||
				(usage.Context == node.ClassName && node.Type == "class") {
				sourceNode = node
				break
			}
		}
	}

	if sourceNode == nil {
		return // Can't find source context
	}

	// Find target node
	targetNodeID := dt.findTargetNode(usage.Name, file.Namespace)
	if targetNodeID == "" {
		return // External dependency or not found
	}

	targetNode := dt.graph.Nodes[targetNodeID]
	if targetNode == nil {
		return
	}

	// Create or update dependency reference
	dt.addDependencyRef(sourceNode, targetNode, usage.Type, usage.Line)
}

// createImportDependency handles import-based dependencies
func (dt *DependencyTracker) createImportDependency(element models.CodeElement, importPath string, file *models.ParsedFile) {
	sourceNodeID := dt.nodeIndex[dt.getFullName(element.Namespace, element.Name)]
	if sourceNodeID == "" {
		return
	}

	sourceNode := dt.graph.Nodes[sourceNodeID]
	if sourceNode == nil {
		return
	}

	// Only create dependencies for imports that actually exist in our codebase
	// Try to find the exact import path first (full namespace match)
	targetNodeID := dt.nodeIndex[importPath]
	if targetNodeID != "" {
		targetNode := dt.graph.Nodes[targetNodeID]
		if targetNode != nil {
			dt.addDependencyRef(sourceNode, targetNode, "imports", element.Line)
		}
		return
	}
}

// addDependencyRef adds or updates a dependency reference
func (dt *DependencyTracker) addDependencyRef(source, target *models.DependencyNode, depType string, line int) {
	if source.ID == target.ID {
		return // No self-dependencies
	}

	dt.graph.Lock()
	defer dt.graph.Unlock()

	// Add to source's dependencies
	if dep, exists := source.Dependencies[target.ID]; exists {
		dep.Count++
		dep.Lines = append(dep.Lines, line)
	} else {
		source.Dependencies[target.ID] = &models.DependencyRef{
			TargetID:   target.ID,
			TargetName: target.Name,
			Type:       depType,
			Count:      1,
			Lines:      []int{line},
		}
	}

	// Add to target's dependents
	if dep, exists := target.Dependents[source.ID]; exists {
		dep.Count++
		dep.Lines = append(dep.Lines, line)
	} else {
		target.Dependents[source.ID] = &models.DependencyRef{
			TargetID:   source.ID,
			TargetName: source.Name,
			Type:       depType,
			Count:      1,
			Lines:      []int{line},
		}
	}

	dt.graph.TotalEdges++
}

// findTargetNode locates a target node by name and context
func (dt *DependencyTracker) findTargetNode(name, namespace string) string {
	// For static calls like "Response::create", extract just the class name
	if strings.Contains(name, "::") {
		parts := strings.Split(name, "::")
		className := parts[0]

		// Try the exact namespace match first
		fullName := dt.getFullName(namespace, className)
		if nodeID, exists := dt.nodeIndex[fullName]; exists {
			return nodeID
		}

		// Try to find in the namespace map (for classes in current namespace)
		if fullName, exists := dt.namespaceMap[className]; exists {
			if nodeID, exists := dt.nodeIndex[fullName]; exists {
				return nodeID
			}
		}

		// Only match by class name alone if it's unambiguous
		// (i.e., there's exactly one class with that name in our codebase)
		if nodeID, exists := dt.nodeIndex[className]; exists {
			// Verify this is actually the right class by checking if it's in our namespace
			if targetNode := dt.graph.Nodes[nodeID]; targetNode != nil {
				// Only return if it's in our codebase (not external)
				if targetNode.Namespace != "" || targetNode.File != "" {
					return nodeID
				}
			}
		}

		return ""
	}

	// For regular method calls, property access, etc.
	// Try the exact match first
	if nodeID, exists := dt.nodeIndex[name]; exists {
		return nodeID
	}

	// Try with the current namespace
	fullName := dt.getFullName(namespace, name)
	if nodeID, exists := dt.nodeIndex[fullName]; exists {
		return nodeID
	}

	// Try to resolve through the namespace map
	if fullName, exists := dt.namespaceMap[name]; exists {
		if nodeID, exists := dt.nodeIndex[fullName]; exists {
			return nodeID
		}
	}

	return ""
}

// calculateComplexityScore assigns a complexity score to an element
func (dt *DependencyTracker) calculateComplexityScore(element *models.CodeElement) int {
	score := 1 // Base score

	switch element.Type {
	case "class":
		score = 5
		if element.IsAbstract {
			score += 2
		}
	case "method", "function":
		score = 3
		score += len(element.Parameters) // More parameters = more complexity
		if element.IsStatic {
			score += 1
		}
		if element.IsAbstract {
			score += 2
		}
	case "property":
		score = 2
		if element.IsStatic {
			score += 1
		}
	}

	return score
}

// calculateMetrics computes various graph metrics
func (dt *DependencyTracker) calculateMetrics() {
	dt.graph.Lock()
	defer dt.graph.Unlock()

	for _, node := range dt.graph.Nodes {
		// Update node scores based on dependencies
		node.Score += len(node.Dependencies) + (len(node.Dependents) * 2)
	}
}

// identifyPatterns finds interesting patterns in the dependency graph
func (dt *DependencyTracker) identifyPatterns() {
	dt.graph.Lock()
	defer dt.graph.Unlock()

	// Find orphans (no dependencies or dependents)
	// Find highly depended nodes
	// Find complex nodes

	var allNodes []*models.DependencyNode
	for _, node := range dt.graph.Nodes {
		allNodes = append(allNodes, node)
	}

	// Sort by different criteria
	sort.Slice(allNodes, func(i, j int) bool {
		return len(allNodes[i].Dependents) > len(allNodes[j].Dependents)
	})

	// Top 10 most depended upon
	maxHighlyDepended := 10
	if len(allNodes) < maxHighlyDepended {
		maxHighlyDepended = len(allNodes)
	}
	dt.graph.HighlyDepended = allNodes[:maxHighlyDepended]

	// Find orphans
	for _, node := range allNodes {
		if len(node.Dependencies) == 0 && len(node.Dependents) == 0 {
			dt.graph.Orphans = append(dt.graph.Orphans, node)
		}
	}

	// Sort by complexity score for complex nodes
	sort.Slice(allNodes, func(i, j int) bool {
		return allNodes[i].Score > allNodes[j].Score
	})

	maxComplexNodes := 10
	if len(allNodes) < maxComplexNodes {
		maxComplexNodes = len(allNodes)
	}
	dt.graph.ComplexNodes = allNodes[:maxComplexNodes]
}

// Helper functions
func (dt *DependencyTracker) getFullName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "\\" + name
}

func (dt *DependencyTracker) extractClassNameFromImport(importPath string) string {
	parts := strings.Split(importPath, "\\")
	return parts[len(parts)-1]
}

// ExportToJSON exports the dependency graph to JSON
func (dt *DependencyTracker) ExportToJSON(filename string) error {
	data, err := json.MarshalIndent(dt.graph, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// PrintSummary displays a human-readable summary of the dependency graph
func (dt *DependencyTracker) PrintSummary(verbose bool) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("DEPENDENCY ANALYSIS SUMMARY")
	fmt.Println(strings.Repeat("=", 70))

	fmt.Printf("📊 Graph Statistics:\n")
	fmt.Printf("   • Total Nodes: %d\n", dt.graph.TotalNodes)
	fmt.Printf("   • Total Dependencies: %d\n", dt.graph.TotalEdges)
	fmt.Printf("   • Orphaned Elements: %d\n", len(dt.graph.Orphans))

	// Determine how many items to show
	maxHighlyDepended := 5
	maxComplexNodes := 5
	maxOrphans := 10
	maxDependentsToShow := 3

	if verbose {
		maxHighlyDepended = len(dt.graph.HighlyDepended)
		maxComplexNodes = len(dt.graph.ComplexNodes)
		maxOrphans = len(dt.graph.Orphans)
		maxDependentsToShow = -1 // Show all
		fmt.Printf("\n🔍 VERBOSE MODE: Showing complete dependency lists\n")
	}

	fmt.Printf("\n🔥 Most Depended Upon Elements:\n")
	for i, node := range dt.graph.HighlyDepended {
		if i >= maxHighlyDepended {
			if !verbose {
				fmt.Printf("   ... and %d more (use -v for full list)\n", len(dt.graph.HighlyDepended)-maxHighlyDepended)
			}
			break
		}

		relativePath := strings.TrimPrefix(node.File, "/")
		if strings.HasPrefix(relativePath, "/") {
			relativePath = relativePath[1:] // Remove leading slash if still present
		}

		fmt.Printf("   %d. %s (%s) - %d dependents\n",
			i+1, node.Name, relativePath, len(node.Dependents))

		// Show dependents
		dependentCount := 0
		for _, dep := range node.Dependents {
			if maxDependentsToShow > 0 && dependentCount >= maxDependentsToShow {
				fmt.Printf("      ... and %d more dependents\n", len(node.Dependents)-maxDependentsToShow)
				break
			}
			fmt.Printf("      ← %s (%s)\n", dep.TargetName, dep.Type)
			dependentCount++
		}

		if verbose && i < len(dt.graph.HighlyDepended)-1 {
			fmt.Println() // Add spacing between entries in verbose mode
		}
	}

	fmt.Printf("\n🧠 Most Complex Elements:\n")
	for i, node := range dt.graph.ComplexNodes {
		if i >= maxComplexNodes {
			if !verbose {
				fmt.Printf("   ... and %d more (use -v for full list)\n", len(dt.graph.ComplexNodes)-maxComplexNodes)
			}
			break
		}

		relativePath := strings.TrimPrefix(node.File, "/")
		if strings.HasPrefix(relativePath, "/") {
			relativePath = relativePath[1:]
		}

		fmt.Printf("   %d. %s (%s) - Score: %d\n",
			i+1, node.Name, relativePath, node.Score)
		fmt.Printf("      Dependencies: %d, Dependents: %d\n",
			len(node.Dependencies), len(node.Dependents))

		if verbose {
			// Show what this node depends on
			if len(node.Dependencies) > 0 {
				fmt.Printf("      Depends on:\n")
				for _, dep := range node.Dependencies {
					fmt.Printf("        → %s (%s, %d times)\n", dep.TargetName, dep.Type, dep.Count)
				}
			}

			// Show what depends on this node
			if len(node.Dependents) > 0 {
				fmt.Printf("      Depended upon by:\n")
				depCount := 0
				for _, dep := range node.Dependents {
					if depCount >= 10 { // Limit even in verbose mode for readability
						fmt.Printf("        ... and %d more\n", len(node.Dependents)-10)
						break
					}
					fmt.Printf("        ← %s (%s, %d times)\n", dep.TargetName, dep.Type, dep.Count)
					depCount++
				}
			}

			if i < len(dt.graph.ComplexNodes)-1 {
				fmt.Println() // Add spacing between entries
			}
		}
	}

	if len(dt.graph.Orphans) > 0 {
		fmt.Printf("\n👻 Orphaned Elements (%d total):\n", len(dt.graph.Orphans))
		for i, node := range dt.graph.Orphans {
			if i >= maxOrphans {
				if !verbose {
					fmt.Printf("   ... and %d more (use -v for full list)\n", len(dt.graph.Orphans)-maxOrphans)
				}
				break
			}

			relativePath := strings.TrimPrefix(node.File, "/")
			if strings.HasPrefix(relativePath, "/") {
				relativePath = relativePath[1:]
			}

			if verbose {
				fmt.Printf("   • %s (%s) in %s (line %d)\n", node.Name, node.Type, relativePath, node.Line)
			} else {
				fmt.Printf("   • %s (%s) in %s\n", node.Name, node.Type, relativePath)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 70))

	// Add function usage report
	if verbose {
		dt.PrintFunctionUsageReport()
	}

	if !verbose {
		fmt.Printf("💡 Tip: Use -v or --verbose flag to see complete dependency lists and function usage report\n")
		fmt.Println(strings.Repeat("=", 70))
	}
}

// PrintFunctionUsageReport shows detailed function usage across the codebase
func (dt *DependencyTracker) PrintFunctionUsageReport() {
	fmt.Printf("\n📋 FUNCTION USAGE REPORT\n")
	fmt.Println(strings.Repeat("=", 70))

	// Group function calls by function name
	functionCalls := make(map[string][]models.UsageElement)
	functionDefinitions := make(map[string]*models.DependencyNode)

	// Find all function definitions in our codebase
	for _, node := range dt.graph.Nodes {
		if node.Type == "function" {
			functionDefinitions[node.Name] = node
		}
	}

	// Group all function calls
	for _, usage := range dt.allUsage {
		if usage.Type == "function_call" {
			if calls, exists := functionCalls[usage.Name]; exists {
				functionCalls[usage.Name] = append(calls, usage)
			} else {
				functionCalls[usage.Name] = []models.UsageElement{usage}
			}
		}
	}

	if len(functionCalls) == 0 {
		fmt.Printf("   No custom function calls detected.\n")
		fmt.Printf("   (Built-in PHP and common Laravel functions are filtered out)\n")
		return
	}

	// Sort functions by number of calls
	type FunctionSummary struct {
		Name       string
		Definition *models.DependencyNode
		Calls      []models.UsageElement
		TotalCalls int
	}

	var functionSummaries []FunctionSummary
	for funcName, calls := range functionCalls {
		functionSummaries = append(functionSummaries, FunctionSummary{
			Name:       funcName,
			Definition: functionDefinitions[funcName],
			Calls:      calls,
			TotalCalls: len(calls),
		})
	}

	// Sort by total calls (descending)
	sort.Slice(functionSummaries, func(i, j int) bool {
		return functionSummaries[i].TotalCalls > functionSummaries[j].TotalCalls
	})

	// Display the report
	for _, summary := range functionSummaries {
		if summary.Definition != nil {
			// Function is defined in our codebase
			relativePath := strings.TrimPrefix(summary.Definition.File, "/")
			if strings.HasPrefix(relativePath, "/") {
				relativePath = relativePath[1:]
			}
			fmt.Printf("\n📁 %s\n", relativePath)
			fmt.Printf("  📋 function %s() (line %d) - %d calls\n",
				summary.Name, summary.Definition.Line, summary.TotalCalls)
		} else {
			// External or helper function
			fmt.Printf("\n🔧 function %s() - %d calls (external/helper)\n",
				summary.Name, summary.TotalCalls)
		}

		fmt.Printf("  🔗 Called from %d locations:\n", len(summary.Calls))

		// Group calls by file for better organization
		callsByFile := make(map[string][]models.UsageElement)
		for _, call := range summary.Calls {
			// We need to find which file this call came from
			// Let's find the node that made this call
			var callerFile string

			for _, node := range dt.graph.Nodes {
				if call.Context == node.Name ||
					(node.Type == "class" && call.Context == node.Name) {
					callerFile = node.File
					break
				}
			}

			if callerFile == "" {
				callerFile = "unknown"
			}

			if calls, exists := callsByFile[callerFile]; exists {
				callsByFile[callerFile] = append(calls, call)
			} else {
				callsByFile[callerFile] = []models.UsageElement{call}
			}
		}

		for filePath, calls := range callsByFile {
			relativePath := strings.TrimPrefix(filePath, "/")
			if strings.HasPrefix(relativePath, "/") {
				relativePath = relativePath[1:]
			}

			if relativePath == "unknown" {
				fmt.Printf("    📂 Unknown context:\n")
			} else {
				fmt.Printf("    📂 %s:\n", relativePath)
			}

			for _, call := range calls {
				contextStr := ""
				if call.Context != "" {
					contextStr = fmt.Sprintf(" in %s()", call.Context)
				}

				fmt.Printf("      → line %d%s\n", call.Line, contextStr)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 70))
}

// Helper function to get usage elements from stored usage data
func (dt *DependencyTracker) getNodeUsage(node *models.DependencyNode) []models.UsageElement {
	var usages []models.UsageElement

	// Find all usage elements that originate from this node's file
	for _, usage := range dt.allUsage {
		// Check if this usage comes from the same file and context as the node
		if usage.Context == node.Name ||
			(node.Type == "class" && usage.Context == node.Name) {
			// Additional check: we need to match the file somehow
			// Since we don't store file in UsageElement, we'll work with what we have
			usages = append(usages, usage)
		}
	}

	return usages
}
