package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func sampleParsedFile() *models.ParsedFile {
	return &models.ParsedFile{
		Path:      "app/Models/User.php",
		Namespace: "App\\Models",
		Uses:      []string{"App\\Services\\Mailer"},
		Elements: []models.CodeElement{
			{
				Type:       "class",
				Name:       "User",
				Namespace:  "App\\Models",
				Line:       8,
				IsAbstract: false,
			},
			{
				Type:      "function",
				Name:      "formatPhone",
				Namespace: "App\\Models",
				Line:      15,
			},
		},
		Usage: []models.UsageElement{
			{
				Type:    "function_call",
				Name:    "formatPhone",
				Context: "User",
				Line:    22,
			},
		},
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	dt := NewDependencyTracker()
	graph := dt.BuildDependencyGraph([]*models.ParsedFile{sampleParsedFile()})

	if graph.TotalNodes == 0 {
		t.Fatalf("expected nodes to be created, got 0")
	}
	if graph.TotalEdges == 0 {
		t.Errorf("expected at least one edge, got 0")
	}
	if len(graph.HighlyDepended) == 0 {
		t.Errorf("expected highly depended nodes, got 0")
	}
	if len(graph.ComplexNodes) == 0 {
		t.Errorf("expected complex nodes, got 0")
	}
}

func TestExportToJSON(t *testing.T) {
	dt := NewDependencyTracker()
	graph := dt.BuildDependencyGraph([]*models.ParsedFile{sampleParsedFile()})
	if graph.TotalNodes == 0 {
		t.Fatalf("graph not built correctly")
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "graph.json")
	if err := dt.ExportToJSON(outPath); err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected file at %s, got error: %v", outPath, err)
	}
}

func TestCalculateComplexityScore(t *testing.T) {
	dt := NewDependencyTracker()

	// class
	classEl := &models.CodeElement{Type: "class", IsAbstract: true}
	if got := dt.calculateComplexityScore(classEl); got < 7 {
		t.Errorf("expected abstract class complexity >= 7, got %d", got)
	}

	// function with 2 params
	fnEl := &models.CodeElement{Type: "function", Parameters: []string{"a", "b"}}
	if got := dt.calculateComplexityScore(fnEl); got < 5 {
		t.Errorf("expected function complexity >= 5, got %d", got)
	}

	// static property
	propEl := &models.CodeElement{Type: "property", IsStatic: true}
	if got := dt.calculateComplexityScore(propEl); got != 3 {
		t.Errorf("expected static property complexity 3, got %d", got)
	}
}
