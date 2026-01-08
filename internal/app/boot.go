package app

import (
	"euphio/internal/logger"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"euphio/internal/config"
	"euphio/internal/store"
)

var (
	Config *config.Config
	Store  *store.Store
	Logger *slog.Logger
)

func Boot(configPath string, quiet bool) error {
	if configPath == "" {
		configPath = "config/example.yml"
	}

	// Load the configuration
	newConfig, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// If all successful, swap globals and cleanup.
	Config = newConfig

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
		Logger.Info("Successfully loaded configuration", "file", configPath)
	}

	return nil
}

// initConfig and initStore are removed as they are now integrated into Boot for safety.
