package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"euphio/internal/app"
	"euphio/internal/assets"
	"euphio/internal/network"
	"euphio/internal/network/telnet"
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
		fmt.Print(string(content))
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

		configPath, err := filepath.Abs(cfgFile)
		if err != nil {
			app.Logger.Error("Failed to resolve config path", "err", err)
			os.Exit(1)
		}

		err = watcher.Add(configPath)
		if err != nil {
			app.Logger.Error("Failed to watch config file", "err", err)
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
						app.Logger.Info("Config file modified, rebooting app...")
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
		var sshServer *network.SSH
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
