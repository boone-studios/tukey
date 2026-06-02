// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/boone-studios/tukey/internal/parser"
	"golang.org/x/term"
)

type initConfig struct {
	rootDir  string
	yes      bool
	showHelp bool
}

type extCount struct {
	ext   string
	count int
}

type CheckboxOption struct {
	Label   string
	Count   int
	Checked bool
}

// runInit is the entry point for `tukey init ...`.
func runInit(args []string) {
	cfg, err := parseInitArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'tukey init --help' for usage.\n")
		os.Exit(1)
	}

	if cfg.showHelp {
		showInitHelp()
		os.Exit(0)
	}

	absPath, err := filepath.Abs(cfg.rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Check if any tukey config file already exists
	candidates := []string{
		".tukey.yml",
		".tukey.yaml",
		".tukey.json",
	}
	var existingConfig string
	for _, name := range candidates {
		path := filepath.Join(absPath, name)
		if _, err := os.Stat(path); err == nil {
			existingConfig = path
			break
		}
	}

	if existingConfig != "" {
		if cfg.yes {
			fmt.Fprintf(os.Stderr, "❌ Tukey configuration file already exists: %s\n", existingConfig)
			os.Exit(1)
		}

		// Prompt to overwrite
		fmt.Printf("⚠️  Tukey configuration file already exists at: %s\n", existingConfig)
		fmt.Print("Do you want to overwrite it? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("\nAborted.")
			os.Exit(0)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	fmt.Printf("🔍 Scanning directory: %s\n", absPath)
	langCounts, allExts, err := scanAndDetectLanguages(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error analyzing project extensions: %v\n", err)
		os.Exit(1)
	}

	// Display overall extension counts (top 5 or all if fewer)
	if len(allExts) > 0 {
		fmt.Println("\n📊 Top file extensions detected:")
		limit := 5
		if len(allExts) < limit {
			limit = len(allExts)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("   • %s: %d file(s)\n", allExts[i].ext, allExts[i].count)
		}
	} else {
		fmt.Println("\n⚠️  No files with extensions discovered (ignoring hidden files and standard excluded dirs).")
	}

	// Build options based on registered languages
	var options []CheckboxOption
	supportedLangs := parser.SupportedLanguages()
	sort.Strings(supportedLangs)

	for _, lang := range supportedLangs {
		count := langCounts[lang]
		options = append(options, CheckboxOption{
			Label:   lang,
			Count:   count,
			Checked: count > 0, // checked by default if it has files in the codebase
		})
	}

	// Sort options so that those with counts > 0 are at the top (descending by count)
	// and those with 0 files are at the bottom (alphabetical)
	sort.Slice(options, func(i, j int) bool {
		if options[i].Count != options[j].Count {
			return options[i].Count > options[j].Count
		}
		return options[i].Label < options[j].Label
	})

	var selectedLanguages []string

	// Determine selection mode
	isInteractive := !cfg.yes && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))

	if isInteractive {
		fmt.Println("\n⚙️  Select languages to configure for Tukey:")
		fmt.Println("   [Use Up/Down arrows to move, Space to toggle, Enter to confirm, Ctrl+C to cancel]")
		fmt.Println()

		selectedLanguages, err = promptCheckboxes(options)
		if err != nil {
			fmt.Printf("\n❌ Selection error: %v\n", err)
			os.Exit(1)
		}
		if len(selectedLanguages) == 0 {
			fmt.Println("⚠️  No languages selected. Defaulting to php.")
			selectedLanguages = []string{"php"}
		}
	} else {
		// Non-interactive or --yes mode.
		// Select up to the top 2 languages that have files.
		for _, opt := range options {
			if opt.Count > 0 {
				selectedLanguages = append(selectedLanguages, opt.Label)
			}
			if len(selectedLanguages) == 2 {
				break
			}
		}
		// Default to php if nothing detected
		if len(selectedLanguages) == 0 {
			selectedLanguages = []string{"php"}
		}
		fmt.Printf("\n🤖 Automatically selected languages: %s\n", strings.Join(selectedLanguages, ", "))
	}

	// Write .tukey.json config file
	configPath := filepath.Join(absPath, ".tukey.json")
	configMap := map[string]interface{}{
		"language":    strings.Join(selectedLanguages, ","),
		"excludeDirs": []string{"vendor", "node_modules", "storage", "cache", "tmp", "temp"},
		"outputFile":  "tukey-results.json",
		"verbose":     false,
	}

	configData, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error generating configuration JSON: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(configPath, configData, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error writing .tukey.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Created Tukey configuration file: %s\n", configPath)

	// Append to .gitignore if available
	gitignorePath := filepath.Join(absPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		err = appendToGitignore(gitignorePath, ".tukey.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to append .tukey.json to .gitignore: %v\n", err)
		} else {
			fmt.Println("✅ Appended .tukey.json to .gitignore")
		}

		err = appendToGitignore(gitignorePath, "tukey-results.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to append tukey-results.json to .gitignore: %v\n", err)
		} else {
			fmt.Println("✅ Appended tukey-results.json to .gitignore")
		}
	} else {
		fmt.Println("ℹ️  No .gitignore file found to append configuration to.")
	}

	fmt.Println("\n🎉 Tukey initialized successfully! Run 'tukey' to start analyzing your project.")
}

func parseInitArgs(args []string) (*initConfig, error) {
	cfg := &initConfig{
		rootDir: ".",
	}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			cfg.showHelp = true
			i++
		case "-y", "--yes":
			cfg.yes = true
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			cfg.rootDir = arg
			i++
		}
	}
	return cfg, nil
}

func showInitHelp() {
	fmt.Printf(`tukey init – initialize tukey configuration in a project

USAGE:
    tukey init [FLAGS] [directory]

FLAGS:
    -y, --yes          Non-interactive mode (automatically detect and set up top languages)
    -h, --help         Show this help message

EXAMPLES:
    tukey init
    tukey init ./my-project
    tukey init --yes
`)
}

// scanAndDetectLanguages scans directory, counts file extensions, and aggregates by Tukey languages.
func scanAndDetectLanguages(rootPath string) (map[string]int, []extCount, error) {
	excludeDirs := map[string]bool{
		"vendor":       true,
		"node_modules": true,
		".git":         true,
		".svn":         true,
		"storage":      true,
		"cache":        true,
		"tmp":          true,
		"temp":         true,
		".idea":        true,
		".vscode":      true,
	}

	extCounts := make(map[string]int)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if excludeDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != "" {
			extCounts[ext]++
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Format all extensions sorted by count
	var allExts []extCount
	for ext, count := range extCounts {
		allExts = append(allExts, extCount{ext: ext, count: count})
	}
	sort.Slice(allExts, func(i, j int) bool {
		if allExts[i].count != allExts[j].count {
			return allExts[i].count > allExts[j].count
		}
		return allExts[i].ext < allExts[j].ext
	})

	// Map extensions to registered Tukey languages
	langCounts := make(map[string]int)
	for _, lang := range parser.SupportedLanguages() {
		p, ok := parser.Get(lang)
		if !ok {
			continue
		}
		for _, parserExt := range p.FileExtensions() {
			if count, found := extCounts[strings.ToLower(parserExt)]; found {
				langCounts[lang] += count
			}
		}
	}

	return langCounts, allExts, nil
}

// promptCheckboxes displays an interactive terminal list of checkboxes
func promptCheckboxes(options []CheckboxOption) ([]string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fallbackPrompt(options)
	}
	defer term.Restore(fd, oldState)

	activeIdx := 0
	firstDraw := true

	// Hide cursor
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")

	buf := make([]byte, 16)
	for {
		if !firstDraw {
			// Move cursor up len(options) lines
			fmt.Printf("\x1b[%dA", len(options))
		}
		for i, opt := range options {
			cursor := "  "
			if i == activeIdx {
				cursor = "\x1b[36m❯ \x1b[0m"
			}
			box := "\x1b[90m[ ]\x1b[0m"
			if opt.Checked {
				box = "\x1b[32m[✓]\x1b[0m"
			}
			fmt.Printf("\x1b[2K\r%s%s \x1b[1m%s\x1b[0m \x1b[90m(%d files)\x1b[0m\n", cursor, box, opt.Label, opt.Count)
		}
		firstDraw = false

		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}

		// Handle split reads of Escape sequences:
		// If we only read 1 byte and it's Escape (27), check if more bytes are coming immediately.
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
			b := buf[0]
			switch b {
			case 3: // Ctrl+C
				return nil, fmt.Errorf("cancelled by user")
			case 13, 10: // Enter/Return
				var selected []string
				for _, opt := range options {
					if opt.Checked {
						selected = append(selected, opt.Label)
					}
				}
				return selected, nil
			case ' ': // Space
				options[activeIdx].Checked = !options[activeIdx].Checked
			case 'j', 's', 'J', 'S':
				activeIdx = (activeIdx + 1) % len(options)
			case 'k', 'w', 'K', 'W':
				activeIdx = (activeIdx - 1 + len(options)) % len(options)
			case 27: // Escape
				return nil, fmt.Errorf("cancelled by user")
			}
		} else if n >= 3 && buf[0] == 27 && (buf[1] == '[' || buf[1] == 'O') {
			switch buf[2] {
			case 'A': // Up
				activeIdx = (activeIdx - 1 + len(options)) % len(options)
			case 'B': // Down
				activeIdx = (activeIdx + 1) % len(options)
			}
		}
	}
}

func fallbackPrompt(options []CheckboxOption) ([]string, error) {
	fmt.Println("Please enter the numbers of the languages you want to enable, separated by commas (e.g. 1,2):")
	for i, opt := range options {
		box := "[ ]"
		if opt.Checked {
			box = "[✓]"
		}
		fmt.Printf("  %d) %s %s (%d files)\n", i+1, box, opt.Label, opt.Count)
	}
	fmt.Print("Selection: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		// Default to checking whichever options were originally checked
		var selected []string
		for _, opt := range options {
			if opt.Checked {
				selected = append(selected, opt.Label)
			}
		}
		return selected, nil
	}

	input = strings.TrimSpace(input)
	if input == "" {
		var selected []string
		for _, opt := range options {
			if opt.Checked {
				selected = append(selected, opt.Label)
			}
		}
		return selected, nil
	}

	var selected []string
	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var idx int
		if _, err := fmt.Sscanf(p, "%d", &idx); err == nil && idx >= 1 && idx <= len(options) {
			selected = append(selected, options[idx-1].Label)
		}
	}
	return selected, nil
}

func appendToGitignore(gitignorePath, filename string) error {
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	hasEntry := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == filename || trimmed == "/"+filename {
			hasEntry = true
			break
		}
	}

	if !hasEntry {
		content := string(data)
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += filename + "\n"
		return os.WriteFile(gitignorePath, []byte(content), 0644)
	}
	return nil
}
