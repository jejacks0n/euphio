package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"euphio/internal/app"
)

var cfgFile string

const (
	DefaultConfigPath = "config/example.yml"
)

func main() {
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", DefaultConfigPath, "config file (default is "+DefaultConfigPath+")")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(userCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
