package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
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
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Expand environment variables in the YAML content
	expandedData := []byte(os.ExpandEnv(string(data)))

	var cfg Config
	err = yaml.Unmarshal(expandedData, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
