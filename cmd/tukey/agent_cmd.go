// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

type agentConfig struct {
	showHelp bool
	yes      bool
	global   bool
}

// RadioOption is a single item in a single-select radio prompt.
type RadioOption struct {
	Label       string
	Description string
}

func runAgent(args []string) {
	cfg, err := parseAgentArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'tukey agent --help' for usage.\n")
		os.Exit(1)
	}
	if cfg.showHelp {
		showAgentHelp()
		os.Exit(0)
	}

	fmt.Println("🤖 Tukey Agent Setup")
	fmt.Println("   Configure Tukey for use with Claude Code agents")
	fmt.Println()

	isInteractive := !cfg.yes && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))

	// ── Step 1: scope ───────────────────────────────────────────────────────
	isGlobal := cfg.global
	if isInteractive {
		fmt.Println("📍 Where should Tukey be configured?")
		fmt.Println("   [Use Up/Down arrows to move, Enter to select, Ctrl+C to cancel]")
		fmt.Println()

		scopeOptions := []RadioOption{
			{Label: "Project", Description: "this project only  (.claude/settings.json)"},
			{Label: "Global", Description: "all projects       (~/.claude/settings.json)"},
		}
		idx, err := promptRadio(scopeOptions)
		if err != nil {
			fmt.Printf("\n❌ %v\n", err)
			os.Exit(1)
		}
		isGlobal = (idx == 1)
		fmt.Println()
	} else {
		if isGlobal {
			fmt.Println("🌐 Scope: global (~/.claude/settings.json)")
		} else {
			fmt.Println("📁 Scope: project (.claude/settings.json)")
		}
	}

	// ── Step 2: derive paths ────────────────────────────────────────────────
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Could not determine executable path: %v\n", err)
		os.Exit(1)
	}
	if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
		execPath = resolved
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Could not determine working directory: %v\n", err)
		os.Exit(1)
	}

	analysisFile := detectAnalysisFilePath(cwd)

	var settingsBase string
	if isGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not determine home directory: %v\n", err)
			os.Exit(1)
		}
		settingsBase = filepath.Join(home, ".claude")
	} else {
		settingsBase = filepath.Join(cwd, ".claude")
	}

	if err := os.MkdirAll(settingsBase, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Could not create %s: %v\n", settingsBase, err)
		os.Exit(1)
	}

	// ── Step 3: write MCP config ────────────────────────────────────────────
	settingsPath := filepath.Join(settingsBase, "settings.json")
	if err := mergeAgentMCPConfig(settingsPath, execPath, analysisFile); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write MCP config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ MCP server configured in %s\n", settingsPath)

	// ── Step 4: write skill file ────────────────────────────────────────────
	skillsDir := filepath.Join(settingsBase, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Could not create skills directory: %v\n", err)
		os.Exit(1)
	}
	skillPath := filepath.Join(skillsDir, "tukey.md")
	if err := os.WriteFile(skillPath, []byte(tukeySkilContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write skill file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Skill file written to %s\n", skillPath)

	// ── Step 5: final guidance ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("🎉 Done! Next steps:")
	fmt.Printf("   1. Run 'tukey -o %s .' to build the analysis\n", filepath.Base(analysisFile))
	fmt.Println("   2. Restart Claude Code to load the new MCP server")
	if !isGlobal {
		fmt.Println("   3. Commit .claude/settings.json to share with your team")
	}
	fmt.Println()
	fmt.Println("   Available tools once the server is loaded:")
	fmt.Println("   • tukey_find_symbol        — search for a class, method, or function")
	fmt.Println("   • tukey_get_callers         — show what calls a symbol")
	fmt.Println("   • tukey_get_dependents      — show what a symbol depends on")
	fmt.Println("   • tukey_find_orphans        — list dead code candidates")
	fmt.Println("   • tukey_get_localized_context — dependency subgraph around a symbol")
}

// detectAnalysisFilePath returns the absolute path to the analysis output file,
// reading outputFile from .tukey.json if present, otherwise defaulting to
// tukey-results.json in the project root.
func detectAnalysisFilePath(root string) string {
	for _, name := range []string{".tukey.json", ".tukey.yml", ".tukey.yaml"} {
		cfgPath := filepath.Join(root, name)
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			continue
		}
		var cfg map[string]interface{}
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		if v, ok := cfg["outputFile"].(string); ok && v != "" {
			if filepath.IsAbs(v) {
				return v
			}
			return filepath.Join(root, v)
		}
	}
	return filepath.Join(root, "tukey-results.json")
}

// mergeAgentMCPConfig reads any existing settings.json, adds/overwrites the
// tukey MCP server entry, and writes it back.
func mergeAgentMCPConfig(settingsPath, execPath, analysisFile string) error {
	settings := make(map[string]interface{})

	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("existing %s contains invalid JSON: %w", settingsPath, err)
		}
	}

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}
	mcpServers["tukey"] = map[string]interface{}{
		"command": execPath,
		"args":    []string{"mcp", analysisFile},
	}
	settings["mcpServers"] = mcpServers

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0644)
}

// promptRadio shows a single-select list; returns the index of the chosen option.
func promptRadio(options []RadioOption) (int, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return promptRadioFallback(options)
	}
	defer term.Restore(fd, oldState)

	activeIdx := 0
	firstDraw := true

	fmt.Print("\x1b[?25l") // hide cursor
	defer fmt.Print("\x1b[?25h")

	buf := make([]byte, 16)
	for {
		if !firstDraw {
			fmt.Printf("\x1b[%dA", len(options))
		}
		for i, opt := range options {
			cursor := "  "
			if i == activeIdx {
				cursor = "\x1b[36m❯ \x1b[0m"
			}
			radio := "\x1b[90m( )\x1b[0m"
			if i == activeIdx {
				radio = "\x1b[32m(•)\x1b[0m"
			}
			fmt.Printf("\x1b[2K\r%s%s \x1b[1m%-10s\x1b[0m \x1b[90m%s\x1b[0m\n",
				cursor, radio, opt.Label, opt.Description)
		}
		firstDraw = false

		n, err := os.Stdin.Read(buf)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			continue
		}

		if n == 1 && buf[0] == 27 {
			syscall.SetNonblock(fd, true)
			time.Sleep(10 * time.Millisecond)
			extraBuf := make([]byte, 10)
			nExtra, errExtra := os.Stdin.Read(extraBuf)
			syscall.SetNonblock(fd, false)
			if errExtra == nil && nExtra >= 2 {
				buf[1] = extraBuf[0]
				buf[2] = extraBuf[1]
				n = 3
			}
		}

		if n == 1 {
			switch buf[0] {
			case 3:     // Ctrl+C
				return 0, fmt.Errorf("cancelled by user")
			case 13, 10: // Enter
				return activeIdx, nil
			case 27: // lone Escape
				return 0, fmt.Errorf("cancelled by user")
			case 'j', 's', 'J', 'S':
				activeIdx = (activeIdx + 1) % len(options)
			case 'k', 'w', 'K', 'W':
				activeIdx = (activeIdx - 1 + len(options)) % len(options)
			}
		} else if n >= 3 && buf[0] == 27 && (buf[1] == '[' || buf[1] == 'O') {
			switch buf[2] {
			case 'A':
				activeIdx = (activeIdx - 1 + len(options)) % len(options)
			case 'B':
				activeIdx = (activeIdx + 1) % len(options)
			}
		}
	}
}

func promptRadioFallback(options []RadioOption) (int, error) {
	fmt.Println("Select an option:")
	for i, opt := range options {
		fmt.Printf("  %d) %s — %s\n", i+1, opt.Label, opt.Description)
	}
	fmt.Print("Selection [1]: ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" || input == "1" {
		return 0, nil
	}
	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx >= 1 && idx <= len(options) {
		return idx - 1, nil
	}
	return 0, nil
}

func parseAgentArgs(args []string) (*agentConfig, error) {
	cfg := &agentConfig{}
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			cfg.showHelp = true
		case "-y", "--yes":
			cfg.yes = true
		case "-g", "--global":
			cfg.global = true
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
		}
	}
	return cfg, nil
}

func showAgentHelp() {
	fmt.Print(`tukey agent – configure Tukey for Claude Code agents

USAGE:
    tukey agent [FLAGS]

FLAGS:
    -g, --global   Install globally (~/.claude/) instead of this project (.claude/)
    -y, --yes      Non-interactive mode (defaults to project scope)
    -h, --help     Show this help message

EXAMPLES:
    tukey agent
    tukey agent --global
    tukey agent --yes
`)
}

// tukeySkilContent is written to .claude/skills/tukey.md (or the global equivalent).
const tukeySkilContent = `---
description: Analyze code dependencies, find dead code, and trace call graphs using Tukey static analysis.
---

Use the Tukey MCP tools to answer questions about this codebase's dependency structure.

**When to use Tukey:**
- Finding what calls a function, method, or class
- Identifying unused / dead code candidates
- Tracing dependency chains between modules
- Understanding the most-referenced parts of the codebase

**Available tools:**
- ` + "`tukey_find_symbol`" + ` — find classes, methods, or functions by name
- ` + "`tukey_get_callers`" + ` — show what calls a given symbol (incoming edges)
- ` + "`tukey_get_dependents`" + ` — show what a symbol depends on (outgoing edges)
- ` + "`tukey_find_orphans`" + ` — list dead code candidates (zero callers and zero dependencies)
- ` + "`tukey_get_localized_context`" + ` — get a dependency subgraph around a symbol

**If tools return empty results or the analysis seems stale**, rebuild it first:

` + "```" + `
tukey -o tukey-results.json .
` + "```" + `

Then restart Claude Code to reload the MCP server.
`
