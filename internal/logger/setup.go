package logger

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"

	"euphio/internal/config"
)

func Setup(configs []config.LoggerConfig, quiet bool) *slog.Logger {
	if quiet {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	var handlers []slog.Handler

	for _, cfg := range configs {
		level := parseLogLevel(cfg.Level)

		// Allow the time to be hidden
		replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
			if cfg.HideTime && a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		}

		// Determine time format
		timeFormat := time.TimeOnly
		if cfg.TimeFormat != "" {
			timeFormat = cfg.TimeFormat
		}

		if cfg.Stdout {
			handlers = append(handlers, tint.NewHandler(os.Stdout, &tint.Options{
				NoColor:     !isatty.IsTerminal(os.Stdout.Fd()),
				Level:       level,
				AddSource:   cfg.Source,
				ReplaceAttr: replaceAttr,
				TimeFormat:  timeFormat,
			}))
		}

		if cfg.File != "" {
			dir := filepath.Dir(cfg.File)
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Printf("Failed to create log directory %s: %v", dir, err)
				continue
			}

			file, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("Failed to open log file %s: %v", cfg.File, err)
				continue
			}

			handlers = append(handlers, tint.NewHandler(file, &tint.Options{
				NoColor:     true,
				Level:       level,
				AddSource:   cfg.Source,
				ReplaceAttr: replaceAttr,
				TimeFormat:  timeFormat,
			}))
		}
	}

	var logger *slog.Logger
	if len(handlers) == 0 {
		// Fallback if no loggers configured
		logger = slog.New(tint.NewHandler(os.Stdout, nil))
	} else if len(handlers) == 1 {
		logger = slog.New(handlers[0])
	} else {
		logger = slog.New(NewFanout(handlers...))
	}

	slog.SetDefault(logger)
	return logger
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
