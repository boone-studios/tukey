// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"os"
	"time"

	"github.com/boone-studios/tukey/internal/models"
)

// JSONExporter handles JSON export functionality
type JSONExporter struct{}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// Export exports the analysis results to a JSON file
func (je *JSONExporter) Export(result *models.AnalysisResult, filename string) error {
	metadata := result.Metadata
	if metadata.SchemaVersion == 0 {
		metadata.SchemaVersion = models.AnalysisSchemaVersion
	}
	if metadata.Generator.Name == "" {
		metadata.Generator.Name = "tukey"
	}
	if metadata.Generator.Version == "" {
		metadata.Generator.Version = "unknown"
	}
	if metadata.GeneratedAt == "" {
		metadata.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if metadata.Languages == nil {
		metadata.Languages = []string{}
	}
	if metadata.ExcludeDirs == nil {
		metadata.ExcludeDirs = []string{}
	}
	if metadata.FileExtensions == nil {
		metadata.FileExtensions = []string{}
	}
	if metadata.SourceFiles == nil {
		metadata.SourceFiles = []models.SourceFileMetadata{}
	}

	exportData := struct {
		models.AnalysisMetadata
		Graph          *models.DependencyGraph `json:"graph"`
		TotalFiles     int                     `json:"totalFiles"`
		TotalElements  int                     `json:"totalElements"`
		ProcessingTime string                  `json:"processingTime"`
		Cycles         [][]string              `json:"cycles"`
	}{
		AnalysisMetadata: metadata,
		Graph:            result.Graph,
		TotalFiles:       result.TotalFiles,
		TotalElements:    result.TotalElements,
		ProcessingTime:   result.ProcessingTime,
		Cycles:           result.Cycles,
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
