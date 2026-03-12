// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boone-studios/tukey/internal/analyzer"
	"github.com/boone-studios/tukey/internal/config"
	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/parser"
	"github.com/boone-studios/tukey/internal/progress"
	"github.com/boone-studios/tukey/internal/scanner"
	"github.com/boone-studios/tukey/pkg/output"

	_ "github.com/boone-studios/tukey/internal/lang"
)

const version = "0.3.0"

func main() {
	argv, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fileCfg, err := config.LoadConfig(argv.RootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Failed to load config file: %v\n", err)
	}

	// Merge CLI args with file config
	argv = mergeConfigs(argv, fileCfg)

	if argv.ShowVersion {
		fmt.Printf("Tukey v%s\n", version)
		os.Exit(0)
	}

	if argv.ShowHelp {
		showHelp()
		os.Exit(0)
	}

	fmt.Printf("🔍 Tukey Code Analyzer v%s\n", version)
	fmt.Printf("🎯 Analyzing codebase in: %s\n", argv.RootPath)
	fmt.Println(strings.Repeat("-", 50))

	// Initialize components
	fileScanner := scanner.NewScanner(argv.RootPath)

	p, ok := parser.Get(argv.Language)
	if !ok {
		fmt.Fprintf(os.Stderr, "❌ Unsupported language: %s\n", argv.Language)
		fmt.Fprintf(os.Stderr, "Supported: %v\n", parser.SupportedLanguages())
		os.Exit(1)
	}

	fileScanner.SetExtensions(p.FileExtensions())

	// Configure scanner exclusions
	for _, dir := range argv.ExcludeDirs {
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
	formatter.PrintSummary(result, argv.Verbose)

	// Step 5: Export if requested
	if argv.OutputFile != "" {
		exportSpinner := progress.NewSpinner(fmt.Sprintf("Exporting to %s...", argv.OutputFile))
		exportSpinner.Start()

		exporter := output.NewJSONExporter()
		if err := exporter.Export(result, argv.OutputFile); err != nil {
			exportSpinner.Stop()
			fmt.Printf("❌ Error exporting: %v\n", err)
			os.Exit(1)
		}

		exportSpinner.Stop()
		fmt.Printf("✅ Analysis exported to %s\n", argv.OutputFile)
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
	argv := &Config{
		ExcludeDirs: []string{},
	}

	args := os.Args[1:]
	if len(args) == 0 {
		argv.ShowHelp = true
		return argv, nil
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		switch arg {
		case "-v", "--verbose":
			argv.Verbose = true
		case "-h", "--help":
			argv.ShowHelp = true
			return argv, nil
		case "--version":
			argv.ShowVersion = true
			return argv, nil
		case "-o", "--output":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--output requires a filename")
			}
			argv.OutputFile = args[i+1]
			i++
		case "--exclude":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--exclude requires a directory name")
			}
			argv.ExcludeDirs = append(argv.ExcludeDirs, args[i+1])
			i++
		case "-l", "--language":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--language requires a language name")
			}
			argv.Language = strings.ToLower(args[i+1])
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			// Assume it's the root path
			argv.RootPath = arg
		}
		i++
	}

	if argv.RootPath == "" {
		return nil, fmt.Errorf("root path is required")
	}

	// Set default output file if not specified
	if argv.OutputFile == "" && argv.Verbose {
		argv.OutputFile = "tukey-results.json"
	}

	if argv.Language == "" {
		argv.Language = "php"
	}

	return argv, nil
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

CONFIGURATION:
    Tukey will automatically load settings from a config file in the project root
    if one exists. Supported file names are:

        .tukey.yml
        .tukey.yaml
        .tukey.json

    These files let you define defaults such as language, excludeDirs, verbose,
    and outputFile so you don’t need to pass flags every run.

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

// mergeConfigs merges CLI args with file config, giving CLI priority.
func mergeConfigs(argv *Config, fileCfg *config.FileConfig) *Config {
	if argv.Language == "" && fileCfg.Language != "" {
		argv.Language = fileCfg.Language
	}
	if len(fileCfg.ExcludeDirs) > 0 {
		argv.ExcludeDirs = append(argv.ExcludeDirs, fileCfg.ExcludeDirs...)
	}
	if argv.OutputFile == "" && fileCfg.OutputFile != "" {
		argv.OutputFile = fileCfg.OutputFile
	}
	if !argv.Verbose && fileCfg.Verbose {
		argv.Verbose = true
	}
	return argv
}
