package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type LayerConfig struct {
	Name string `json:"name" yaml:"name"`
	Path string `json:"path" yaml:"path"`
}

type RuleConfig struct {
	From           string   `json:"from" yaml:"from"`
	CannotDependOn []string `json:"cannot_depend_on" yaml:"cannot_depend_on"`
}

type ArchitectureConfig struct {
	Layers []LayerConfig `json:"layers" yaml:"layers"`
	Rules  []RuleConfig  `json:"rules" yaml:"rules"`
}

type FileConfig struct {
	Language     string             `json:"language" yaml:"language"`
	ExcludeDirs  []string           `json:"excludeDirs" yaml:"excludeDirs"`
	OutputFile   string             `json:"outputFile" yaml:"outputFile"`
	Verbose      bool               `json:"verbose" yaml:"verbose"`
	Benchmark    bool               `json:"benchmark" yaml:"benchmark"`
	ComparePath  string             `json:"comparePath" yaml:"comparePath"`
	Architecture ArchitectureConfig `json:"architecture" yaml:"architecture"`
}

func LoadConfig(projectRoot string) (*FileConfig, error) {
	candidates := []string{
		".tukey.yml",
		".tukey.yaml",
		".tukey.json",
	}

	for _, name := range candidates {
		path := filepath.Join(projectRoot, name)
		if _, err := os.Stat(path); err == nil {
			return parseFile(path)
		}
	}

	// no config file found, return empty
	return &FileConfig{}, nil
}

func parseFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &FileConfig{}
	switch filepath.Ext(path) {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, cfg)
	case ".json":
		err = json.Unmarshal(data, cfg)
	default:
		err = errors.New("unsupported config format")
	}
	return cfg, err
}
