// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

//go:build !testcover

package parser

import (
	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/progress"
)

// LanguageParser is the contract any language parser must satisfy
type LanguageParser interface {
	ProcessFiles(files []models.FileInfo, progressBar *progress.ProgressBar) ([]*models.ParsedFile, error)
	Language() string // e.g., "php", "go", etc.
	FileExtensions() []string
}
