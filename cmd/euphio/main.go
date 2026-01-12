package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"euphio/internal/app"
)

var cfgFile string

func main() {
	configPath := os.Getenv("EUPHIO_CONFIG")
	if configPath == "" {
		configPath = "config.yml"
	}

	var rootCmd = &cobra.Command{
		Use:     "euphio",
		Short:   "Euphio BBS",
		Version: app.Version,
		Run: func(cmd *cobra.Command, args []string) {
			bootAppForServer(cmd, args)
			startServer(cmd, args)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", configPath, "config file")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
