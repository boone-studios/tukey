// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/boone-studios/tukey/internal/models"
)

// DependencyTracker builds dependency relationships
type DependencyTracker struct {
	graph          *models.DependencyGraph
	nodeIndex      map[string][]string
	shortNameIndex map[string][]string
	sourceIndex    map[string]map[string][]string
	allUsage       []models.UsageElement // Store all usage for function reporting
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
		nodeIndex:      make(map[string][]string),
		shortNameIndex: make(map[string][]string),
		sourceIndex:    make(map[string]map[string][]string),
		allUsage:       []models.UsageElement{},
	}
}

// BuildDependencyGraph creates the complete dependency graph from parsed files
func (dt *DependencyTracker) BuildDependencyGraph(parsedFiles []*models.ParsedFile) *models.DependencyGraph {
	parsedFiles = append([]*models.ParsedFile(nil), parsedFiles...)
	sort.Slice(parsedFiles, func(i, j int) bool {
		return parsedFiles[i].Path < parsedFiles[j].Path
	})

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
		if dt.sourceIndex[file.Path] == nil {
			dt.sourceIndex[file.Path] = make(map[string][]string)
		}

		for _, element := range file.Elements {
			baseID := models.StableSymbolID(element.Type, element.Namespace, element.ClassName, element.Name)
			nodeID := baseID
			for suffix := 2; dt.graph.Nodes[nodeID] != nil; suffix++ {
				nodeID = fmt.Sprintf("%s#%d", baseID, suffix)
			}

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
			dt.addIndex(dt.nodeIndex, symbolKey(element.Namespace, element.ClassName, element.Name), nodeID)
			dt.addIndex(dt.shortNameIndex, element.Name, nodeID)
			dt.addIndex(dt.sourceIndex[file.Path], element.Name, nodeID)
			if element.ClassName != "" {
				dt.addIndex(dt.sourceIndex[file.Path], element.ClassName+"::"+element.Name, nodeID)
			}
		}
	}

	dt.graph.TotalNodes = len(dt.graph.Nodes)
}

func (dt *DependencyTracker) addIndex(index map[string][]string, key, nodeID string) {
	for _, existingID := range index[key] {
		if existingID == nodeID {
			return
		}
	}
	index[key] = append(index[key], nodeID)
}

func symbolKey(namespace, className, name string) string {
	if className != "" {
		name = className + "::" + name
	}
	if namespace != "" {
		return namespace + "\\" + name
	}
	return name
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
				dt.createImportDependency(element, use)
			}
		}
	}
}

// createDependency establishes a dependency relationship
func (dt *DependencyTracker) createDependency(usage models.UsageElement, file *models.ParsedFile) {
	if usage.Context == "" {
		return
	}

	sourceNode := dt.findSourceNode(file.Path, usage.Context)
	if sourceNode == nil {
		return
	}

	targetNodeID := dt.findTargetNode(usage.Name, usage.Type, file, sourceNode)
	if targetNodeID == "" {
		return // External, unresolved, or ambiguous dependency
	}

	targetNode := dt.graph.Nodes[targetNodeID]
	if targetNode == nil {
		return
	}

	dt.addDependencyRef(sourceNode, targetNode, usage.Type, usage.Line)
}

func (dt *DependencyTracker) findSourceNode(filePath, context string) *models.DependencyNode {
	candidates := dt.sourceIndex[filePath][context]
	if len(candidates) > 1 {
		var members []string
		for _, nodeID := range candidates {
			node := dt.graph.Nodes[nodeID]
			if node != nil && node.ClassName == "" && node.Type != "class" && node.Type != "interface" && node.Type != "trait" && node.Type != "enum" {
				members = append(members, nodeID)
			}
		}
		if len(members) == 1 {
			candidates = members
		}
	}
	if len(candidates) != 1 {
		return nil
	}
	return dt.graph.Nodes[candidates[0]]
}

// createImportDependency handles import-based dependencies
func (dt *DependencyTracker) createImportDependency(element models.CodeElement, importPath string) {
	sourceNodeID := dt.uniqueNode([]string{symbolKey(element.Namespace, "", element.Name)}, "imports")
	if sourceNodeID == "" {
		return
	}

	targetNodeID := dt.uniqueNode([]string{importPath}, "imports")
	if targetNodeID == "" {
		return
	}

	dt.addDependencyRef(dt.graph.Nodes[sourceNodeID], dt.graph.Nodes[targetNodeID], "imports", element.Line)
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

// findTargetNode locates an unambiguous target using qualified names, imports, and scope.
func (dt *DependencyTracker) findTargetNode(name, usageType string, file *models.ParsedFile, sourceNode *models.DependencyNode) string {
	if qualifiedClass, memberName, ok := strings.Cut(name, "::"); ok {
		if strings.HasPrefix(memberName, "$") {
			usageType = "property"
		}
		memberName = strings.TrimPrefix(memberName, "$")
		fullyQualified := strings.HasPrefix(qualifiedClass, "\\")
		qualifiedClass = strings.TrimPrefix(qualifiedClass, "\\")
		if qualifiedClass == "self" || qualifiedClass == "static" {
			qualifiedClass = sourceNode.ClassName
			if qualifiedClass == "" {
				qualifiedClass = sourceNode.Name
			}
		}

		if fullyQualified || file.Namespace == "" {
			return dt.uniqueNode([]string{symbolKey("", qualifiedClass, memberName)}, usageType)
		}
		if imported := resolveImport(file, qualifiedClass); imported != "" {
			return dt.uniqueNode([]string{symbolKey("", imported, memberName)}, usageType)
		}
		if imported := resolveQualifiedImport(file, qualifiedClass); imported != "" {
			return dt.uniqueNode([]string{symbolKey("", imported, memberName)}, usageType)
		}
		if strings.Contains(qualifiedClass, "\\") {
			return dt.uniqueNode([]string{
				symbolKey("", qualifiedClass, memberName),
				symbolKey(file.Namespace, qualifiedClass, memberName),
			}, usageType)
		}
		return dt.firstUniqueNode(usageType,
			[]string{symbolKey(file.Namespace, qualifiedClass, memberName)},
			[]string{symbolKey("", qualifiedClass, memberName)},
		)
	}

	fullyQualified := strings.HasPrefix(name, "\\")
	name = strings.TrimPrefix(name, "\\")
	if fullyQualified {
		return dt.uniqueNode([]string{name}, usageType)
	}
	if imported := resolveImport(file, name); imported != "" {
		return dt.uniqueNode([]string{imported}, usageType)
	}
	if imported := resolveQualifiedImport(file, name); imported != "" {
		return dt.uniqueNode([]string{imported}, usageType)
	}
	if strings.Contains(name, "\\") {
		if nodeID := dt.firstUniqueNode(usageType,
			[]string{symbolKey(file.Namespace, "", name)},
			[]string{name},
		); nodeID != "" {
			return nodeID
		}
	} else if nodeID := dt.firstUniqueNode(usageType,
		[]string{symbolKey(file.Namespace, "", name)},
		[]string{name},
	); nodeID != "" {
		return nodeID
	}
	return dt.uniqueCandidates(dt.shortNameIndex[name], usageType)
}

func (dt *DependencyTracker) firstUniqueNode(usageType string, keyGroups ...[]string) string {
	for _, keys := range keyGroups {
		candidates := dt.matchingCandidates(keys, usageType)
		if len(candidates) > 1 {
			return ""
		}
		if len(candidates) == 1 {
			return candidates[0]
		}
	}
	return ""
}

func (dt *DependencyTracker) uniqueNode(keys []string, usageType string) string {
	return dt.uniqueCandidates(dt.matchingCandidates(keys, usageType), usageType)
}

func (dt *DependencyTracker) matchingCandidates(keys []string, usageType string) []string {
	var candidates []string
	seen := make(map[string]bool)
	for _, key := range keys {
		for _, nodeID := range dt.nodeIndex[key] {
			node := dt.graph.Nodes[nodeID]
			if node != nil && matchesUsageType(node.Type, usageType) && !seen[nodeID] {
				seen[nodeID] = true
				candidates = append(candidates, nodeID)
			}
		}
	}
	return candidates
}

func (dt *DependencyTracker) uniqueCandidates(candidates []string, usageType string) string {
	var match string
	for _, nodeID := range candidates {
		node := dt.graph.Nodes[nodeID]
		if node == nil || !matchesUsageType(node.Type, usageType) {
			continue
		}
		if match != "" {
			return ""
		}
		match = nodeID
	}
	return match
}

func matchesUsageType(nodeType, usageType string) bool {
	switch usageType {
	case "instantiation", "extends", "implements", "uses_trait", "class", "imports":
		return nodeType == "class" || nodeType == "interface" || nodeType == "trait" || nodeType == "enum"
	case "function_call", "function":
		return nodeType == "function"
	case "method_call":
		return nodeType == "method" || nodeType == "property"
	case "static_call":
		return nodeType == "method" || nodeType == "constant"
	case "property":
		return nodeType == "property"
	case "constant":
		return nodeType == "constant"
	default:
		return true
	}
}

func resolveImport(file *models.ParsedFile, name string) string {
	if imported := file.ImportAliases[name]; imported != "" {
		return imported
	}
	for _, importPath := range file.Uses {
		parts := strings.Split(strings.Trim(importPath, "\\"), "\\")
		if parts[len(parts)-1] == name {
			return importPath
		}
	}
	return ""
}

func resolveQualifiedImport(file *models.ParsedFile, name string) string {
	qualifier, remainder, ok := strings.Cut(name, "\\")
	if !ok {
		return ""
	}
	if imported := resolveImport(file, qualifier); imported != "" {
		return strings.TrimSuffix(imported, "\\") + "\\" + remainder
	}
	return ""
}

// calculateComplexityScore assigns a complexity score to an element
func (dt *DependencyTracker) calculateComplexityScore(element *models.CodeElement) int {
	score := 1 // Base score

	switch element.Type {
	case "class", "interface", "trait", "enum":
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
		if len(allNodes[i].Dependents) != len(allNodes[j].Dependents) {
			return len(allNodes[i].Dependents) > len(allNodes[j].Dependents)
		}
		return allNodes[i].ID < allNodes[j].ID
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
		if allNodes[i].Score != allNodes[j].Score {
			return allNodes[i].Score > allNodes[j].Score
		}
		return allNodes[i].ID < allNodes[j].ID
	})

	maxComplexNodes := 10
	if len(allNodes) < maxComplexNodes {
		maxComplexNodes = len(allNodes)
	}
	dt.graph.ComplexNodes = allNodes[:maxComplexNodes]
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
