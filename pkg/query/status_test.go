package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/source"
)

func buildStatusEngine(t *testing.T, root string, files []models.FileInfo) *Engine {
	t.Helper()
	snapshot, err := source.Snapshot(files)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	return &Engine{metadata: models.AnalysisMetadata{
		SchemaVersion:  models.AnalysisSchemaVersion,
		GeneratedAt:    "2026-01-01T00:00:00Z",
		RootPath:       root,
		FileExtensions: []string{".go"},
		SourceFiles:    snapshot,
	}}
}

func sourceFile(path string) models.FileInfo {
	info, err := os.Stat(path)
	if err != nil {
		return models.FileInfo{Path: path, RelativePath: filepath.Base(path)}
	}
	return models.FileInfo{Path: path, RelativePath: filepath.Base(path), Size: info.Size()}
}

func TestStatusFresh(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	engine := buildStatusEngine(t, root, []models.FileInfo{sourceFile(path)})

	result, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if result.Status != "fresh" || result.CheckedFiles != 1 || result.ChangeCount != 0 {
		t.Fatalf("unexpected fresh status: %+v", result)
	}
	if result.Changes == nil {
		t.Fatal("changes should be an empty array, not nil")
	}
}

func TestStatusDetectsContentChangeWithSameSize(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n// a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	engine := buildStatusEngine(t, root, []models.FileInfo{sourceFile(path)})
	if err := os.WriteFile(path, []byte("package main\n// b\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if result.Status != "stale" || len(result.Changes) != 1 || result.Changes[0].Change != "modified" {
		t.Fatalf("expected modified-file status, got %+v", result)
	}
}

func TestStatusDetectsAddedAndDeletedFiles(t *testing.T) {
	root := t.TempDir()
	keptPath := filepath.Join(root, "kept.go")
	deletedPath := filepath.Join(root, "deleted.go")
	for _, path := range []string{keptPath, deletedPath} {
		if err := os.WriteFile(path, []byte("package sample\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	engine := buildStatusEngine(t, root, []models.FileInfo{sourceFile(keptPath), sourceFile(deletedPath)})
	if err := os.Remove(deletedPath); err != nil {
		t.Fatal(err)
	}
	addedPath := filepath.Join(root, "added.go")
	if err := os.WriteFile(addedPath, []byte("package sample\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if result.Status != "stale" || result.ChangeCount != 2 {
		t.Fatalf("expected added and deleted changes, got %+v", result)
	}
	if result.Changes[0].Path != "added.go" || result.Changes[0].Change != "added" ||
		result.Changes[1].Path != "deleted.go" || result.Changes[1].Change != "deleted" {
		t.Fatalf("unexpected changes: %+v", result.Changes)
	}
}

func TestStatusLegacyAnalysisIsUnknown(t *testing.T) {
	engine := &Engine{}
	result, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if result.Status != "unknown" || result.Changes == nil {
		t.Fatalf("unexpected legacy status: %+v", result)
	}
}
