// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"os"

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
	// Create the export data structure
	exportData := struct {
		Graph          *models.DependencyGraph `json:"graph"`
		TotalFiles     int                     `json:"totalFiles"`
		TotalElements  int                     `json:"totalElements"`
		ProcessingTime string                  `json:"processingTime"`
		GeneratedAt    string                  `json:"generatedAt"`
	}{
		Graph:          result.Graph,
		TotalFiles:     result.TotalFiles,
		TotalElements:  result.TotalElements,
		ProcessingTime: result.ProcessingTime,
		GeneratedAt:    "2025-09-24T18:54:12Z", // You might want to make this dynamic
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// ExportGraph exports just the dependency graph to JSON (for backwards compatibility)
func (je *JSONExporter) ExportGraph(graph *models.DependencyGraph, filename string) error {
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
