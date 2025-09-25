package parser

import (
	"testing"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/progress"
)

// DummyParser is a simple parser for testing the registry.
type DummyParser struct{}

func (d *DummyParser) ProcessFiles(files []models.FileInfo, pb *progress.ProgressBar) ([]*models.ParsedFile, error) {
	if pb != nil {
		for range files {
			pb.Update(1)
		}
		pb.Finish()
	}
	return []*models.ParsedFile{{Path: "dummy"}}, nil
}

func (d *DummyParser) Language() string {
	return "dummy"
}

func (d *DummyParser) FileExtensions() []string {
	return []string{".dummy"}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry = map[string]LanguageParser{}

	d := &DummyParser{}
	Register(d)

	// Should be retrievable
	p, ok := Get("dummy")
	if !ok {
		t.Fatalf("expected parser to be registered")
	}
	if p.Language() != "dummy" {
		t.Errorf("expected Language= dummy, got %s", p.Language())
	}

	// SupportedLanguages should include a dummy parser
	supported := SupportedLanguages()
	found := false
	for _, lang := range supported {
		if lang == "dummy" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected SupportedLanguages to include dummy, got %v", supported)
	}
}

func TestRegistry_DuplicatePanics(t *testing.T) {
	registry = map[string]LanguageParser{}

	d := &DummyParser{}
	Register(d)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on duplicate registration")
		}
	}()

	// Registering the same language again should panic
	Register(d)
}
