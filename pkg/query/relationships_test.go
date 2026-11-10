package query

import (
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func relationshipTestNode(id, name string) *models.DependencyNode {
	return &models.DependencyNode{
		ID: id, Name: name, File: id + ".php",
		Dependencies: make(map[string]*models.DependencyRef),
		Dependents:   make(map[string]*models.DependencyRef),
	}
}

func relationshipTestLink(source, target *models.DependencyNode) {
	source.Dependencies[target.ID] = &models.DependencyRef{
		TargetID: target.ID, TargetName: target.Name, Type: "calls", Count: 1, Lines: []int{10},
	}
	target.Dependents[source.ID] = &models.DependencyRef{
		TargetID: source.ID, TargetName: source.Name, Type: "calls", Count: 1, Lines: []int{10},
	}
}

func buildAmbiguousRelationshipEngine() (*Engine, map[string]*models.DependencyNode) {
	nodes := map[string]*models.DependencyNode{
		"invoiceSave": relationshipTestNode("method:App\\Invoice::save", "save"),
		"orderSave":   relationshipTestNode("method:App\\Order::save", "save"),
		"invoiceCall": relationshipTestNode("function:App\\processInvoice", "processInvoice"),
		"orderCall":   relationshipTestNode("function:App\\processOrder", "processOrder"),
		"logger":      relationshipTestNode("class:App\\Logger", "Logger"),
		"audit":       relationshipTestNode("class:App\\Audit", "Audit"),
		"shared":      relationshipTestNode("class:App\\Shared", "Shared"),
	}
	relationshipTestLink(nodes["invoiceCall"], nodes["invoiceSave"])
	relationshipTestLink(nodes["orderCall"], nodes["orderSave"])
	relationshipTestLink(nodes["invoiceSave"], nodes["logger"])
	relationshipTestLink(nodes["invoiceSave"], nodes["shared"])
	relationshipTestLink(nodes["orderSave"], nodes["audit"])
	relationshipTestLink(nodes["orderSave"], nodes["shared"])
	graph := &models.DependencyGraph{Nodes: make(map[string]*models.DependencyNode)}
	for _, node := range nodes {
		graph.Nodes[node.ID] = node
	}
	return &Engine{graph: graph}, nodes
}

func TestCallersPreserveAmbiguousTargetsAndEdges(t *testing.T) {
	e, nodes := buildAmbiguousRelationshipEngine()
	result := e.Callers("save")
	if result.Resolution == nil || !result.Resolution.Ambiguous || result.Resolution.MatchType != "exact" {
		t.Fatalf("expected ambiguous exact resolution, got %+v", result.Resolution)
	}
	if len(result.Resolution.Targets) != 2 || result.Count != 2 || len(result.Relationships) != 2 {
		t.Fatalf("expected two targets, callers, and relationships, got %+v", result)
	}
	seen := make(map[string]string)
	for _, edge := range result.Relationships {
		seen[edge.TargetID] = edge.SourceID
	}
	if seen[nodes["invoiceSave"].ID] != nodes["invoiceCall"].ID || seen[nodes["orderSave"].ID] != nodes["orderCall"].ID {
		t.Fatalf("relationships lost target identity: %+v", result.Relationships)
	}
}

func TestCallersQualifiedNameNarrowsResolution(t *testing.T) {
	e, nodes := buildAmbiguousRelationshipEngine()
	result := e.Callers("Invoice::save")
	if result.Resolution == nil || result.Resolution.Ambiguous || len(result.Resolution.Targets) != 1 {
		t.Fatalf("expected one qualified target, got %+v", result.Resolution)
	}
	if len(result.Relationships) != 1 || result.Relationships[0].TargetID != nodes["invoiceSave"].ID {
		t.Fatalf("qualified query returned wrong edge: %+v", result.Relationships)
	}
}

func TestDependentsPreserveSharedNodeSources(t *testing.T) {
	e, nodes := buildAmbiguousRelationshipEngine()
	result := e.Dependents("save")
	if result.Count != 3 || len(result.Relationships) != 4 {
		t.Fatalf("expected three unique nodes and four edges, got %+v", result)
	}
	sharedSources := make(map[string]bool)
	for _, edge := range result.Relationships {
		if edge.TargetID == nodes["shared"].ID {
			sharedSources[edge.SourceID] = true
		}
	}
	if !sharedSources[nodes["invoiceSave"].ID] || !sharedSources[nodes["orderSave"].ID] {
		t.Fatalf("shared dependency lost one source: %+v", result.Relationships)
	}
}

func TestRelationshipResolutionStates(t *testing.T) {
	e, _ := buildAmbiguousRelationshipEngine()
	partial := e.Callers("sav")
	if partial.Resolution.MatchType != "substring" || !partial.Resolution.Ambiguous {
		t.Fatalf("unexpected substring resolution: %+v", partial.Resolution)
	}
	missing := e.Callers("missing")
	if missing.Resolution.MatchType != "not_found" || missing.Resolution.Targets == nil {
		t.Fatalf("unexpected missing resolution: %+v", missing.Resolution)
	}
}

func TestLocalizedContextRelationshipsAndDepth(t *testing.T) {
	e := buildTestEngine()
	result := e.LocalizedContext("GatewayFactory", 1)
	if result.Resolution.MatchType != "exact" || result.Resolution.Ambiguous {
		t.Fatalf("unexpected localized resolution: %+v", result.Resolution)
	}
	if len(result.Relationships) != 2 {
		t.Fatalf("expected two localized relationships, got %+v", result.Relationships)
	}
	for _, edge := range result.Relationships {
		if edge.Depth != 1 {
			t.Fatalf("expected first-depth edges, got %+v", edge)
		}
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Depth != 1 ||
		len(result.Dependents) != 1 || result.Dependents[0].Depth != 1 {
		t.Fatalf("localized nodes lost depth: %+v", result)
	}
}

func TestLocalizedContextCycleKeepsShortestUniqueEdges(t *testing.T) {
	a := relationshipTestNode("class:App\\A", "A")
	b := relationshipTestNode("class:App\\B", "B")
	relationshipTestLink(a, b)
	relationshipTestLink(b, a)
	e := &Engine{graph: &models.DependencyGraph{Nodes: map[string]*models.DependencyNode{
		a.ID: a,
		b.ID: b,
	}}}

	result := e.LocalizedContext("A", 2)
	if len(result.Relationships) != 2 {
		t.Fatalf("cycle should contain two unique edges, got %+v", result.Relationships)
	}
	for _, edge := range result.Relationships {
		if edge.Depth != 1 {
			t.Fatalf("expected shortest depth 1 after cycle traversal, got %+v", edge)
		}
	}
	if len(result.Dependencies) != 1 || len(result.Dependents) != 1 {
		t.Fatalf("cycle should not duplicate neighbor nodes: %+v", result)
	}
}

func TestLocalizedContextNotFoundHasEmptyCollections(t *testing.T) {
	e := buildTestEngine()
	result := e.LocalizedContext("missing", 2)
	if result.Resolution.MatchType != "not_found" || result.TargetNodes == nil || result.Relationships == nil {
		t.Fatalf("unexpected missing localized context: %+v", result)
	}
}
