package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/boone-studios/tukey/internal/config"
)

func captureOutput(f func()) string {
	// Save original stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function
	f()

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestShowHelp_OutputContainsUsageAndFlags(t *testing.T) {
	out := captureOutput(showHelp)

	if !strings.Contains(out, "USAGE:") {
		t.Errorf("help output missing USAGE section:\n%s", out)
	}
	if !strings.Contains(out, "FLAGS:") {
		t.Errorf("help output missing FLAGS section:\n%s", out)
	}
	if !strings.Contains(out, "Tukey v") {
		t.Errorf("help output missing version string:\n%s", out)
	}
}

func TestParseArgs_VerboseAndOutput(t *testing.T) {
	os.Args = []string{"tukey", "-v", "-o", "out.json", "myproj"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Verbose {
		t.Errorf("expected verbose")
	}
	if cfg.OutputFile != "out.json" {
		t.Errorf("expected out.json, got %s", cfg.OutputFile)
	}
	if cfg.RootPath != "myproj" {
		t.Errorf("expected root path myproj, got %s", cfg.RootPath)
	}
}

func TestParseArgs_ExcludeDirs(t *testing.T) {
	os.Args = []string{"tukey", "--exclude", "vendor", "--exclude", "tests", "myproj"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"vendor", "tests"}
	if !reflect.DeepEqual(cfg.ExcludeDirs, want) {
		t.Errorf("expected %v, got %v", want, cfg.ExcludeDirs)
	}
}

func TestParseArgs_Errors(t *testing.T) {
	tests := [][]string{
		{"tukey", "--output"},  // missing filename
		{"tukey", "--exclude"}, // missing dir
		{"tukey", "-x"},        // unknown flag
	}
	for _, args := range tests {
		os.Args = args
		_, err := parseArgs()
		if err == nil {
			t.Errorf("expected error for args %v", args)
		}
	}
}

func TestParseArgs_NoArgsShowsHelp(t *testing.T) {
	os.Args = []string{"tukey"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}
	if !cfg.ShowHelp {
		t.Errorf("expected ShowHelp to be true when no args")
	}
}

func TestMergeConfigs_FileProvidesDefaults(t *testing.T) {
	argv := &Config{
		RootPath: "myproj",
		// nothing else set
	}
	fileCfg := &config.FileConfig{
		Language:    "php",
		ExcludeDirs: []string{"vendor", "tests"},
		OutputFile:  "report.json",
		Verbose:     true,
	}

	merged := mergeConfigs(argv, fileCfg)

	if merged.Language != "php" {
		t.Errorf("expected language php, got %s", merged.Language)
	}
	if merged.OutputFile != "report.json" {
		t.Errorf("expected report.json, got %s", merged.OutputFile)
	}
	if !merged.Verbose {
		t.Errorf("expected verbose = true")
	}
	if len(merged.ExcludeDirs) != 2 {
		t.Errorf("expected 2 excludeDirs, got %d", len(merged.ExcludeDirs))
	}
}

func TestParseQueryArgs_Find(t *testing.T) {
	cfg, err := parseQueryArgs([]string{"--find", "GatewayFactory", "analysis.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.mode != "find" {
		t.Errorf("mode: got %q want find", cfg.mode)
	}
	if cfg.term != "GatewayFactory" {
		t.Errorf("term: got %q want GatewayFactory", cfg.term)
	}
	if cfg.inputFile != "analysis.json" {
		t.Errorf("inputFile: got %q want analysis.json", cfg.inputFile)
	}
}

func TestParseQueryArgs_Orphans(t *testing.T) {
	cfg, err := parseQueryArgs([]string{"--orphans", "analysis.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.mode != "orphans" {
		t.Errorf("mode: got %q want orphans", cfg.mode)
	}
	if cfg.term != "" {
		t.Errorf("term should be empty for --orphans")
	}
}

func TestParseQueryArgs_Help(t *testing.T) {
	cfg, err := parseQueryArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil cfg for --help")
	}
}

func TestParseQueryArgs_MissingMode(t *testing.T) {
	_, err := parseQueryArgs([]string{"analysis.json"})
	if err == nil {
		t.Error("expected error when no mode flag given")
	}
}

func TestParseQueryArgs_MissingFile(t *testing.T) {
	_, err := parseQueryArgs([]string{"--find", "Foo"})
	if err == nil {
		t.Error("expected error when no input file given")
	}
}

func TestParseQueryArgs_DefaultFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tukey-query-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change working dir: %v", err)
	}

	// Case 1: No file exists, should error
	_, err = parseQueryArgs([]string{"--find", "Foo"})
	if err == nil {
		t.Error("expected error when no default file exists")
	}

	// Case 2: "tukey-results.json" exists, should default to it
	dummyFile := "tukey-results.json"
	if err := os.WriteFile(dummyFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	cfg, err := parseQueryArgs([]string{"--find", "Foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.inputFile != "tukey-results.json" {
		t.Errorf("expected default to be tukey-results.json, got %s", cfg.inputFile)
	}
}

func TestParseQueryArgs_MissingTerm(t *testing.T) {
	tests := [][]string{
		{"--find"},
		{"--callers"},
		{"--dependents"},
	}
	for _, args := range tests {
		_, err := parseQueryArgs(args)
		if err == nil {
			t.Errorf("expected error for args %v", args)
		}
	}
}

func TestParseQueryArgs_UnknownFlag(t *testing.T) {
	_, err := parseQueryArgs([]string{"--unknown", "analysis.json"})
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestMergeConfigs_CLIOverridesFile(t *testing.T) {
	argv := &Config{
		RootPath:    "myproj",
		Language:    "go",
		OutputFile:  "cli.json",
		Verbose:     true,
		ExcludeDirs: []string{"cli-only"},
	}
	fileCfg := &config.FileConfig{
		Language:    "php",
		ExcludeDirs: []string{"vendor"},
		OutputFile:  "file.json",
		Verbose:     false,
	}

	merged := mergeConfigs(argv, fileCfg)

	if merged.Language != "go" { // CLI wins
		t.Errorf("expected go, got %s", merged.Language)
	}
	if merged.OutputFile != "cli.json" {
		t.Errorf("expected cli.json, got %s", merged.OutputFile)
	}
	if !merged.Verbose {
		t.Errorf("expected verbose = true from CLI")
	}
	if len(merged.ExcludeDirs) != 2 {
		t.Errorf("expected merged excludeDirs length 2, got %d", len(merged.ExcludeDirs))
	}
}

func TestParseArgs_MultiLanguage(t *testing.T) {
	// Back up os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"tukey", "-l", "php,js", "myproj"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Language != "php,js" {
		t.Errorf("expected php,js, got %s", cfg.Language)
	}
}

func TestParseArgs_Benchmark(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"tukey", "-b", "myproj"}
	cfg, err := parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Benchmark {
		t.Errorf("expected benchmark mode to be active via -b")
	}

	os.Args = []string{"tukey", "--benchmark", "myproj"}
	cfg, err = parseArgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Benchmark {
		t.Errorf("expected benchmark mode to be active via --benchmark")
	}
}

func TestMergeConfigs_Benchmark(t *testing.T) {
	argv := &Config{
		RootPath: "myproj",
	}
	fileCfg := &config.FileConfig{
		Benchmark: true,
	}

	merged := mergeConfigs(argv, fileCfg)
	if !merged.Benchmark {
		t.Errorf("expected benchmark to merge as true")
	}
}

func TestLanguageResolution(t *testing.T) {
	// Scenario 1: CLI language specified, config language also specified. CLI should win.
	argv1 := &Config{Language: "go"}
	fileCfg1 := &config.FileConfig{Language: "js"}
	merged1 := mergeConfigs(argv1, fileCfg1)
	if merged1.Language != "go" {
		t.Errorf("expected language to be go, got %s", merged1.Language)
	}

	// Scenario 2: CLI language empty, config language specified. Config should win.
	argv2 := &Config{Language: ""}
	fileCfg2 := &config.FileConfig{Language: "go"}
	merged2 := mergeConfigs(argv2, fileCfg2)
	if merged2.Language != "go" {
		t.Errorf("expected language to be go, got %s", merged2.Language)
	}

	// Scenario 3: Both empty. Default should resolve to php.
	argv3 := &Config{Language: ""}
	fileCfg3 := &config.FileConfig{Language: ""}
	merged3 := mergeConfigs(argv3, fileCfg3)
	lang3 := merged3.Language
	if lang3 == "" {
		lang3 = "php"
	}
	if lang3 != "php" {
		t.Errorf("expected language to default to php, got %s", lang3)
	}
}

func TestParseAgentArgs(t *testing.T) {
	tests := []struct {
		args    []string
		want    *agentConfig
		wantErr bool
	}{
		{
			args: []string{"-h"},
			want: &agentConfig{showHelp: true},
		},
		{
			args: []string{"--help"},
			want: &agentConfig{showHelp: true},
		},
		{
			args: []string{"-y", "-g"},
			want: &agentConfig{yes: true, global: true},
		},
		{
			args: []string{"--agent", "antigravity"},
			want: &agentConfig{agent: "antigravity"},
		},
		{
			args: []string{"--agent=claude"},
			want: &agentConfig{agent: "claude"},
		},
		{
			args: []string{"--agent", "codex"},
			want: &agentConfig{agent: "codex"},
		},
		{
			args: []string{"--agent", "cursor"},
			want: &agentConfig{agent: "cursor"},
		},
		{
			args: []string{"--agent", "grok"},
			want: &agentConfig{agent: "grok"},
		},
		{
			args:    []string{"--agent"},
			wantErr: true,
		},
		{
			args:    []string{"--unknown-flag"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := parseAgentArgs(tt.args)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseAgentArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if got.showHelp != tt.want.showHelp || got.yes != tt.want.yes || got.global != tt.want.global || got.agent != tt.want.agent {
			t.Errorf("parseAgentArgs(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestFindAgent_CodexAliases(t *testing.T) {
	for _, key := range []string{"codex", "CODEX", "openai"} {
		agent, ok := findAgent(key)
		if !ok {
			t.Fatalf("findAgent(%q) did not find Codex", key)
		}
		if agent.Key != "codex" {
			t.Fatalf("findAgent(%q) = %q, want codex", key, agent.Key)
		}
		if agent.ConfigFormat != agentConfigTOML {
			t.Fatalf("Codex config format = %q, want %q", agent.ConfigFormat, agentConfigTOML)
		}
		if agent.SkillFile != "skills/tukey/SKILL.md" {
			t.Fatalf("Codex skill file = %q, want skills/tukey/SKILL.md", agent.SkillFile)
		}
	}
}

func TestFindAgent_Cursor(t *testing.T) {
	agent, ok := findAgent("cursor")
	if !ok {
		t.Fatal("findAgent(\"cursor\") did not find Cursor")
	}
	if agent.Key != "cursor" {
		t.Fatalf("findAgent(\"cursor\") = %q, want cursor", agent.Key)
	}
	if agent.ConfigFormat != agentConfigJSON {
		t.Fatalf("Cursor config format = %q, want %q", agent.ConfigFormat, agentConfigJSON)
	}
	if agent.SkillFile != "skills/tukey/SKILL.md" {
		t.Fatalf("Cursor skill file = %q, want skills/tukey/SKILL.md", agent.SkillFile)
	}
}

func TestFindAgent_Grok(t *testing.T) {
	for _, key := range []string{"grok", "GROK", "xai"} {
		agent, ok := findAgent(key)
		if !ok {
			t.Fatalf("findAgent(%q) did not find Grok", key)
		}
		if agent.Key != "grok" {
			t.Fatalf("findAgent(%q) = %q, want grok", key, agent.Key)
		}
		if agent.ConfigFormat != agentConfigTOML {
			t.Fatalf("Grok config format = %q, want %q", agent.ConfigFormat, agentConfigTOML)
		}
		if agent.SkillFile != "skills/tukey/SKILL.md" {
			t.Fatalf("Grok skill file = %q, want skills/tukey/SKILL.md", agent.SkillFile)
		}
	}
}

func TestMergeJSONMCPConfig(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/mcp.json"
	existing := `{
  "mcpServers": {
    "other-server": {
      "command": "npx",
      "args": ["-y", "some-mcp-server"]
    },
    "tukey": {
      "command": "/old/tukey",
      "args": ["mcp", "/old/results.json"]
    }
  }
}
`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	agent, ok := findAgent("cursor")
	if !ok {
		t.Fatal("Cursor agent not registered")
	}
	if err := mergeAgentMCPConfig(agent, path, `/Applications/Tukey "Dev"/tukey`, `/tmp/project/tukey-results.json`); err != nil {
		t.Fatalf("mergeAgentMCPConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`"other-server"`,
		`"npx"`,
		`"tukey"`,
		`/Applications/Tukey \"Dev\"/tukey`,
		`"/tmp/project/tukey-results.json"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("merged config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "/old/tukey") {
		t.Fatalf("merged config retained old tukey command:\n%s", got)
	}
}

func TestMergeTOMLConfig(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.toml"
	existing := `[projects."/tmp/project"]
trust_level = "trusted"

[mcp_servers.tukey]
command = "/old/tukey"
args = ["mcp", "/old/results.json"]
enabled = true

[plugins."browser-use@openai-bundled"]
enabled = true
`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	agent, ok := findAgent("codex")
	if !ok {
		t.Fatal("Codex agent not registered")
	}
	if err := mergeAgentMCPConfig(agent, path, `/Applications/Tukey "Dev"/tukey`, `/tmp/project/tukey-results.json`); err != nil {
		t.Fatalf("mergeAgentMCPConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`[projects."/tmp/project"]`,
		`trust_level = "trusted"`,
		`[plugins."browser-use@openai-bundled"]`,
		`[mcp_servers.tukey]`,
		`command = "/Applications/Tukey \"Dev\"/tukey"`,
		`args = ["mcp", "/tmp/project/tukey-results.json"]`,
		`enabled = true`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("merged config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "/old/tukey") {
		t.Fatalf("merged config retained old tukey block:\n%s", got)
	}
	if strings.Count(got, "[mcp_servers.tukey]") != 1 {
		t.Fatalf("merged config should contain one tukey block:\n%s", got)
	}

	// Verify Grok (also TOML format) merges correctly into a .grok/config.toml style file.
	grokAgent, ok := findAgent("grok")
	if !ok {
		t.Fatal("Grok agent not registered")
	}
	grokPath := dir + "/grok-config.toml"
	if err := os.WriteFile(grokPath, []byte("[ui]\nmax_thoughts_width = 120\n"), 0644); err != nil {
		t.Fatalf("failed to write grok config: %v", err)
	}
	if err := mergeAgentMCPConfig(grokAgent, grokPath, "/usr/local/bin/tukey", "/tmp/project/grok-results.json"); err != nil {
		t.Fatalf("mergeAgentMCPConfig for grok failed: %v", err)
	}
	grokData, err := os.ReadFile(grokPath)
	if err != nil {
		t.Fatalf("failed to read grok config: %v", err)
	}
	grokGot := string(grokData)
	for _, want := range []string{
		`[ui]`,
		`max_thoughts_width = 120`,
		`[mcp_servers.tukey]`,
		`command = "/usr/local/bin/tukey"`,
		`args = ["mcp", "/tmp/project/grok-results.json"]`,
		`enabled = true`,
	} {
		if !strings.Contains(grokGot, want) {
			t.Fatalf("grok merged config missing %q:\n%s", want, grokGot)
		}
	}
	if strings.Count(grokGot, "[mcp_servers.tukey]") != 1 {
		t.Fatalf("grok merged config should contain one tukey block:\n%s", grokGot)
	}
}
