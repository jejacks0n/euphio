package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"euphio/internal/app"
	"euphio/internal/assets"
	"euphio/internal/network"
	"euphio/internal/network/ssh"
	"euphio/internal/network/telnet"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the server",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := app.Boot(cfgFile, false); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
	Run: startServer,
}

func startServer(cmd *cobra.Command, args []string) {
	if content, err := assets.FS.ReadFile("boot.asc"); err == nil {
		lines := strings.Split(string(content), "\n")
		width, _, err := term.GetSize(int(os.Stdout.Fd()))

		for _, line := range lines {
			if err == nil && width > 0 {
				runes := []rune(line)
				if len(runes) > width {
					line = string(runes[:width])
				}
			}
			fmt.Println(line)
		}
	}

	restartChan := make(chan struct{}, 1)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	if app.Config.General.HotReload {
		// Setup Watcher
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			app.Logger.Error("Failed to create watcher", "err", err)
			os.Exit(1)
		}
		defer watcher.Close()

		// Watch all loaded config files
		for _, file := range app.Config.LoadedFiles {
			err = watcher.Add(file)

			// Try to make path relative for cleaner logging
			relPath := file
			if cwd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(cwd, file); err == nil {
					relPath = rel
				}
			}

			if err != nil {
				app.Logger.Error("Failed to watch config file", "file", relPath, "err", err)
			} else {
				app.Logger.Debug("Watching config file", "file", relPath)
			}
		}

		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						// Check if hot reload is still enabled (in case it was disabled in the new config)
						if !app.Config.General.HotReload {
							continue
						}

						app.Logger.Info("Config modified, rebooting app...")
						select {
						case restartChan <- struct{}{}:
						default:
							// restart pending
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					app.Logger.Error("Watcher error", "err", err)
				}
			}
		}()
	}

	for {
		var wg sync.WaitGroup
		var sshServer *ssh.Server
		var telnetServer *telnet.Server

		sshEnabled := app.Config.LoginServers.SSH.Enabled
		telnetEnabled := app.Config.LoginServers.Telnet.Enabled

		if !telnetEnabled && !sshEnabled {
			app.Logger.Warn("No login servers enabled.")
			// Wait for config change or stop
			select {
			case <-stopChan:
				return
			case <-restartChan:
				app.Boot(cfgFile, false)
				continue
			}
		}

		// Start SSH Server
		if sshEnabled {
			wg.Add(1)
			sshServer = network.NewSSH()
			go func() {
				defer wg.Done()
				if err := sshServer.ListenAndServe(); err != nil {
					app.Logger.Error("SSH Server stopped", "err", err)
				}
			}()
		}

		// Start Telnet Server
		if telnetEnabled {
			wg.Add(1)
			telnetServer = network.NewTelnet()
			go func() {
				defer wg.Done()
				if err := telnetServer.ListenAndServe(); err != nil {
					app.Logger.Error("Telnet Server stopped", "err", err)
				}
			}()
		}

		// Wait for stop or restart
		select {
		case <-stopChan:
			app.Logger.Info("Shutting down...")
			if sshServer != nil {
				sshServer.Stop()
			}
			if telnetServer != nil {
				telnetServer.Stop()
			}
			return

		case <-restartChan:
			if sshServer != nil {
				sshServer.Stop()
			}
			if telnetServer != nil {
				telnetServer.Stop()
			}

			// Wait for servers to stop
			wg.Wait()

			// Reload Config
			if err := app.Boot(cfgFile, false); err != nil {
				app.Logger.Error("Failed to reload config", "err", err)
				// We continue, which will restart servers with the *existing* config/store
				// because Boot did not swap them on failure.
			}
		}
	}
}
