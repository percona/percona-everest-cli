// Package commands implements main logic for cli commands.
package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewRootCmd creates a new root command for the cli.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "everest",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			verbose, err := cmd.Flags().GetBool("verbose")
			if err != nil {
				logrus.Warn(`Could not parse "verbose" flag`)
			}

			if verbose {
				logrus.SetLevel(logrus.DebugLevel)
			}
		},
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose mode")
	rootCmd.PersistentFlags().Bool("json", false, "Set output type to JSON")

	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newListCmd())

	return rootCmd
}
