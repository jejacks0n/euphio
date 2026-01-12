package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LoadedFiles []string        `yaml:"-"` // Track all files loaded for this config
	Include     []string        `yaml:"include"`
	Debug       bool            `yaml:"debug"`
	MaxNodes    int             `yaml:"maxNodes"`
	HotReload   bool            `yaml:"hotReload"`
	General     GeneralConfig   `yaml:"general"`
	Paths       PathsConfig     `yaml:"paths"`
	Loggers     []LoggerConfig  `yaml:"loggers"`
	Listeners   ListenersConfig `yaml:"listeners"`
	Views       map[string]View `yaml:"views"`
}

type GeneralConfig struct {
	BoardName       string `yaml:"boardName"`
	PrettyBoardName string `yaml:"prettyBoardName"`
	Description     string `yaml:"description"`
	Hostname        string `yaml:"hostname"`
	Website         string `yaml:"website"`
}

type PathsConfig struct {
	Data string `yaml:"data"`
	Keys string `yaml:"keys"`
	Art  string `yaml:"art"`
}

type LoggerConfig struct {
	Stdout     bool   `yaml:"stdout,omitempty"`
	File       string `yaml:"file,omitempty"`
	Level      string `yaml:"level"`
	Source     bool   `yaml:"source"`
	HideTime   bool   `yaml:"hideTime,omitempty"`
	TimeFormat string `yaml:"timeFormat,omitempty"`
}

type ListenersConfig struct {
	Telnet TelnetConfig `yaml:"telnet"`
	SSH    SSHConfig    `yaml:"ssh"`
}

type TelnetConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Port        int    `yaml:"port"`
	InitialView string `yaml:"initialView"`
}

type SSHConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Port        int    `yaml:"port"`
	InitialView string `yaml:"initialView"`
	KeyFile     string `yaml:"keyFile"`
}

type View struct {
	Type    string                 `yaml:"type"`
	Module  string                 `yaml:"module,omitempty"` // Name of the module to use
	Art     string                 `yaml:"art,omitempty"`
	Options map[string]interface{} `yaml:"options,omitempty"`
	Actions map[string]string      `yaml:"actions,omitempty"`
	Next    *NextView              `yaml:"next,omitempty"`
}

type NextView struct {
	View  string `yaml:"view"`
	Delay int    `yaml:"delay"` // Delay in milliseconds
}

// UnmarshalYAML implements custom unmarshaling for NextView to handle both string and object formats
func (n *NextView) UnmarshalYAML(value *yaml.Node) error {
	// Case 1: next: "viewName"
	if value.Kind == yaml.ScalarNode {
		n.View = value.Value
		return nil
	}

	// Case 2: next: { view: "viewName", delay: 1000 }
	// We need a temporary struct to avoid recursion
	type plain NextView
	var tmp plain
	if err := value.Decode(&tmp); err != nil {
		return err
	}

	n.View = tmp.View
	n.Delay = tmp.Delay
	return nil
}

func Load(filename string) (*Config, error) {
	// Start with a base config
	cfg := &Config{
		LoadedFiles: []string{},
	}

	// Keep track of processed files to avoid infinite loops
	processed := make(map[string]bool)

	err := loadRecursive(filename, cfg, processed)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadRecursive(filename string, cfg *Config, processed map[string]bool) error {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	if processed[absPath] {
		return nil // Already processed
	}
	processed[absPath] = true
	cfg.LoadedFiles = append(cfg.LoadedFiles, absPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	// Expand environment variables in the YAML content
	expandedData := []byte(os.ExpandEnv(string(data)))

	// Unmarshal into a temporary struct to load includes first
	var tempCfg struct {
		Include []string `yaml:"include"`
	}
	if err := yaml.Unmarshal(expandedData, &tempCfg); err != nil {
		return err
	}

	baseDir := filepath.Dir(absPath)
	for _, includePath := range tempCfg.Include {
		// Resolve relative paths relative to the current config file
		fullPath := includePath
		if !filepath.IsAbs(includePath) {
			fullPath = filepath.Join(baseDir, includePath)
		}

		if err := loadRecursive(fullPath, cfg, processed); err != nil {
			return fmt.Errorf("failed to load included config %s: %w", fullPath, err)
		}
	}

	// Now apply the current file's configuration over the accumulated config
	if err := yaml.Unmarshal(expandedData, cfg); err != nil {
		return err
	}

	return nil
}
