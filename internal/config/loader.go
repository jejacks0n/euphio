package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Include      []string           `yaml:"include"`
	Debug        bool               `yaml:"debug"`
	General      GeneralConfig      `yaml:"general"`
	Paths        PathsConfig        `yaml:"paths"`
	Loggers      []LoggerConfig     `yaml:"loggers"`
	LoginServers LoginServersConfig `yaml:"loginServers"`
}

type GeneralConfig struct {
	BoardName       string `yaml:"boardName"`
	PrettyBoardName string `yaml:"prettyBoardName"`
	Description     string `yaml:"description"`
	Hostname        string `yaml:"hostname"`
	Website         string `yaml:"website"`
	MaxNodes        int    `yaml:"maxNodes"`
	HotReload       bool   `yaml:"hotReload"`
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

type LoginServersConfig struct {
	Telnet TelnetConfig `yaml:"telnet"`
	SSH    SSHConfig    `yaml:"ssh"`
}

type TelnetConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type SSHConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	KeyFile string `yaml:"keyFile"`
}

func Load(filename string) (*Config, error) {
	// Start with a base config
	cfg := &Config{}

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
