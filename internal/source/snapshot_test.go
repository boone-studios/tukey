// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package source

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func TestSnapshotSkipsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	presentPath := filepath.Join(dir, "present.php")
	if err := os.WriteFile(presentPath, []byte("<?php\n"), 0644); err != nil {
		t.Fatal(err)
	}

	snapshot, err := Snapshot([]models.FileInfo{
		{Path: filepath.Join(dir, "missing.php"), RelativePath: "missing.php"},
		{Path: presentPath, RelativePath: "present.php", Size: 6},
	})
	if err != nil {
		t.Fatalf("Snapshot should skip missing files: %v", err)
	}
	if len(snapshot) != 1 || snapshot[0].Path != "present.php" || snapshot[0].SHA256 == "" {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}
