package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_YAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tukey.yaml")
	content := `
language: php
excludeDirs:
  - vendor
  - node_modules
outputFile: report.json
verbose: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Language != "php" {
		t.Errorf("expected php, got %q", cfg.Language)
	}
	if len(cfg.ExcludeDirs) != 2 {
		t.Errorf("expected 2 excludeDirs, got %d", len(cfg.ExcludeDirs))
	}
	if cfg.OutputFile != "report.json" {
		t.Errorf("expected report.json, got %q", cfg.OutputFile)
	}
	if !cfg.Verbose {
		t.Errorf("expected verbose = true")
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tukey.json")
	content := `{
		"language": "go",
		"excludeDirs": ["vendor"],
		"outputFile": "out.json",
		"verbose": false
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Language != "go" {
		t.Errorf("expected go, got %q", cfg.Language)
	}
	if cfg.OutputFile != "out.json" {
		t.Errorf("expected out.json, got %q", cfg.OutputFile)
	}
	if cfg.Verbose {
		t.Errorf("expected verbose = false")
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Expect defaults (empty config)
	if cfg.Language != "" {
		t.Errorf("expected empty language, got %q", cfg.Language)
	}
	if len(cfg.ExcludeDirs) != 0 {
		t.Errorf("expected no excludeDirs, got %d", len(cfg.ExcludeDirs))
	}
}
