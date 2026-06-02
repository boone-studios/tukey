package analyzer

import (
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


// TestFindTargetNode_MethodCall verifies that a method call recorded in the Go
// parser's "Namespace\ClassName::method" format is correctly linked to the
// method node, so the method is not reported as an orphan.
func TestFindTargetNode_MethodCall(t *testing.T) {
	file := &models.ParsedFile{
		Path:      "internal/tracker.go",
		Namespace: "myapp",
		Elements: []models.CodeElement{
			{Type: "method", Name: "helper", Namespace: "myapp", ClassName: "Tracker", Line: 5},
			{Type: "method", Name: "caller", Namespace: "myapp", ClassName: "Tracker", Line: 10},
		},
		Usage: []models.UsageElement{
			{Type: "method_call", Name: "myapp\\Tracker::helper", Context: "caller", Line: 12},
		},
	}

	dt := NewDependencyTracker()
	graph := dt.BuildDependencyGraph([]*models.ParsedFile{file})

	var helperNode *models.DependencyNode
	for _, node := range graph.Nodes {
		if node.Name == "helper" && node.ClassName == "Tracker" {
			helperNode = node
			break
		}
	}
	if helperNode == nil {
		t.Fatal("helper node not found in graph")
	}
	if len(helperNode.Dependents) == 0 {
		t.Error("helper should have dependents (called by caller), got 0")
	}
	for _, orphan := range graph.Orphans {
		if orphan.Name == "helper" {
			t.Error("helper should not be reported as an orphan")
		}
	}
}

// TestFindTargetNode_FieldAccess verifies that a struct field access recorded in
// the Go parser's "Namespace\ClassName::field" format is correctly linked to the
// field node, so it is not reported as an orphan.
func TestFindTargetNode_FieldAccess(t *testing.T) {
	file := &models.ParsedFile{
		Path:      "pkg/report.go",
		Namespace: "myapp",
		Elements: []models.CodeElement{
			{Type: "property", Name: "Total", Namespace: "myapp", ClassName: "Report", Line: 3},
			{Type: "function", Name: "process", Namespace: "myapp", Line: 8},
		},
		Usage: []models.UsageElement{
			{Type: "property", Name: "myapp\\Report::Total", Context: "process", Line: 10},
		},
	}

	dt := NewDependencyTracker()
	graph := dt.BuildDependencyGraph([]*models.ParsedFile{file})

	var totalNode *models.DependencyNode
	for _, node := range graph.Nodes {
		if node.Name == "Total" && node.ClassName == "Report" {
			totalNode = node
			break
		}
	}
	if totalNode == nil {
		t.Fatal("Total node not found in graph")
	}
	if len(totalNode.Dependents) == 0 {
		t.Error("Total should have dependents (accessed by process), got 0")
	}
	for _, orphan := range graph.Orphans {
		if orphan.Name == "Total" {
			t.Error("Total should not be reported as an orphan")
		}
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
