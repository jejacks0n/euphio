package app

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"euphio/internal/config"
	"euphio/internal/logger"
	"euphio/internal/nodes"
	"euphio/internal/store"
)

var (
	Version = "v0.1.000" // Default version, can be overwritten by build flags
	Config  *config.Config
	Store   *store.Store
	Logger  *slog.Logger
	Nodes   *nodes.Manager
)

func Boot(configPath string, quiet bool) error {
	if configPath == "" {
		configPath = "config/example.yml"
	}

	// Load the configuration
	newConfig, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("configuration file not found: %s\n\nPlease ensure you have a valid configuration file.\nYou can specify one using the -c flag or EUPHIO_CONFIG environment variable.\nExample: euphio -c config/myconfig.yml\n\nYou can get a basic configuration setup by initializing the current path.\nExample: euphio init", configPath)
		}
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// If all successful, swap globals and cleanup.
	Config = newConfig

	Nodes = nodes.NewManager(Config.MaxNodes)

	// Setup Logger
	Logger = logger.Setup(Config.Loggers, quiet)

	// Prepare the data store
	dir := Config.Paths.Data
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create data path: %w", err)
	}

	newStore, err := store.New(filepath.Clean(filepath.Join(dir, "data.sqlite3")), quiet)
	if err != nil {
		return fmt.Errorf("failed to connect to the database: %w", err)
	}

	if Store != nil {
		if err := Store.Close(); err != nil {
			Logger.Error("Failed to close existing store", "err", err)
		}
	}
	Store = newStore

	if !quiet {
		Logger.Info("Loaded configuration", "file", configPath)
	}

	return nil
}

// initConfig and initStore are removed as they are now integrated into Boot for safety.
