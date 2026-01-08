package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"euphio/internal/app"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		app.Boot(cfgFile, !verbose)
	},
}

var verbose bool

func init() {
	userCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userInfoCmd)
	userCmd.AddCommand(userPassCmd)
	userCmd.AddCommand(userRemoveCmd)
	userCmd.AddCommand(userRenameCmd)
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			username string
			password string
		)

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Username").
					Description("Enter the desired username").
					Value(&username).
					Validate(func(str string) error {
						if len(str) < 3 {
							return fmt.Errorf("username must be at least 3 characters")
						}
						// Check if user exists
						if _, err := app.Store.FindUserByUsername(str); err == nil {
							return fmt.Errorf("username already taken")
						}
						return nil
					}),
				huh.NewInput().
					Title("Password").
					Description("Enter a strong password").
					EchoMode(huh.EchoModePassword).
					Value(&password).
					Validate(func(str string) error {
						if len(str) < 6 {
							return fmt.Errorf("password must be at least 6 characters")
						}
						return nil
					}),
			),
		)

		err := form.Run()
		if err != nil {
			log.Fatal(err)
		}

		if err := app.Store.CreateUser(username, password); err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}

		fmt.Printf("User '%s' created successfully!\n", username)
	},
}

var userInfoCmd = &cobra.Command{
	Use:   "info [username]",
	Short: "Display information about a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		user, err := app.Store.FindUserByUsername(username)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		// Use tabwriter for pretty output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%d\n", user.ID)
		fmt.Fprintf(w, "Username:\t%s\n", user.Username)
		fmt.Fprintf(w, "Created At:\t%s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
		w.Flush()
	},
}

var userPassCmd = &cobra.Command{
	Use:   "password [username] [new_password]",
	Short: "Set a user's password",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		newPass := args[1]

		if err := app.Store.UpdatePassword(username, newPass); err != nil {
			log.Fatalf("Error updating password: %v", err)
		}
		fmt.Printf("Password updated for user '%s'.\n", username)
	},
}

var userRemoveCmd = &cobra.Command{
	Use:   "remove [username]",
	Short: "Permanently remove a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]

		if err := app.Store.RemoveUser(username); err != nil {
			log.Fatalf("Error removing user: %v", err)
		}
		fmt.Printf("User '%s' removed.\n", username)
	},
}

var userRenameCmd = &cobra.Command{
	Use:   "rename [old_name] [new_name]",
	Short: "Rename a user",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldName := args[0]
		newName := args[1]

		if err := app.Store.RenameUser(oldName, newName); err != nil {
			log.Fatalf("Error renaming user: %v", err)
		}
		fmt.Printf("User '%s' renamed to '%s'.\n", oldName, newName)
	},
}
