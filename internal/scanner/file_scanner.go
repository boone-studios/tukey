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
		if !info.IsDir() && s.isPHPFile(path) {
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

// shouldExcludeDir checks if a directory should be excluded
func (s *Scanner) shouldExcludeDir(dirName string) bool {
	excluded, exists := s.excludeDirs[strings.ToLower(dirName)]
	return exists && excluded
}

// isPHPFile checks if a file is a PHP file
func (s *Scanner) isPHPFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".php" || ext == ".phtml" || ext == ".php3" || ext == ".php4" || ext == ".php5"
}

// GetStats returns scanning statistics
func (s *Scanner) GetStats() (int, map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fileCount, s.excludeDirs
}
