package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONExporter_Export(t *testing.T) {
	res := makeDummyResult()
	je := NewJSONExporter()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "result.json")
	if err := je.Export(res, outPath); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, `"totalFiles": 1`) {
		t.Errorf("expected totalFiles=1 in JSON, got:\n%s", out)
	}
	if !strings.Contains(out, `"graph"`) {
		t.Errorf("expected graph in JSON, got:\n%s", out)
	}
}

func TestJSONExporter_ExportGraph(t *testing.T) {
	res := makeDummyResult()
	je := NewJSONExporter()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "graph.json")
	if err := je.ExportGraph(res.Graph, outPath); err != nil {
		t.Fatalf("ExportGraph failed: %v", err)
	}

	data, _ := os.ReadFile(outPath)
	if !strings.Contains(string(data), `"totalNodes": 1`) {
		t.Errorf("expected graph JSON to contain totalNodes=1")
	}
}
