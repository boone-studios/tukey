package parser_test

import (
	"testing"

	_ "github.com/boone-studios/tukey/internal/lang"
	"github.com/boone-studios/tukey/internal/parser"
)

// TestAtLeastOneParserRegistered ensures the registry has a usable parser
// (e.g. PHPParser, JSParser, etc.) when the package is imported.
func TestAtLeastOneParserRegistered(t *testing.T) {
	langs := parser.SupportedLanguages()
	if len(langs) == 0 {
		t.Fatal("expected at least one parser registered, got none")
	}

	// Try the first one
	lang := langs[0]
	p, ok := parser.Get(lang)
	if !ok {
		t.Fatalf("parser %q was listed but not retrievable", lang)
	}

	// Sanity check: must return a non-empty language and some extensions
	if p.Language() == "" {
		t.Errorf("parser returned empty Language() for %q", lang)
	}
	if len(p.FileExtensions()) == 0 {
		t.Errorf("parser %q returned no extensions", lang)
	}
}
