// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseInitArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *initConfig
		wantErr bool
	}{
		{
			name: "default args",
			args: []string{},
			want: &initConfig{
				rootDir:  ".",
				yes:      false,
				showHelp: false,
			},
			wantErr: false,
		},
		{
			name: "custom directory",
			args: []string{"./some-dir"},
			want: &initConfig{
				rootDir:  "./some-dir",
				yes:      false,
				showHelp: false,
			},
			wantErr: false,
		},
		{
			name: "yes flag",
			args: []string{"--yes"},
			want: &initConfig{
				rootDir:  ".",
				yes:      true,
				showHelp: false,
			},
			wantErr: false,
		},
		{
			name: "yes flag short",
			args: []string{"-y"},
			want: &initConfig{
				rootDir:  ".",
				yes:      true,
				showHelp: false,
			},
			wantErr: false,
		},
		{
			name: "help flag",
			args: []string{"-h"},
			want: &initConfig{
				rootDir:  ".",
				yes:      false,
				showHelp: true,
			},
			wantErr: false,
		},
		{
			name: "custom directory and yes flag",
			args: []string{"-y", "./my-proj"},
			want: &initConfig{
				rootDir:  "./my-proj",
				yes:      true,
				showHelp: false,
			},
			wantErr: false,
		},
		{
			name:    "unknown flag",
			args:    []string{"--invalid"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInitArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseInitArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseInitArgs() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestScanAndDetectLanguages(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"main.go":              "package main",
		"helper.go":            "package main",
		"another.go":           "package main",
		"index.php":            "<?php",
		"api.php":              "<?php",
		"app.js":               "console.log('hello')",
		"readme.md":            "# Readme",
		"notes.txt":            "some notes",
		"vendor/ignored.php":   "<?php", // should be ignored
		"node_modules/test.js": "const x = 1", // should be ignored
		".git/config":          "[core]", // should be ignored
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	langCounts, allExts, err := scanAndDetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("scanAndDetectLanguages failed: %v", err)
	}

	// Verify language counts
	if langCounts["go"] != 3 {
		t.Errorf("expected 3 go files, got %d", langCounts["go"])
	}
	if langCounts["php"] != 2 {
		t.Errorf("expected 2 php files, got %d", langCounts["php"])
	}
	if langCounts["js"] != 1 {
		t.Errorf("expected 1 js file, got %d", langCounts["js"])
	}

	// Verify extension counts and order
	// Expected sorted order:
	// .go: 3
	// .php: 2
	// .js: 1
	// .md: 1
	// .txt: 1
	// config: 1
	expectedExts := map[string]int{
		".go":    3,
		".php":   2,
		".js":    1,
		".md":    1,
		".txt":   1,
		"config": 1,
	}

	for _, extInfo := range allExts {
		expectedCount, ok := expectedExts[extInfo.ext]
		if !ok {
			t.Errorf("unexpected extension found: %s", extInfo.ext)
		}
		if extInfo.count != expectedCount {
			t.Errorf("for extension %s, expected count %d, got %d", extInfo.ext, expectedCount, extInfo.count)
		}
	}

	// Verify order is sorted descending by count
	for i := 0; i < len(allExts)-1; i++ {
		if allExts[i].count < allExts[i+1].count {
			t.Errorf("extensions not sorted descending: at index %d count %d, at index %d count %d", i, allExts[i].count, i+1, allExts[i+1].count)
		}
	}
}

func TestAppendToGitignore(t *testing.T) {
	tempDir := t.TempDir()
	gitignorePath := filepath.Join(tempDir, ".gitignore")

	// 1. Appending to non-existent gitignore does nothing (fails gracefully/silently as per code)
	err := appendToGitignore(gitignorePath, ".tukey.json")
	if err != nil {
		t.Fatalf("expected no error appending to non-existent gitignore, got %v", err)
	}

	// 2. Create gitignore and append
	initialContent := "# My Gitignore\nnode_modules/\n"
	err = os.WriteFile(gitignorePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write gitignore: %v", err)
	}

	err = appendToGitignore(gitignorePath, ".tukey.json")
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read gitignore: %v", err)
	}

	wantContent := "# My Gitignore\nnode_modules/\n.tukey.json\n"
	if string(data) != wantContent {
		t.Errorf("gitignore content = %q, want %q", string(data), wantContent)
	}

	// 3. Appending duplicate does nothing
	err = appendToGitignore(gitignorePath, ".tukey.json")
	if err != nil {
		t.Fatalf("failed to append duplicate: %v", err)
	}

	data, err = os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read gitignore: %v", err)
	}

	if string(data) != wantContent {
		t.Errorf("gitignore content changed after duplicate append: %q, want %q", string(data), wantContent)
	}
}
