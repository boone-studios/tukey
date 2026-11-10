// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package query

import (
	"fmt"
	"sort"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/scanner"
	"github.com/boone-studios/tukey/internal/source"
)

type SourceChange struct {
	Path   string `json:"path"`
	Change string `json:"change"`
}

type StatusResult struct {
	Query         string         `json:"query"`
	Status        string         `json:"status"`
	SchemaVersion int            `json:"schemaVersion"`
	GeneratedAt   string         `json:"generatedAt,omitempty"`
	RootPath      string         `json:"rootPath,omitempty"`
	CheckedFiles  int            `json:"checkedFiles"`
	ChangeCount   int            `json:"changeCount"`
	Changes       []SourceChange `json:"changes"`
	Message       string         `json:"message"`
	Page          *PageInfo      `json:"page,omitempty"`
}

// Status compares the current source tree with the snapshot stored in the analysis.
func (e *Engine) Status() (StatusResult, error) {
	result, err := e.StatusWithOptions(PageOptions{Unbounded: true})
	result.Page = nil
	return result, err
}

// StatusWithOptions returns freshness status with a bounded page of changes.
func (e *Engine) StatusWithOptions(options PageOptions) (StatusResult, error) {
	offset, limit, err := resolvePage(options)
	if err != nil {
		return StatusResult{}, err
	}
	result := StatusResult{
		Query:         "status",
		Status:        "unknown",
		SchemaVersion: e.metadata.SchemaVersion,
		GeneratedAt:   e.metadata.GeneratedAt,
		RootPath:      e.metadata.RootPath,
		Changes:       []SourceChange{},
	}
	if e.metadata.SchemaVersion == 0 || e.metadata.GeneratedAt == "" || e.metadata.RootPath == "" {
		result.Message = "Analysis has no source snapshot; rebuild it to enable freshness checks"
		result.Page = makePage(limit, offset, 0, 0, false)
		return result, nil
	}

	fileScanner := scanner.NewScanner(e.metadata.RootPath)
	fileScanner.SetExtensions(e.metadata.FileExtensions)
	for _, dir := range e.metadata.ExcludeDirs {
		fileScanner.AddExcludeDir(dir)
	}
	files, err := fileScanner.ScanFiles()
	if err != nil {
		return result, fmt.Errorf("scan current source: %w", err)
	}
	currentSnapshot, err := source.Snapshot(files)
	if err != nil {
		return result, fmt.Errorf("snapshot current source: %w", err)
	}
	result.CheckedFiles = len(currentSnapshot)

	previous := make(map[string]models.SourceFileMetadata, len(e.metadata.SourceFiles))
	for _, file := range e.metadata.SourceFiles {
		previous[file.Path] = file
	}
	current := make(map[string]models.SourceFileMetadata, len(currentSnapshot))
	for _, file := range currentSnapshot {
		current[file.Path] = file
		oldFile, exists := previous[file.Path]
		if !exists {
			result.Changes = append(result.Changes, SourceChange{Path: file.Path, Change: "added"})
		} else if oldFile.SHA256 != file.SHA256 {
			result.Changes = append(result.Changes, SourceChange{Path: file.Path, Change: "modified"})
		}
	}
	for path := range previous {
		if _, exists := current[path]; !exists {
			result.Changes = append(result.Changes, SourceChange{Path: path, Change: "deleted"})
		}
	}
	sort.Slice(result.Changes, func(i, j int) bool {
		if result.Changes[i].Path != result.Changes[j].Path {
			return result.Changes[i].Path < result.Changes[j].Path
		}
		return result.Changes[i].Change < result.Changes[j].Change
	})

	result.ChangeCount = len(result.Changes)
	if result.ChangeCount > 0 {
		result.Status = "stale"
		result.Message = "Source files changed since analysis; rebuild the graph"
	} else {
		result.Status = "fresh"
		result.Message = "Analysis matches the current source files"
	}
	total := len(result.Changes)
	var hasMore bool
	result.Changes, hasMore = pageSlice(result.Changes, offset, limit)
	result.Page = makePage(limit, offset, len(result.Changes), total, hasMore)
	return result, nil
}
