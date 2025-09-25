package scanner

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestScanFiles_Golden(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_project")
	goldenPath := filepath.Join("..", "..", "testdata", "sample_project.golden")

	s := NewScanner(root)
	// Tell the scanner which extensions to accept
	s.SetExtensions([]string{".php", ".phtml", ".php3", ".php4", ".php5"})

	files, err := s.ScanFiles()
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	var got []string
	for _, f := range files {
		got = append(got, filepath.ToSlash(f.RelativePath))
	}
	sort.Strings(got)
	gotStr := strings.Join(got, "\n") + "\n"

	if *update {
		if err := os.WriteFile(goldenPath, []byte(gotStr), 0644); err != nil {
			t.Fatalf("failed to update golden file: %v", err)
		}
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	wantStr := string(wantBytes)

	if gotStr != wantStr {
		t.Errorf("scanner output mismatch.\nGot:\n%s\nWant:\n%s", gotStr, wantStr)
	}
}
