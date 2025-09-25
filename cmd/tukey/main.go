// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boone-studios/tukey/internal/analyzer"
	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/parser"
	"github.com/boone-studios/tukey/internal/progress"
	"github.com/boone-studios/tukey/internal/scanner"
	"github.com/boone-studios/tukey/pkg/output"

	_ "github.com/boone-studios/tukey/internal/lang"
)

const version = "0.2.0"

func main() {
	config, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if config.ShowVersion {
		fmt.Printf("Tukey v%s\n", version)
		os.Exit(0)
	}

	if config.ShowHelp {
		showHelp()
		os.Exit(0)
	}

	fmt.Printf("🔍 Tukey Code Analyzer v%s\n", version)
	fmt.Printf("🎯 Analyzing codebase in: %s\n", config.RootPath)
	fmt.Println(strings.Repeat("-", 50))

	// Initialize components
	fileScanner := scanner.NewScanner(config.RootPath)

	p, ok := parser.Get(config.Language)
	if !ok {
		fmt.Fprintf(os.Stderr, "❌ Unsupported language: %s\n", config.Language)
		fmt.Fprintf(os.Stderr, "Supported: %v\n", parser.SupportedLanguages())
		os.Exit(1)
	}

	fileScanner.SetExtensions(p.FileExtensions())

	// Configure scanner exclusions
	for _, dir := range config.ExcludeDirs {
		fileScanner.AddExcludeDir(dir)
	}

	// Step 1: Scan for files
	spinner := progress.NewSpinner("Scanning for code files...")
	spinner.Start()

	files, err := fileScanner.ScanFiles()
	if err != nil {
		spinner.Stop()
		fmt.Printf("❌ Error scanning files: %v\n", err)
		os.Exit(1)
	}

	spinner.Stop()
	fmt.Printf("✅ Found %d files (%.2f MB total)\n",
		len(files), float64(getTotalSize(files))/(1024*1024))

	// Step 2: Parse files
	fmt.Printf("🔧 Parsing project files and extracting elements...\n")
	parseProgress := progress.NewProgressBar(len(files), "Parsing files")

	startTime := time.Now()
	parsedFiles, err := p.ProcessFiles(files, parseProgress)
	if err != nil {
		fmt.Printf("❌ Error parsing files: %v\n", err)
		os.Exit(1)
	}

	totalElements := getTotalElements(parsedFiles)
	fmt.Printf("✅ Parsing complete! Found %d code elements in %d files\n",
		totalElements, len(parsedFiles))

	// Step 3: Build dependency graph
	dependencySpinner := progress.NewSpinner("Building dependency relationships...")
	dependencySpinner.Start()

	tracker := analyzer.NewDependencyTracker()
	graph := tracker.BuildDependencyGraph(parsedFiles)

	dependencySpinner.Stop()

	processingTime := time.Since(startTime)

	// Create result object
	result := &models.AnalysisResult{
		Graph:          graph,
		ParsedFiles:    parsedFiles,
		TotalFiles:     len(files),
		TotalElements:  getTotalElements(parsedFiles),
		ProcessingTime: processingTime.String(),
	}

	// Step 4: Display results
	formatter := output.NewConsoleFormatter()
	formatter.PrintSummary(result, config.Verbose)

	// Step 5: Export if requested
	if config.OutputFile != "" {
		exportSpinner := progress.NewSpinner(fmt.Sprintf("Exporting to %s...", config.OutputFile))
		exportSpinner.Start()

		exporter := output.NewJSONExporter()
		if err := exporter.Export(result, config.OutputFile); err != nil {
			exportSpinner.Stop()
			fmt.Printf("❌ Error exporting: %v\n", err)
			os.Exit(1)
		}

		exportSpinner.Stop()
		fmt.Printf("✅ Analysis exported to %s\n", config.OutputFile)
	}

	fmt.Printf("\n🎉 Analysis complete! Processed %d files with %d dependencies\n",
		len(files), graph.TotalEdges)
}

// Config holds application configuration
type Config struct {
	RootPath    string
	OutputFile  string
	Verbose     bool
	ShowHelp    bool
	ShowVersion bool
	ExcludeDirs []string
	Language    string
}

// parseArgs parses command line arguments
func parseArgs() (*Config, error) {
	config := &Config{
		ExcludeDirs: []string{},
	}

	args := os.Args[1:]
	if len(args) == 0 {
		config.ShowHelp = true
		return config, nil
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		switch arg {
		case "-v", "--verbose":
			config.Verbose = true
		case "-h", "--help":
			config.ShowHelp = true
			return config, nil
		case "--version":
			config.ShowVersion = true
			return config, nil
		case "-o", "--output":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--output requires a filename")
			}
			config.OutputFile = args[i+1]
			i++
		case "--exclude":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--exclude requires a directory name")
			}
			config.ExcludeDirs = append(config.ExcludeDirs, args[i+1])
			i++
		case "-l", "--language":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--language requires a language name")
			}
			config.Language = strings.ToLower(args[i+1])
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			// Assume it's the root path
			config.RootPath = arg
		}
		i++
	}

	if config.RootPath == "" {
		return nil, fmt.Errorf("root path is required")
	}

	// Set default output file if not specified
	if config.OutputFile == "" && config.Verbose {
		config.OutputFile = "tukey-results.json"
	}

	if config.Language == "" {
		config.Language = "php"
	}

	return config, nil
}

// showHelp displays usage information
func showHelp() {
	fmt.Printf(`Tukey v%s

USAGE:
    Tukey [FLAGS] <directory>

FLAGS:
    -v, --verbose           Show detailed output including function usage report
    -o, --output <file>     Export results to JSON file
    --exclude <dir>         Exclude directory from analysis (can be used multiple times)
    -h, --help              Show this help message
    -l, --language    	    Specify the programming language to use
    --version               Show version information

EXAMPLES:
    tukey ./my-project
    tukey -v ./my-project -o analysis.json
    tukey --exclude vendor --exclude tests ./my-project

`, version)
}

// getTotalSize calculates total size of files
func getTotalSize(files []models.FileInfo) int64 {
	var total int64
	for _, file := range files {
		total += file.Size
	}
	return total
}

// getTotalElements counts total elements in parsed files
func getTotalElements(parsedFiles []*models.ParsedFile) int {
	total := 0
	for _, file := range parsedFiles {
		total += len(file.Elements)
	}
	return total
}
