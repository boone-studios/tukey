package query

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func buildPaginatedEngine(count int) *Engine {
	graph := &models.DependencyGraph{Nodes: make(map[string]*models.DependencyNode)}
	for i := 0; i < count; i++ {
		node := relationshipTestNode(fmt.Sprintf("function:Node%03d", i), fmt.Sprintf("Node%03d", i))
		graph.Nodes[node.ID] = node
	}
	return &Engine{graph: graph}
}

func TestFindPagination(t *testing.T) {
	e := buildPaginatedEngine(120)
	first, err := e.FindWithOptions("Node", PageOptions{Limit: 2})
	if err != nil {
		t.Fatalf("FindWithOptions: %v", err)
	}
	if first.Count != 2 || first.Page.Total != 120 || first.Page.Returned != 2 || first.Page.NextCursor == "" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	second, err := e.FindWithOptions("Node", PageOptions{Limit: 2, Cursor: first.Page.NextCursor})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if second.Count != 2 || second.Results[0].ID == first.Results[0].ID || second.Page.NextCursor == "" {
		t.Fatalf("cursor did not advance: %+v", second)
	}

	defaultPage, err := e.FindWithOptions("Node", PageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if defaultPage.Count != DefaultPageLimit || defaultPage.Page.Limit != DefaultPageLimit {
		t.Fatalf("default limit not applied: %+v", defaultPage.Page)
	}
	maxPage, err := e.FindWithOptions("Node", PageOptions{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if maxPage.Count != MaxPageLimit || maxPage.Page.Limit != MaxPageLimit {
		t.Fatalf("maximum limit not applied: %+v", maxPage.Page)
	}

	unbounded := e.Find("Node")
	if unbounded.Count != 120 || unbounded.Page != nil {
		t.Fatalf("compatibility wrapper should be unbounded: count=%d page=%+v", unbounded.Count, unbounded.Page)
	}
}

func TestPaginationRejectsInvalidInput(t *testing.T) {
	e := buildPaginatedEngine(2)
	if _, err := e.FindWithOptions("Node", PageOptions{Cursor: "not-a-cursor"}); err == nil {
		t.Fatal("expected invalid cursor error")
	}
	if _, err := e.FindWithOptions("Node", PageOptions{Limit: -1}); err == nil {
		t.Fatal("expected negative limit error")
	}
}

func TestRelationshipPaginationBoundsEveryCollection(t *testing.T) {
	e, _ := buildAmbiguousRelationshipEngine()
	result, err := e.CallersWithOptions("save", PageOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || len(result.Resolution.Targets) != 1 || len(result.Relationships) != 1 {
		t.Fatalf("expected one item per collection, got %+v", result)
	}
	if result.Resolution.TargetCount != 2 || result.Page.Total != 6 || result.Page.Returned != 3 || result.Page.NextCursor == "" {
		t.Fatalf("unexpected relationship page metadata: %+v", result)
	}
	last, err := e.CallersWithOptions("save", PageOptions{Limit: 1, Cursor: "Mg"})
	if err != nil {
		t.Fatal(err)
	}
	if last.Count != 0 || len(last.Resolution.Targets) != 0 || len(last.Relationships) != 0 || last.Page.NextCursor != "" {
		t.Fatalf("unexpected final relationship page: %+v", last)
	}
}

func TestDiagnosticsPagination(t *testing.T) {
	e := &Engine{graph: &models.DependencyGraph{Diagnostics: []models.AnalysisDiagnostic{
		{File: "a.php", Count: 1},
		{File: "b.php", Count: 1},
		{File: "c.php", Count: 1},
	}}}
	result, err := e.DiagnosticsWithOptions(PageOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 2 || result.Page.Total != 3 || result.Page.NextCursor == "" {
		t.Fatalf("unexpected diagnostics page: %+v", result)
	}
}

func TestLocalizedContextDepthCap(t *testing.T) {
	nodes := make([]*models.DependencyNode, 7)
	graph := &models.DependencyGraph{Nodes: make(map[string]*models.DependencyNode)}
	for i := range nodes {
		nodes[i] = relationshipTestNode(fmt.Sprintf("class:App\\Node%d", i), fmt.Sprintf("Node%d", i))
		graph.Nodes[nodes[i].ID] = nodes[i]
		if i > 0 {
			relationshipTestLink(nodes[i-1], nodes[i])
		}
	}
	e := &Engine{graph: graph}
	result, err := e.LocalizedContextWithOptions("Node0", 99, PageOptions{Unbounded: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != MaxContextDepth {
		t.Fatalf("expected %d dependencies after depth cap, got %d", MaxContextDepth, len(result.Dependencies))
	}
	if result.Dependencies[len(result.Dependencies)-1].Depth != MaxContextDepth {
		t.Fatalf("unexpected maximum depth: %+v", result.Dependencies)
	}
}

func TestStatusPagination(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("package sample\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	e := &Engine{metadata: models.AnalysisMetadata{
		SchemaVersion:  models.AnalysisSchemaVersion,
		GeneratedAt:    "2026-01-01T00:00:00Z",
		RootPath:       root,
		FileExtensions: []string{".go"},
		SourceFiles:    []models.SourceFileMetadata{},
	}}
	first, err := e.StatusWithOptions(PageOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "stale" || len(first.Changes) != 2 || first.ChangeCount != 3 || first.Page.NextCursor == "" {
		t.Fatalf("unexpected first status page: %+v", first)
	}
	second, err := e.StatusWithOptions(PageOptions{Limit: 2, Cursor: first.Page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Changes) != 1 || second.Page.NextCursor != "" {
		t.Fatalf("unexpected final status page: %+v", second)
	}
}
