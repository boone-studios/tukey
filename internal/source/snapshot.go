// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package source

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/boone-studios/tukey/internal/models"
)

func Fingerprint(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func Snapshot(files []models.FileInfo) ([]models.SourceFileMetadata, error) {
	snapshot := make([]models.SourceFileMetadata, 0, len(files))
	for _, file := range files {
		hash, err := Fingerprint(file.Path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
				continue
			}
			return nil, fmt.Errorf("hash %s: %w", file.Path, err)
		}
		snapshot = append(snapshot, models.SourceFileMetadata{
			Path:   filepath.ToSlash(file.RelativePath),
			Size:   file.Size,
			SHA256: hash,
		})
	}
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].Path < snapshot[j].Path
	})
	return snapshot, nil
}
