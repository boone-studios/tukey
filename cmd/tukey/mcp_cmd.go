// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/boone-studios/tukey/pkg/mcp"
	"github.com/boone-studios/tukey/pkg/query"
)

// mcpConfig holds the parsed arguments for the mcp subcommand
type mcpConfig struct {
	inputFile string
	showHelp  bool
}

// runMCPServer is the entry point for `tukey mcp ...`
func runMCPServer(args []string) {
	cfg, err := parseMCPArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'tukey mcp --help' for usage.\n")
		os.Exit(1)
	}

	if cfg.showHelp {
		showMCPHelp()
		os.Exit(0)
	}

	// Load query engine with the provided analysis file
	engine, err := query.Load(cfg.inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading analysis file %s: %v\n", cfg.inputFile, err)
		fmt.Fprintf(os.Stderr, "Ensure you run static analysis first (e.g. 'tukey -o %s .') before running the MCP server.\n", cfg.inputFile)
		os.Exit(1)
	}

	// Initialize and start the MCP server
	// Using os.Stdin and os.Stdout for stdio transport channel
	server := mcp.NewServer(engine, os.Stdin, os.Stdout)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server execution error: %v\n", err)
		os.Exit(1)
	}
}

// parseMCPArgs parses args that follow `tukey mcp`
func parseMCPArgs(args []string) (*mcpConfig, error) {
	cfg := &mcpConfig{
		inputFile: "tukey-results.json",
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			cfg.showHelp = true
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			cfg.inputFile = arg
			i++
		}
	}

	return cfg, nil
}

func showMCPHelp() {
	fmt.Printf(`tukey mcp – start a native Model Context Protocol (MCP) server

USAGE:
    tukey mcp [FLAG] [analysis-file]

FLAGS:
    -h, --help            Show this help message

DESCRIPTION:
    Starts a native MCP server communicating via JSON-RPC 2.0 over standard I/O (stdin/stdout).
    This allows AI agents (e.g., in Cursor, Claude Desktop) to invoke Tukey analysis tools
    directly as part of their context/tool-use loop.

    It reads a pre-built analysis file (defaults to 'tukey-results.json') and exposes:
      - tukey_find_symbol: Locate classes, methods, functions
      - tukey_get_callers: Trace who calls a symbol
      - tukey_get_dependents: Trace what a symbol depends on
      - tukey_find_orphans: Trace dead/unused code candidates

EXAMPLES:
    tukey mcp analysis.json
    tukey mcp (uses default tukey-results.json in current directory)

`)
}
