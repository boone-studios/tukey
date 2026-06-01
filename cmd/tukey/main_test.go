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

func TestParseQueryArgs_Find(t *testing.T) {
	cfg, err := parseQueryArgs([]string{"--find", "GatewayFactory", "analysis.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.mode != "find" {
		t.Errorf("mode: got %q want find", cfg.mode)
	}
	if cfg.term != "GatewayFactory" {
		t.Errorf("term: got %q want GatewayFactory", cfg.term)
	}
	if cfg.inputFile != "analysis.json" {
		t.Errorf("inputFile: got %q want analysis.json", cfg.inputFile)
	}
}

func TestParseQueryArgs_Orphans(t *testing.T) {
	cfg, err := parseQueryArgs([]string{"--orphans", "analysis.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.mode != "orphans" {
		t.Errorf("mode: got %q want orphans", cfg.mode)
	}
	if cfg.term != "" {
		t.Errorf("term should be empty for --orphans")
	}
}

func TestParseQueryArgs_Help(t *testing.T) {
	cfg, err := parseQueryArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil cfg for --help")
	}
}

func TestParseQueryArgs_MissingMode(t *testing.T) {
	_, err := parseQueryArgs([]string{"analysis.json"})
	if err == nil {
		t.Error("expected error when no mode flag given")
	}
}

func TestParseQueryArgs_MissingFile(t *testing.T) {
	_, err := parseQueryArgs([]string{"--find", "Foo"})
	if err == nil {
		t.Error("expected error when no input file given")
	}
}

func TestParseQueryArgs_MissingTerm(t *testing.T) {
	tests := [][]string{
		{"--find"},
		{"--callers"},
		{"--dependents"},
	}
	for _, args := range tests {
		_, err := parseQueryArgs(args)
		if err == nil {
			t.Errorf("expected error for args %v", args)
		}
	}
}

func TestParseQueryArgs_UnknownFlag(t *testing.T) {
	_, err := parseQueryArgs([]string{"--unknown", "analysis.json"})
	if err == nil {
		t.Error("expected error for unknown flag")
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

func TestParseArgs_MultiLanguage(t *testing.T) {
	// Back up os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"tukey", "-l", "php,js", "myproj"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Language != "php,js" {
		t.Errorf("expected php,js, got %s", cfg.Language)
	}
}
