// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/boone-studios/tukey/internal/analyzer"
	"github.com/boone-studios/tukey/internal/config"
	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/parser"
	"github.com/boone-studios/tukey/internal/progress"
	"github.com/boone-studios/tukey/internal/scanner"
	"github.com/boone-studios/tukey/pkg/compare"
	"github.com/boone-studios/tukey/pkg/guard"
	"github.com/boone-studios/tukey/pkg/output"
	"github.com/boone-studios/tukey/pkg/query"

	_ "github.com/boone-studios/tukey/internal/lang"
)

var (
	version = "0.4.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Dispatch query subcommand before any analysis work
	if len(os.Args) > 1 && os.Args[1] == "query" {
		runQuery(os.Args[2:])
		return
	}

	// Dispatch mcp subcommand.
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		runMCPServer(os.Args[2:])
		return
	}

	// Dispatch init subcommand.
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit(os.Args[2:])
		return
	}

	// Dispatch agent subcommand.
	if len(os.Args) > 1 && os.Args[1] == "agent" {
		runAgent(os.Args[2:])
		return
	}

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

	if argv.Language == "" {
		argv.Language = "php"
	}

	if argv.ShowVersion {
		displayVersion := version
		if strings.HasPrefix(displayVersion, "v") {
			displayVersion = displayVersion[1:]
		}
		if commit != "none" && date != "unknown" {
			fmt.Printf("Tukey v%s (commit: %s, date: %s)\n", displayVersion, commit, date)
		} else {
			fmt.Printf("Tukey v%s\n", displayVersion)
		}
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

	// Resolve active languages and merge file extensions
	langParts := strings.Split(argv.Language, ",")
	var activeParsers []parser.LanguageParser
	var mergedExtensions []string

	for _, part := range langParts {
		langKey := strings.TrimSpace(part)
		if langKey == "" {
			continue
		}
		p, ok := parser.Get(langKey)
		if !ok {
			fmt.Fprintf(os.Stderr, "❌ Unsupported language: %s\n", langKey)
			fmt.Fprintf(os.Stderr, "Supported: %v\n", parser.SupportedLanguages())
			os.Exit(1)
		}
		activeParsers = append(activeParsers, p)
		mergedExtensions = append(mergedExtensions, p.FileExtensions()...)
	}

	if len(activeParsers) == 0 {
		fmt.Fprintf(os.Stderr, "❌ No programming languages selected.\n")
		os.Exit(1)
	}

	fileScanner.SetExtensions(mergedExtensions)

	// Configure scanner exclusions
	for _, dir := range argv.ExcludeDirs {
		fileScanner.AddExcludeDir(dir)
	}

	// Step 1: Scan for files
	var spinner *progress.Spinner
	if !argv.Benchmark {
		spinner = progress.NewSpinner("Scanning for code files...")
		spinner.Start()
	}

	scanStart := time.Now()
	files, err := fileScanner.ScanFiles()
	if err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		fmt.Printf("❌ Error scanning files: %v\n", err)
		os.Exit(1)
	}
	scanTime := time.Since(scanStart)

	if spinner != nil {
		spinner.Stop()
	}

	if !argv.Benchmark {
		fmt.Printf("✅ Found %d files (%.2f MB total)\n",
			len(files), float64(getTotalSize(files))/(1024*1024))
	}

	// Step 2: Parse files
	var parseProgress *progress.ProgressBar
	if !argv.Benchmark {
		fmt.Printf("🔧 Parsing project files and extracting elements...\n")
		parseProgress = progress.NewProgressBar(len(files), "Parsing files")
	}

	parseStart := time.Now()

	// Group discovered files by their corresponding language parser
	filesByParser := make(map[parser.LanguageParser][]models.FileInfo)
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		var matchedParser parser.LanguageParser
		for _, p := range activeParsers {
			for _, parserExt := range p.FileExtensions() {
				if parserExt == ext {
					matchedParser = p
					break
				}
			}
			if matchedParser != nil {
				break
			}
		}

		if matchedParser != nil {
			filesByParser[matchedParser] = append(filesByParser[matchedParser], f)
		}
	}

	var parsedFiles []*models.ParsedFile
	var parseErrors []string

	// Process each parser's files
	for p, pFiles := range filesByParser {
		pParsed, err := p.ProcessFiles(pFiles, parseProgress)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s parser error: %v", p.Language(), err))
		} else {
			parsedFiles = append(parsedFiles, pParsed...)
		}
	}
	parseTime := time.Since(parseStart)

	if len(parseErrors) > 0 && !argv.Benchmark {
		fmt.Fprintf(os.Stderr, "⚠️ Parser warnings:\n - %s\n", strings.Join(parseErrors, "\n - "))
	}

	totalElements := getTotalElements(parsedFiles)
	if !argv.Benchmark {
		fmt.Printf("✅ Parsing complete! Found %d code elements in %d files\n",
			totalElements, len(parsedFiles))
	}

	// Step 3: Build dependency graph
	var dependencySpinner *progress.Spinner
	if !argv.Benchmark {
		dependencySpinner = progress.NewSpinner("Building dependency relationships...")
		dependencySpinner.Start()
	}

	graphStart := time.Now()
	tracker := analyzer.NewDependencyTracker()
	graph := tracker.BuildDependencyGraph(parsedFiles)
	graphTime := time.Since(graphStart)

	if dependencySpinner != nil {
		dependencySpinner.Stop()
	}

	processingTime := time.Since(parseStart) // Original Tukey timed from parse start to graph end
	totalTime := time.Since(scanStart)

	// Find circular dependencies (cycles)
	cycles := analyzer.FindCycles(graph)

	// Create result object
	result := &models.AnalysisResult{
		Graph:          graph,
		ParsedFiles:    parsedFiles,
		TotalFiles:     len(files),
		TotalElements:  getTotalElements(parsedFiles),
		ProcessingTime: processingTime.String(),
		Cycles:         cycles,
	}

	// Step 4: Display results
	if !argv.Benchmark {
		formatter := output.NewConsoleFormatter()
		formatter.PrintSummary(result, argv.Verbose)
	}

	// Run comparison/regression report if a baseline is specified
	if argv.ComparePath != "" {
		baselineEngine, err := query.Load(argv.ComparePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error loading baseline JSON file: %v\n", err)
			os.Exit(1)
		}
		comparisonReport := compare.CompareGraphs(baselineEngine.Graph(), graph)
		compare.PrintComparisonReport(comparisonReport, len(graph.Orphans))
	}

	// Run architectural boundary guardrails check if configured
	var violations []guard.Violation
	if len(fileCfg.Architecture.Layers) > 0 {
		violations = guard.ValidateBoundaries(graph, fileCfg.Architecture)
		if !argv.Benchmark {
			guard.PrintViolations(violations)
		}
	}

	// Step 5: Export if requested
	var exportTime time.Duration
	if argv.OutputFile != "" {
		var exportSpinner *progress.Spinner
		if !argv.Benchmark {
			exportSpinner = progress.NewSpinner(fmt.Sprintf("Exporting to %s...", argv.OutputFile))
			exportSpinner.Start()
		}

		exportStart := time.Now()
		exporter := output.NewJSONExporter()
		if err := exporter.Export(result, argv.OutputFile); err != nil {
			if exportSpinner != nil {
				exportSpinner.Stop()
			}
			fmt.Printf("❌ Error exporting: %v\n", err)
			os.Exit(1)
		}
		exportTime = time.Since(exportStart)

		if exportSpinner != nil {
			exportSpinner.Stop()
		}
		if !argv.Benchmark {
			fmt.Printf("✅ Analysis exported to %s\n", argv.OutputFile)
		}
	}

	if argv.Benchmark {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		totalSizeMB := float64(getTotalSize(files)) / (1024 * 1024)
		filesPerSec := float64(len(parsedFiles)) / parseTime.Seconds()
		mbPerSec := totalSizeMB / parseTime.Seconds()

		fmt.Println()
		fmt.Println("======================================================================")
		fmt.Println("⚡ TUKEY PERFORMANCE BENCHMARK")
		fmt.Println("======================================================================")
		fmt.Printf("📂 Codebase Metrics:\n")
		fmt.Printf("   • Total Discovered Files:  %d\n", len(files))
		fmt.Printf("   • Total Parsed Files:      %d\n", len(parsedFiles))
		fmt.Printf("   • Total Code Elements:     %d\n", totalElements)
		fmt.Printf("   • Total Codebase Size:     %.2f MB\n", totalSizeMB)
		fmt.Println()
		fmt.Printf("⏱️ Phase Durations:\n")
		fmt.Printf("   • File Scanning:           %s\n", scanTime)
		if parseTime.Seconds() > 0 {
			fmt.Printf("   • File Parsing:            %s (%.2f MB/sec, %.1f files/sec)\n", parseTime, mbPerSec, filesPerSec)
		} else {
			fmt.Printf("   • File Parsing:            %s\n", parseTime)
		}
		fmt.Printf("   • Graph Building:          %s\n", graphTime)
		if argv.OutputFile != "" {
			fmt.Printf("   • Export to JSON:          %s\n", exportTime)
		}
		fmt.Printf("   • Total Elapsed Time:      %s\n", totalTime)
		fmt.Println()
		fmt.Printf("🧠 Memory & System Stats:\n")
		fmt.Printf("   • Heap Alloc:              %.2f MB\n", float64(memStats.Alloc)/(1024*1024))
		fmt.Printf("   • Total Heap Allocated:    %.2f MB\n", float64(memStats.TotalAlloc)/(1024*1024))
		fmt.Printf("   • Virtual Memory (Sys):    %.2f MB\n", float64(memStats.Sys)/(1024*1024))
		fmt.Printf("   • Active Goroutines:       %d\n", runtime.NumGoroutine())
		fmt.Printf("   • Garbage Collections:     %d\n", memStats.NumGC)
		fmt.Println("======================================================================")
	} else {
		fmt.Printf("\n🎉 Analysis complete! Processed %d files with %d dependencies\n",
			len(files), graph.TotalEdges)
	}

	// Exit with error code if strict mode is enabled and violations exist
	if len(violations) > 0 && argv.Strict {
		fmt.Fprintf(os.Stderr, "❌ Strict mode: exiting due to %d architectural boundary violation(s)\n", len(violations))
		os.Exit(2)
	}
}

// Config holds application configuration
type Config struct {
	RootPath    string
	OutputFile  string
	Verbose     bool
	Benchmark   bool
	ComparePath string
	Strict      bool
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
		case "-b", "--benchmark":
			argv.Benchmark = true
		case "-s", "--strict":
			argv.Strict = true
		case "-c", "--compare":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--compare requires a baseline JSON file")
			}
			argv.ComparePath = args[i+1]
			i++
		case "-h", "--help":
			argv.ShowHelp = true
			return argv, nil
		case "-V", "--version":
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

	return argv, nil
}

// showHelp displays usage information
func showHelp() {
	fmt.Printf(`Tukey v%s

USAGE:
    tukey [FLAGS] <directory>       Run analysis on a codebase
    tukey query [FLAG] <file>       Query a pre-built analysis file
    tukey mcp [FLAG] [file]         Start a native Model Context Protocol (MCP) server
    tukey init [FLAGS] [directory]  Initialize tukey configuration in a project
    tukey agent [FLAGS]             Configure Tukey for use with AI coding agents

FLAGS (analysis):
    -v, --verbose           Show detailed output including function usage report
    -b, --benchmark         Run in benchmark mode (disables spinners and prints detailed performance metrics)
    -c, --compare <file>    Compare current codebase with a baseline JSON analysis file (regression detection)
    -s, --strict            Strict mode (exits with non-zero status if architectural violations are found)
    -o, --output <file>     Export results to JSON file
    --exclude <dir>         Exclude directory from analysis (can be used multiple times)
    -h, --help              Show this help message
    -l, --language          Specify the programming language to use
    -V, --version           Show version information

QUERY FLAGS:
    --find <term>           Find nodes whose name contains term (case-insensitive)
    --callers <name>        Show all nodes that call or reference the named symbol
    --dependents <name>     Show all nodes that the named symbol depends on
    --orphans               List all orphaned nodes (dead code candidates)

MCP FLAGS:
    -h, --help              Show help message for MCP subcommand

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
    tukey -b ./my-project
    tukey --compare baseline.json ./my-project
    tukey --strict ./my-project
    tukey -v ./my-project -o analysis.json
    tukey --exclude vendor --exclude tests ./my-project
    tukey query --find "GatewayFactory" analysis.json
    tukey query --callers "makeApiRequest" analysis.json
    tukey query --dependents "PaymentService" analysis.json
    tukey query --orphans analysis.json
    tukey mcp analysis.json
    tukey init ./my-project
    tukey agent
    tukey agent --global

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

// mergeConfigs merges CLI args with file config, giving CLI priority
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
	if !argv.Benchmark && fileCfg.Benchmark {
		argv.Benchmark = true
	}
	if argv.ComparePath == "" && fileCfg.ComparePath != "" {
		argv.ComparePath = fileCfg.ComparePath
	}
	return argv
}
