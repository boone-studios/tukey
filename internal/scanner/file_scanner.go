// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/boone-studios/tukey/internal/models"
)

// Scanner handles file discovery and filtering
type Scanner struct {
	rootPath    string
	excludeDirs map[string]bool
	fileCount   int
	extensions  map[string]bool
	mu          sync.Mutex
}

// NewScanner creates a new file scanner instance
func NewScanner(rootPath string) *Scanner {
	// Common directories to exclude from scanning
	excludeDirs := map[string]bool{
		"vendor":       true,
		"node_modules": true,
		".git":         true,
		".svn":         true,
		"storage":      true,
		"cache":        true,
		"tmp":          true,
		"temp":         true,
		".idea":        true,
		".vscode":      true,
		"bootstrap":    true,  // Laravel bootstrap cache
		"public":       false, // Keep public, might have PHP files
	}

	return &Scanner{
		rootPath:    rootPath,
		excludeDirs: excludeDirs,
		extensions:  make(map[string]bool),
	}
}

// AddExcludeDir adds a directory to the exclusion list
func (s *Scanner) AddExcludeDir(dir string) {
	s.excludeDirs[dir] = true
}

// ScanFiles discovers all PHP files in the codebase
func (s *Scanner) ScanFiles() ([]models.FileInfo, error) {
	var files []models.FileInfo
	var mu sync.Mutex

	err := filepath.Walk(s.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory we want to exclude
		if info.IsDir() && s.shouldExcludeDir(info.Name()) {
			return filepath.SkipDir
		}

		// Only process PHP files
		// todo: add support for other file types
		if !info.IsDir() && s.hasAllowedExtension(path) {
			relativePath, _ := filepath.Rel(s.rootPath, path)

			fileData := models.FileInfo{
				Path:         path,
				RelativePath: relativePath,
				Size:         info.Size(),
			}

			mu.Lock()
			files = append(files, fileData)
			s.fileCount++
			mu.Unlock()
		}

		return nil
	})

	return files, err
}

// SetExtensions configures which file extensions to include
func (s *Scanner) SetExtensions(exts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.extensions = make(map[string]bool)
	for _, e := range exts {
		s.extensions[strings.ToLower(e)] = true
	}
}

// shouldExcludeDir checks if a directory should be excluded
func (s *Scanner) shouldExcludeDir(dirName string) bool {
	excluded, exists := s.excludeDirs[strings.ToLower(dirName)]
	return exists && excluded
}

// GetStats returns scanning statistics
func (s *Scanner) GetStats() (int, map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fileCount, s.excludeDirs
}

// hasAllowedExtension checks if the extension is expected of the set language
func (s *Scanner) hasAllowedExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(s.extensions) == 0 {
		return false
	}
	return s.extensions[ext]
}
