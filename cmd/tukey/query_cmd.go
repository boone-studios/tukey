// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/boone-studios/tukey/internal/config"
	"github.com/boone-studios/tukey/pkg/query"
)

// queryConfig holds the parsed arguments for the query subcommand.
type queryConfig struct {
	mode      string // "find" | "callers" | "dependents" | "orphans"
	term      string
	inputFile string
}

// runQuery is the entry point for `tukey query ...`.
func runQuery(args []string) {
	cfg, err := parseQueryArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'tukey query --help' for usage.\n")
		os.Exit(1)
	}

	if cfg == nil {
		// --help was requested
		showQueryHelp()
		os.Exit(0)
	}

	engine, err := query.Load(cfg.inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var result query.QueryResult
	switch cfg.mode {
	case "find":
		result = engine.Find(cfg.term)
	case "callers":
		result = engine.Callers(cfg.term)
	case "dependents":
		result = engine.Dependents(cfg.term)
	case "orphans":
		result = engine.Orphans()
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding result: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// parseQueryArgs parses args that follow `tukey query`.
// Returns nil cfg (and no error) when --help is requested.
func parseQueryArgs(args []string) (*queryConfig, error) {
	cfg := &queryConfig{}

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			return nil, nil
		case "--find":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--find requires a search term")
			}
			cfg.mode = "find"
			cfg.term = args[i+1]
			i += 2
		case "--callers":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--callers requires a symbol name")
			}
			cfg.mode = "callers"
			cfg.term = args[i+1]
			i += 2
		case "--dependents":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--dependents requires a symbol name")
			}
			cfg.mode = "dependents"
			cfg.term = args[i+1]
			i += 2
		case "--orphans":
			cfg.mode = "orphans"
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			cfg.inputFile = arg
			i++
		}
	}

	if cfg.mode == "" {
		return nil, fmt.Errorf("query requires one of: --find TERM, --callers TERM, --dependents TERM, --orphans")
	}
	if cfg.inputFile == "" {
		// Try to find a default analysis file
		// 1. Look in the local directory config (.tukey.json etc.)
		if fileCfg, err := config.LoadConfig("."); err == nil && fileCfg.OutputFile != "" {
			if _, err := os.Stat(fileCfg.OutputFile); err == nil {
				cfg.inputFile = fileCfg.OutputFile
			}
		}
		// 2. Fall back to "tukey-results.json" if it exists
		if cfg.inputFile == "" {
			if _, err := os.Stat("tukey-results.json"); err == nil {
				cfg.inputFile = "tukey-results.json"
			}
		}
		// 3. If still empty, return error
		if cfg.inputFile == "" {
			return nil, fmt.Errorf("query requires an analysis file path (e.g. tukey-results.json)")
		}
	}
	return cfg, nil
}

func showQueryHelp() {
	fmt.Printf(`tukey query – query a pre-built analysis file

USAGE:
    tukey query [FLAG] <analysis-file>

FLAGS:
    --find <term>         Find nodes whose name contains term (case-insensitive)
    --callers <name>      Show all nodes that call or reference the named symbol
    --dependents <name>   Show all nodes that the named symbol depends on
    --orphans             List all orphaned nodes (no dependencies and no dependents)
    -h, --help            Show this help message

EXAMPLES:
    tukey query --find "GatewayFactory" analysis.json
    tukey query --callers "makeApiRequest" analysis.json
    tukey query --dependents "PaymentService" analysis.json
    tukey query --orphans analysis.json

OUTPUT:
    All queries emit a JSON object to stdout:
    {
      "query":   "<mode>",
      "term":    "<search term>",      // omitted for --orphans
      "count":   <int>,
      "results": [ ... ]
    }

    Each result includes: id, name, type, file, line, score,
    dependencyCount, dependentCount. Callers and --dependents results
    also include refType, refLines, and refCount.

`)
}
