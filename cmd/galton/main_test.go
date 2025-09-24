package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
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
	if !strings.Contains(out, "tukey v") {
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
