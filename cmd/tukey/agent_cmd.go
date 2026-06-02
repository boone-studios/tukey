// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

type agentConfig struct {
	showHelp bool
	yes      bool
	global   bool
	agent    string
}

type agentConfigFormat string

const (
	agentConfigJSON agentConfigFormat = "json"
	agentConfigTOML agentConfigFormat = "toml"
)

// AgentInfo holds metadata and configuration directory paths for a target agent.
type AgentInfo struct {
	Name           string
	Key            string
	Aliases        []string
	Description    string
	GlobalDir      string // relative to user home directory
	ProjectDir     string // relative to project root
	SettingsFile   string // config file name (e.g. settings.json, mcp_config.json)
	ConfigFormat   agentConfigFormat
	NeedsSkillFile bool
	SkillFile      string // relative to the agent settings base directory
}

// ResolveGlobalDir returns the absolute path to the global settings folder for this agent.
func (a AgentInfo) ResolveGlobalDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, a.GlobalDir), nil
}

// supportedAgents is the registry of agents. To support a new agent, simply add its AgentInfo here!
var supportedAgents = []AgentInfo{
	{
		Name:           "Claude Code",
		Key:            "claude",
		Description:    "Configure for Claude Code (.claude/settings.json)",
		GlobalDir:      ".claude",
		ProjectDir:     ".claude",
		SettingsFile:   "settings.json",
		ConfigFormat:   agentConfigJSON,
		NeedsSkillFile: true,
		SkillFile:      filepath.Join("skills", "tukey.md"),
	},
	{
		Name:           "Antigravity",
		Key:            "antigravity",
		Aliases:        []string{"gemini"},
		Description:    "Configure for Antigravity (.agents/mcp_config.json)",
		GlobalDir:      filepath.Join(".gemini", "config"),
		ProjectDir:     ".agents",
		SettingsFile:   "mcp_config.json",
		ConfigFormat:   agentConfigJSON,
		NeedsSkillFile: false,
	},
	{
		Name:           "Codex",
		Key:            "codex",
		Aliases:        []string{"openai"},
		Description:    "Configure for Codex (.codex/config.toml)",
		GlobalDir:      ".codex",
		ProjectDir:     ".codex",
		SettingsFile:   "config.toml",
		ConfigFormat:   agentConfigTOML,
		NeedsSkillFile: true,
		SkillFile:      filepath.Join("skills", "tukey", "SKILL.md"),
	},
	{
		Name:           "Cursor",
		Key:            "cursor",
		Description:    "Configure for Cursor (.cursor/mcp.json)",
		GlobalDir:      ".cursor",
		ProjectDir:     ".cursor",
		SettingsFile:   "mcp.json",
		ConfigFormat:   agentConfigJSON,
		NeedsSkillFile: true,
		SkillFile:      filepath.Join("skills", "tukey", "SKILL.md"),
	},
	{
		Name:           "Grok",
		Key:            "grok",
		Aliases:        []string{"xai"},
		Description:    "Configure for Grok (.grok/config.toml)",
		GlobalDir:      ".grok",
		ProjectDir:     ".grok",
		SettingsFile:   "config.toml",
		ConfigFormat:   agentConfigTOML,
		NeedsSkillFile: true,
		SkillFile:      filepath.Join("skills", "tukey", "SKILL.md"),
	},
}

func findAgent(key string) (AgentInfo, bool) {
	k := strings.ToLower(key)
	for _, agent := range supportedAgents {
		if agent.Key == k {
			return agent, true
		}
		for _, alias := range agent.Aliases {
			if strings.ToLower(alias) == k {
				return agent, true
			}
		}
	}
	return AgentInfo{}, false
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
	fmt.Println("   Configure Tukey for use with AI agents")
	fmt.Println()

	isInteractive := !cfg.yes && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))

	// ── Step 1: select agent ──────────────────────────────────────────────────
	var selectedAgent AgentInfo
	if cfg.agent == "" {
		if isInteractive {
			fmt.Println("🤖 Select the agent to configure Tukey for:")
			fmt.Println("   [Use Up/Down arrows to move, Enter to select, Ctrl+C to cancel]")
			fmt.Println()

			var agentOptions []RadioOption
			for _, agent := range supportedAgents {
				agentOptions = append(agentOptions, RadioOption{
					Label:       agent.Name,
					Description: agent.Description,
				})
			}
			idx, err := promptRadio(agentOptions)
			if err != nil {
				fmt.Printf("\n❌ %v\n", err)
				os.Exit(1)
			}
			selectedAgent = supportedAgents[idx]
			fmt.Println()
		} else {
			// default is first one (Claude Code)
			selectedAgent = supportedAgents[0]
			fmt.Printf("🤖 Defaulting agent to: %s (%s)\n", selectedAgent.Key, selectedAgent.Name)
		}
	} else {
		var found bool
		selectedAgent, found = findAgent(cfg.agent)
		if !found {
			var names []string
			for _, agent := range supportedAgents {
				names = append(names, agent.Key)
			}
			fmt.Fprintf(os.Stderr, "❌ Unsupported agent: %s (supported: %s)\n", cfg.agent, strings.Join(names, ", "))
			os.Exit(1)
		}
	}

	// ── Step 2: scope ───────────────────────────────────────────────────────
	isGlobal := cfg.global
	if isInteractive {
		fmt.Println("📍 Where should Tukey be configured?")
		fmt.Println("   [Use Up/Down arrows to move, Enter to select, Ctrl+C to cancel]")
		fmt.Println()

		scopeOptions := []RadioOption{
			{
				Label:       "Project",
				Description: fmt.Sprintf("this project only  (%s/%s)", selectedAgent.ProjectDir, selectedAgent.SettingsFile),
			},
			{
				Label:       "Global",
				Description: fmt.Sprintf("all projects       (~/%s/%s)", selectedAgent.GlobalDir, selectedAgent.SettingsFile),
			},
		}

		idx, err := promptRadio(scopeOptions)
		if err != nil {
			fmt.Printf("\n❌ %v\n", err)
			os.Exit(1)
		}
		isGlobal = (idx == 1)
		fmt.Println()
	}

	// ── Step 3: derive paths ────────────────────────────────────────────────
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
		settingsBase, err = selectedAgent.ResolveGlobalDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not determine settings directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		settingsBase = filepath.Join(cwd, selectedAgent.ProjectDir)
	}

	settingsPath := filepath.Join(settingsBase, selectedAgent.SettingsFile)

	if !isInteractive {
		fmt.Printf("🤖 Agent: %s\n", selectedAgent.Name)
		if isGlobal {
			fmt.Printf("🌐 Scope: global (%s)\n", settingsPath)
		} else {
			fmt.Printf("📁 Scope: project (%s)\n", settingsPath)
		}
	}

	if err := os.MkdirAll(settingsBase, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Could not create %s: %v\n", settingsBase, err)
		os.Exit(1)
	}

	// ── Step 4: write MCP config ────────────────────────────────────────────
	if err := mergeAgentMCPConfig(selectedAgent, settingsPath, execPath, analysisFile); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write MCP config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ MCP server configured in %s\n", settingsPath)

	// ── Step 5: write skill file (if required by agent) ─────────────────────
	if selectedAgent.NeedsSkillFile {
		skillPath := filepath.Join(settingsBase, selectedAgent.SkillFile)
		if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not create skills directory: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(skillPath, []byte(tukeySkillContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to write skill file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Skill file written to %s\n", skillPath)
	}

	// ── Step 6: final guidance ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("🎉 Done! Next steps:")
	fmt.Printf("   1. Run 'tukey -o %s .' to build the analysis\n", filepath.Base(analysisFile))
	fmt.Printf("   2. Restart/refresh your %s client to load the new MCP server\n", selectedAgent.Name)
	if !isGlobal {
		fmt.Printf("   3. Commit %s/%s to share with your team\n", selectedAgent.ProjectDir, selectedAgent.SettingsFile)
	}
	fmt.Println()
	fmt.Println("   Available tools once the server is loaded:")
	fmt.Println("   • tukey_find_symbol          — search for a class, method, or function")
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

// mergeAgentMCPConfig reads any existing agent settings file, adds/overwrites the
// tukey MCP server entry, and writes it back in the format required by the target agent.
func mergeAgentMCPConfig(agent AgentInfo, settingsPath, execPath, analysisFile string) error {
	switch agent.ConfigFormat {
	case "", agentConfigJSON:
		return mergeJSONMCPConfig(settingsPath, execPath, analysisFile)
	case agentConfigTOML:
		return mergeTOMLConfig(settingsPath, execPath, analysisFile)
	default:
		return fmt.Errorf("unsupported config format %q", agent.ConfigFormat)
	}
}

// mergeJSONMCPConfig reads any existing settings.json or mcp_config.json, adds/overwrites the
// tukey MCP server entry, and writes it back.
func mergeJSONMCPConfig(settingsPath, execPath, analysisFile string) error {
	settings := make(map[string]interface{})

	if data, err := os.ReadFile(settingsPath); err == nil {
		trimmed := strings.TrimSpace(string(data))
		if len(trimmed) > 0 {
			if err := json.Unmarshal(data, &settings); err != nil {
				return fmt.Errorf("existing %s contains invalid JSON: %w", settingsPath, err)
			}
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

// mergeTOMLConfig updates a TOML-based MCP config (e.g. Codex .codex/config.toml or Grok .grok/config.toml)
// with a stdio MCP server entry for tukey.
func mergeTOMLConfig(settingsPath, execPath, analysisFile string) error {
	var content string
	if data, err := os.ReadFile(settingsPath); err == nil {
		content = string(data)
	}

	content = removeTOMLTable(content, "[mcp_servers.tukey]")
	content = strings.TrimRight(content, "\n")
	if content != "" {
		content += "\n\n"
	}
	content += fmt.Sprintf(`[mcp_servers.tukey]
command = %s
args = [%s, %s]
enabled = true
`, tomlString(execPath), tomlString("mcp"), tomlString(analysisFile))

	return os.WriteFile(settingsPath, []byte(content), 0644)
}

func removeTOMLTable(content, header string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	skipping := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == header {
			skipping = true
			continue
		}
		if skipping && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			skipping = false
		}
		if !skipping {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

func tomlString(value string) string {
	escaped := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	).Replace(value)
	return `"` + escaped + `"`
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
			fmt.Printf("\x1b[2K\r%s%s \x1b[1m%-12s\x1b[0m \x1b[90m%s\x1b[0m\n",
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
			setNonblock(fd, true)
			time.Sleep(10 * time.Millisecond)
			extraBuf := make([]byte, 10)
			nExtra, errExtra := os.Stdin.Read(extraBuf)
			setNonblock(fd, false)
			if errExtra == nil && nExtra >= 2 {
				buf[1] = extraBuf[0]
				buf[2] = extraBuf[1]
				n = 3
			}
		}

		if n == 1 {
			switch buf[0] {
			case 3: // Ctrl+C
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
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			cfg.showHelp = true
		case arg == "-y" || arg == "--yes":
			cfg.yes = true
		case arg == "-g" || arg == "--global":
			cfg.global = true
		case arg == "-a" || arg == "--agent":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --agent")
			}
			cfg.agent = args[i+1]
			i++
		case strings.HasPrefix(arg, "--agent="):
			cfg.agent = strings.TrimPrefix(arg, "--agent=")
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
		}
	}
	return cfg, nil
}

func showAgentHelp() {
	var names []string
	for _, agent := range supportedAgents {
		names = append(names, fmt.Sprintf("'%s'", agent.Key))
	}
	fmt.Printf(`tukey agent – configure Tukey for AI agents

USAGE:
    tukey agent [FLAGS]

FLAGS:
    -a, --agent    Specify target agent: %s (default: '%s')
    -g, --global   Install globally instead of project-level
    -y, --yes      Non-interactive mode (defaults to project scope and %s)
    -h, --help     Show this help message

EXAMPLES:
    tukey agent
    tukey agent --agent antigravity
    tukey agent --agent grok
    tukey agent --global
    tukey agent --agent antigravity --yes
    tukey agent --agent codex
    tukey agent --agent cursor
`, strings.Join(names, ", "), supportedAgents[0].Key, supportedAgents[0].Name)
}

// tukeySkillContent is written to each agent's skill path when that agent supports skills.
const tukeySkillContent = `---
name: tukey
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

Then restart or refresh your agent client to reload the MCP server.
`
