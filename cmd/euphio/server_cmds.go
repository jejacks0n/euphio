package main

import (
	"euphio/internal/ansi"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"euphio/internal/app"
	"euphio/internal/network"
	"euphio/internal/network/ssh"
	"euphio/internal/network/telnet"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:              "server",
	Short:            "Start the server",
	PersistentPreRun: bootAppForServer,
	Run:              startServer,
}

func bootAppForServer(cmd *cobra.Command, args []string) {
	if err := app.Boot(cfgFile, false); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func startServer(cmd *cobra.Command, args []string) {
	ansi.RenderArt(os.Stdout, "boot", true)

	restartChan := make(chan struct{}, 1)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	for {
		var watcher *fsnotify.Watcher
		if app.Config.HotReload {
			// Setup Watcher
			var err error
			watcher, err = fsnotify.NewWatcher()
			if err != nil {
				app.Logger.Error("Failed to create watcher", "err", err)
			} else {
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

				go func(w *fsnotify.Watcher) {
					for {
						select {
						case event, ok := <-w.Events:
							if !ok {
								return
							}
							if event.Op&fsnotify.Write == fsnotify.Write {
								// Check if hot reload is still enabled (in case it was disabled in the new config)
								if !app.Config.HotReload {
									continue
								}

								// Try to make path relative for cleaner logging
								relPath := event.Name
								if cwd, err := os.Getwd(); err == nil {
									if rel, err := filepath.Rel(cwd, event.Name); err == nil {
										relPath = rel
									}
								}

								app.Logger.Info("Config file modified, rebooting app...", "file", relPath)
								select {
								case restartChan <- struct{}{}:
								default:
									// restart pending
								}
							}
						case err, ok := <-w.Errors:
							if !ok {
								return
							}
							app.Logger.Error("Watcher error", "err", err)
						}
					}
				}(watcher)
			}
		}

		var wg sync.WaitGroup
		var sshServer *ssh.Server
		var telnetServer *telnet.Server

		sshEnabled := app.Config.Listeners.SSH.Enabled
		telnetEnabled := app.Config.Listeners.Telnet.Enabled

		if !telnetEnabled && !sshEnabled {
			app.Logger.Warn("No listeners enabled.")
			// Wait for config change or stop
			select {
			case <-stopChan:
				if watcher != nil {
					watcher.Close()
				}
				return
			case <-restartChan:
				if watcher != nil {
					watcher.Close()
				}
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
			if watcher != nil {
				watcher.Close()
			}
			return

		case <-restartChan:
			if sshServer != nil {
				sshServer.Stop()
			}
			if telnetServer != nil {
				telnetServer.Stop()
			}
			if watcher != nil {
				watcher.Close()
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
