package output

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func makeDummyResult() *models.AnalysisResult {
	node := &models.DependencyNode{
		ID:   "1",
		Name: "User",
		Type: "class",
		File: "app/User.php",
	}
	graph := &models.DependencyGraph{
		Nodes:          map[string]*models.DependencyNode{"1": node},
		TotalNodes:     1,
		TotalEdges:     0,
		Orphans:        []*models.DependencyNode{node},
		HighlyDepended: []*models.DependencyNode{node},
		ComplexNodes:   []*models.DependencyNode{node},
	}
	return &models.AnalysisResult{
		Graph:          graph,
		ParsedFiles:    []*models.ParsedFile{},
		TotalFiles:     1,
		TotalElements:  1,
		ProcessingTime: "1s",
	}
}

func TestConsoleFormatter_PrintSummary_NonVerbose(t *testing.T) {
	res := makeDummyResult()
	cf := NewConsoleFormatter()
	out := captureOutput(func() { cf.PrintSummary(res, false) })

	if !strings.Contains(out, "DEPENDENCY ANALYSIS SUMMARY") {
		t.Errorf("expected summary header in output:\n%s", out)
	}
	if !strings.Contains(out, "Tip: Use -v") {
		t.Errorf("expected verbose tip in output:\n%s", out)
	}
}

func TestConsoleFormatter_PrintSummary_Verbose(t *testing.T) {
	res := makeDummyResult()
	cf := NewConsoleFormatter()
	out := captureOutput(func() { cf.PrintSummary(res, true) })

	if !strings.Contains(out, "VERBOSE MODE") {
		t.Errorf("expected verbose marker in output:\n%s", out)
	}
	if !strings.Contains(out, "FUNCTION USAGE REPORT") {
		t.Errorf("expected function usage report in verbose output:\n%s", out)
	}
}
