package lang

import (
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/parser"
)

func TestProcessFilesRetainsParseDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		parser    parser.LanguageParser
		extension string
	}{
		{name: "PHP", parser: NewPHPParser(), extension: ".php"},
		{name: "JavaScript", parser: NewJSParser(), extension: ".js"},
		{name: "Go", parser: NewGoParser(), extension: ".go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing"+tt.extension)
			files := []models.FileInfo{{Path: path, RelativePath: filepath.Base(path)}}
			parsed, err := tt.parser.ProcessFiles(files, nil)
			if err != nil {
				t.Fatalf("ProcessFiles returned error: %v", err)
			}
			if len(parsed) != 1 {
				t.Fatalf("expected failed file placeholder, got %d files", len(parsed))
			}
			if len(parsed[0].Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %+v", parsed[0].Diagnostics)
			}
			diagnostic := parsed[0].Diagnostics[0]
			if diagnostic.Code != models.DiagnosticParseError || diagnostic.Severity != models.DiagnosticSeverityError {
				t.Fatalf("unexpected diagnostic: %+v", diagnostic)
			}
			if diagnostic.File != path || diagnostic.Count != 1 || diagnostic.Message == "" {
				t.Fatalf("diagnostic lost failure context: %+v", diagnostic)
			}
		})
	}
}
