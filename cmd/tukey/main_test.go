package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/boone-studios/tukey/internal/config"
)

func captureOutput(f func()) string {
	// Save original stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function
	f()

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestShowHelp_OutputContainsUsageAndFlags(t *testing.T) {
	out := captureOutput(showHelp)

	if !strings.Contains(out, "USAGE:") {
		t.Errorf("help output missing USAGE section:\n%s", out)
	}
	if !strings.Contains(out, "FLAGS:") {
		t.Errorf("help output missing FLAGS section:\n%s", out)
	}
	if !strings.Contains(out, "Tukey v") {
		t.Errorf("help output missing version string:\n%s", out)
	}
}

func TestParseArgs_VerboseAndOutput(t *testing.T) {
	os.Args = []string{"tukey", "-v", "-o", "out.json", "myproj"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Verbose {
		t.Errorf("expected verbose")
	}
	if cfg.OutputFile != "out.json" {
		t.Errorf("expected out.json, got %s", cfg.OutputFile)
	}
	if cfg.RootPath != "myproj" {
		t.Errorf("expected root path myproj, got %s", cfg.RootPath)
	}
}

func TestParseArgs_ExcludeDirs(t *testing.T) {
	os.Args = []string{"tukey", "--exclude", "vendor", "--exclude", "tests", "myproj"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"vendor", "tests"}
	if !reflect.DeepEqual(cfg.ExcludeDirs, want) {
		t.Errorf("expected %v, got %v", want, cfg.ExcludeDirs)
	}
}

func TestParseArgs_Errors(t *testing.T) {
	tests := [][]string{
		{"tukey", "--output"},  // missing filename
		{"tukey", "--exclude"}, // missing dir
		{"tukey", "-x"},        // unknown flag
	}
	for _, args := range tests {
		os.Args = args
		_, err := parseArgs()
		if err == nil {
			t.Errorf("expected error for args %v", args)
		}
	}
}

func TestParseArgs_NoArgsShowsHelp(t *testing.T) {
	os.Args = []string{"tukey"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}
	if !cfg.ShowHelp {
		t.Errorf("expected ShowHelp to be true when no args")
	}
}

func TestMergeConfigs_FileProvidesDefaults(t *testing.T) {
	argv := &Config{
		RootPath: "myproj",
		// nothing else set
	}
	fileCfg := &config.FileConfig{
		Language:    "php",
		ExcludeDirs: []string{"vendor", "tests"},
		OutputFile:  "report.json",
		Verbose:     true,
	}

	merged := mergeConfigs(argv, fileCfg)

	if merged.Language != "php" {
		t.Errorf("expected language php, got %s", merged.Language)
	}
	if merged.OutputFile != "report.json" {
		t.Errorf("expected report.json, got %s", merged.OutputFile)
	}
	if !merged.Verbose {
		t.Errorf("expected verbose = true")
	}
	if len(merged.ExcludeDirs) != 2 {
		t.Errorf("expected 2 excludeDirs, got %d", len(merged.ExcludeDirs))
	}
}

func TestMergeConfigs_CLIOverridesFile(t *testing.T) {
	argv := &Config{
		RootPath:    "myproj",
		Language:    "go",
		OutputFile:  "cli.json",
		Verbose:     true,
		ExcludeDirs: []string{"cli-only"},
	}
	fileCfg := &config.FileConfig{
		Language:    "php",
		ExcludeDirs: []string{"vendor"},
		OutputFile:  "file.json",
		Verbose:     false,
	}

	merged := mergeConfigs(argv, fileCfg)

	if merged.Language != "go" { // CLI wins
		t.Errorf("expected go, got %s", merged.Language)
	}
	if merged.OutputFile != "cli.json" {
		t.Errorf("expected cli.json, got %s", merged.OutputFile)
	}
	if !merged.Verbose {
		t.Errorf("expected verbose = true from CLI")
	}
	if len(merged.ExcludeDirs) != 2 {
		t.Errorf("expected merged excludeDirs length 2, got %d", len(merged.ExcludeDirs))
	}
}
