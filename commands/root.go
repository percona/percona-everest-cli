// Package commands implements main logic for cli commands.
package commands

import "github.com/spf13/cobra"

// NewRootCmd creates a new root command for the cli.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "everest",
	}

	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newProvisionCmd())
	rootCmd.AddCommand(newDeleteCmd())

	return rootCmd
}
