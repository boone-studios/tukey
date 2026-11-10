// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package lang

import (
	"github.com/boone-studios/tukey/internal/models"
)

func failedParsedFile(file models.FileInfo, err error) *models.ParsedFile {
	return &models.ParsedFile{
		Path:          file.Path,
		Uses:          []string{},
		ImportAliases: make(map[string]string),
		Elements:      []models.CodeElement{},
		Usage:         []models.UsageElement{},
		Diagnostics: []models.AnalysisDiagnostic{
			{
				Code:     models.DiagnosticParseError,
				Severity: models.DiagnosticSeverityError,
				File:     file.Path,
				Message:  err.Error(),
				Count:    1,
			},
		},
	}
}
