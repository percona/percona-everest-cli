// Package commands implements main logic for cli commands.
package commands

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/logger"
)

// NewRootCmd creates a new root command for the cli.
func NewRootCmd(l *zap.SugaredLogger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "everest",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.InitLoggerInRootCmd(cmd, l)
			l.Debug("Debug logging enabled")
		},
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose mode")
	rootCmd.PersistentFlags().Bool("json", false, "Set output type to JSON")

	rootCmd.AddCommand(newInstallCmd(l))
	rootCmd.AddCommand(newProvisionCmd(l))
	rootCmd.AddCommand(newListCmd(l))
	rootCmd.AddCommand(newDeleteCmd(l))

	return rootCmd
}
