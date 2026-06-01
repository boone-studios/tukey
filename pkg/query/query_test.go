// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package query

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

// buildTestEngine returns an Engine backed by a synthetic graph for unit tests
func buildTestEngine() *Engine {
	gateway := &models.DependencyNode{
		ID:        "class:App\\Factories\\GatewayFactory:15",
		Name:      "GatewayFactory",
		Type:      "class",
		File:      "app/Factories/GatewayFactory.php",
		Namespace: "App\\Factories",
		Line:      15,
		Score:     12,
		Dependencies: map[string]*models.DependencyRef{
			"class:App\\Http\\Client:8": {
				TargetID:   "class:App\\Http\\Client:8",
				TargetName: "HttpClient",
				Type:       "instantiation",
				Count:      1,
				Lines:      []int{20},
			},
		},
		Dependents: map[string]*models.DependencyRef{
			"class:App\\Services\\PaymentService:5": {
				TargetID:   "class:App\\Services\\PaymentService:5",
				TargetName: "PaymentService",
				Type:       "instantiation",
				Count:      2,
				Lines:      []int{30, 45},
			},
		},
	}
	httpClient := &models.DependencyNode{
		ID:           "class:App\\Http\\Client:8",
		Name:         "HttpClient",
		Type:         "class",
		File:         "app/Http/Client.php",
		Namespace:    "App\\Http",
		Line:         8,
		Score:        5,
		Dependencies: map[string]*models.DependencyRef{},
		Dependents: map[string]*models.DependencyRef{
			"class:App\\Factories\\GatewayFactory:15": {
				TargetID:   "class:App\\Factories\\GatewayFactory:15",
				TargetName: "GatewayFactory",
				Type:       "instantiation",
				Count:      1,
				Lines:      []int{20},
			},
		},
	}
	payment := &models.DependencyNode{
		ID:        "class:App\\Services\\PaymentService:5",
		Name:      "PaymentService",
		Type:      "class",
		File:      "app/Services/PaymentService.php",
		Namespace: "App\\Services",
		Line:      5,
		Score:     18,
		Dependencies: map[string]*models.DependencyRef{
			"class:App\\Factories\\GatewayFactory:15": {
				TargetID:   "class:App\\Factories\\GatewayFactory:15",
				TargetName: "GatewayFactory",
				Type:       "instantiation",
				Count:      2,
				Lines:      []int{30, 45},
			},
		},
		Dependents: map[string]*models.DependencyRef{},
	}
	orphan := &models.DependencyNode{
		ID:           "function:deadHelper:99",
		Name:         "deadHelper",
		Type:         "function",
		File:         "app/helpers.php",
		Line:         99,
		Score:        3,
		Dependencies: map[string]*models.DependencyRef{},
		Dependents:   map[string]*models.DependencyRef{},
	}

	graph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{
			gateway.ID:    gateway,
			httpClient.ID: httpClient,
			payment.ID:    payment,
			orphan.ID:     orphan,
		},
		TotalNodes: 4,
		TotalEdges: 3,
		Orphans:    []*models.DependencyNode{orphan},
	}
	return &Engine{graph: graph}
}

func TestFind_ExactName(t *testing.T) {
	e := buildTestEngine()
	r := e.Find("GatewayFactory")
	if r.Query != "find" {
		t.Errorf("query field: got %q want %q", r.Query, "find")
	}
	if r.Count != 1 {
		t.Errorf("count: got %d want 1", r.Count)
	}
	if r.Results[0].Name != "GatewayFactory" {
		t.Errorf("result name: got %q", r.Results[0].Name)
	}
}

func TestFind_CaseInsensitive(t *testing.T) {
	e := buildTestEngine()
	r := e.Find("gateway")
	if r.Count != 1 {
		t.Errorf("case-insensitive find: got %d results, want 1", r.Count)
	}
}

func TestFind_NoMatch(t *testing.T) {
	e := buildTestEngine()
	r := e.Find("NonExistent")
	if r.Count != 0 {
		t.Errorf("expected 0 results, got %d", r.Count)
	}
	if r.Results == nil {
		t.Error("results should be empty slice, not nil")
	}
}

func TestCallers_ReturnsIncomingEdges(t *testing.T) {
	e := buildTestEngine()
	r := e.Callers("GatewayFactory")
	if r.Query != "callers" {
		t.Errorf("query field: got %q", r.Query)
	}
	if r.Count != 1 {
		t.Errorf("callers count: got %d want 1", r.Count)
	}
	caller := r.Results[0]
	if caller.Name != "PaymentService" {
		t.Errorf("caller name: got %q want PaymentService", caller.Name)
	}
	if caller.RefType != "instantiation" {
		t.Errorf("refType: got %q want instantiation", caller.RefType)
	}
	if caller.RefCount != 2 {
		t.Errorf("refCount: got %d want 2", caller.RefCount)
	}
}

func TestCallers_Unknown(t *testing.T) {
	e := buildTestEngine()
	r := e.Callers("NoSuchThing")
	if r.Count != 0 {
		t.Errorf("expected 0 callers, got %d", r.Count)
	}
	if r.Results == nil {
		t.Error("results should be empty slice, not nil")
	}
}

func TestDependents_ReturnsOutgoingEdges(t *testing.T) {
	e := buildTestEngine()
	r := e.Dependents("GatewayFactory")
	if r.Query != "dependents" {
		t.Errorf("query field: got %q", r.Query)
	}
	if r.Count != 1 {
		t.Errorf("dependents count: got %d want 1", r.Count)
	}
	dep := r.Results[0]
	if dep.Name != "HttpClient" {
		t.Errorf("dep name: got %q want HttpClient", dep.Name)
	}
}

func TestOrphans(t *testing.T) {
	e := buildTestEngine()
	r := e.Orphans()
	if r.Query != "orphans" {
		t.Errorf("query field: got %q", r.Query)
	}
	if r.Count != 1 {
		t.Errorf("orphan count: got %d want 1", r.Count)
	}
	if r.Results[0].Name != "deadHelper" {
		t.Errorf("orphan name: got %q", r.Results[0].Name)
	}
}

func TestLoad_InvalidPath(t *testing.T) {
	_, err := Load("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	f, _ := os.CreateTemp("", "tukey-test-*.json")
	f.WriteString("not json")
	f.Close()
	defer os.Remove(f.Name())

	_, err := Load(f.Name())
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoad_MissingGraphField(t *testing.T) {
	f, _ := os.CreateTemp("", "tukey-test-*.json")
	f.WriteString(`{"totalFiles":1}`)
	f.Close()
	defer os.Remove(f.Name())

	_, err := Load(f.Name())
	if err == nil {
		t.Error("expected error when graph field is missing")
	}
}

func TestLoad_RoundTrip(t *testing.T) {
	e := buildTestEngine()

	// Marshal the engine's graph into an analysis envelope and write to a temp file
	env := analysisEnvelope{Graph: e.graph}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	f, _ := os.CreateTemp("", "tukey-roundtrip-*.json")
	f.Write(data)
	f.Close()
	defer os.Remove(f.Name())

	loaded, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	r := loaded.Find("GatewayFactory")
	if r.Count != 1 {
		t.Errorf("round-trip find: got %d results, want 1", r.Count)
	}
}

func TestQueryResult_EmptyResultsNotNull(t *testing.T) {
	e := buildTestEngine()
	r := e.Find("zzz_no_match")
	data, _ := json.Marshal(r)
	if string(data) == "" {
		t.Fatal("marshal failed")
	}
	// "results":null would break agent JSON parsers; it must be an array
	if !contains(string(data), `"results":[]`) {
		t.Errorf("expected results to be [] not null, got: %s", string(data))
	}
}

func TestQuery_ResilientMatching(t *testing.T) {
	methodNode := &models.DependencyNode{
		ID:           "method:App\\Services\\PaymentService\\pay:30",
		Name:         "pay",
		Type:         "method",
		File:         "app/Services/PaymentService.php",
		Namespace:    "App\\Services",
		ClassName:    "PaymentService",
		Line:         30,
		Score:        5,
		Dependencies: map[string]*models.DependencyRef{},
		Dependents:   map[string]*models.DependencyRef{},
	}
	classNode := &models.DependencyNode{
		ID:        "class:App\\Services\\PaymentService:5",
		Name:      "PaymentService",
		Type:      "class",
		File:      "app/Services/PaymentService.php",
		Namespace: "App\\Services",
		Line:      5,
		Score:     18,
		Dependencies: map[string]*models.DependencyRef{
			methodNode.ID: {
				TargetID:   methodNode.ID,
				TargetName: methodNode.Name,
				Type:       "calls",
				Count:      1,
				Lines:      []int{12},
			},
		},
		Dependents: map[string]*models.DependencyRef{},
	}
	methodNode.Dependents[classNode.ID] = &models.DependencyRef{
		TargetID:   classNode.ID,
		TargetName: classNode.Name,
		Type:       "calls",
		Count:      1,
		Lines:      []int{12},
	}

	graph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{
			classNode.ID:  classNode,
			methodNode.ID: methodNode,
		},
		TotalNodes: 2,
		TotalEdges: 1,
	}

	e := &Engine{graph: graph}

	// 1. Test exact ID match
	t.Run("Exact ID Match", func(t *testing.T) {
		r := e.Find("method:App\\Services\\PaymentService\\pay:30")
		if r.Count != 1 || r.Results[0].Name != "pay" {
			t.Errorf("ID lookup failed: got %d results, want 1 with name 'pay'", r.Count)
		}
	})

	// 2. Test fully-qualified class match
	t.Run("Fully-Qualified Class Match", func(t *testing.T) {
		r := e.Find("App\\Services\\PaymentService")
		// Find does substring matching, so it will match both the class
		// and the class's method because the method's namespaced FQDN includes the class
		if r.Count != 2 {
			t.Errorf("FQ class lookup via Find failed: got %d results, want 2", r.Count)
		}

		// findNodes does exact matching first, so it should resolve exactly to the class node
		resolved := e.findNodes("App\\Services\\PaymentService")
		if len(resolved) != 1 || resolved[0].Name != "PaymentService" {
			t.Errorf("FQ class lookup via findNodes failed: got %d results, want 1 with name 'PaymentService'", len(resolved))
		}
	})

	// 3. Test class-scoped method match
	t.Run("Class-Scoped Method Match", func(t *testing.T) {
		r := e.Find("PaymentService::pay")
		if r.Count != 1 || r.Results[0].Name != "pay" {
			t.Errorf("Class-scoped method lookup failed: got %d results, want 1 with name 'pay'", r.Count)
		}
	})

	// 4. Test fully-qualified namespaced method match
	t.Run("Fully-Qualified Method Match", func(t *testing.T) {
		r := e.Find("App\\Services\\PaymentService::pay")
		if r.Count != 1 || r.Results[0].Name != "pay" {
			t.Errorf("FQ method lookup failed: got %d results, want 1 with name 'pay'", r.Count)
		}
	})

	// 5. Test callers and dependents with resilient matches
	t.Run("Callers Resilient Match", func(t *testing.T) {
		r := e.Callers("App\\Services\\PaymentService::pay")
		if r.Count != 1 || r.Results[0].Name != "PaymentService" {
			t.Errorf("Callers with resilient match failed: got %d, want 1 with name 'PaymentService'", r.Count)
		}
	})

	t.Run("Dependents Resilient Match", func(t *testing.T) {
		r := e.Dependents("class:App\\Services\\PaymentService:5")
		if r.Count != 1 || r.Results[0].Name != "pay" {
			t.Errorf("Dependents with resilient match failed: got %d, want 1 with name 'pay'", r.Count)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
