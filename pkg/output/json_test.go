package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func TestJSONExporter_Export(t *testing.T) {
	res := makeDummyResult()
	res.Graph.Diagnostics = []models.AnalysisDiagnostic{{
		Code: models.DiagnosticParseError, Severity: models.DiagnosticSeverityError,
		File: "broken.php", Message: "invalid syntax", Count: 1,
	}}
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
	if !strings.Contains(out, `"code": "parse_error"`) {
		t.Errorf("expected diagnostics in JSON, got:\n%s", out)
	}
}
