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
		configPath = "config/example.yml"
	}

	var rootCmd = &cobra.Command{
		Use:     "euphio",
		Short:   "EUPHiO CLI",
		Version: "0.1.000",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if err := app.Boot(cfgFile, false); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
		Run: startServer,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", configPath, "config file")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(userCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
