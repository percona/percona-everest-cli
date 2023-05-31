package commands

import (
	"github.com/spf13/cobra"

	"github.com/percona/percona-everest-cli/commands/install"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "TODO: Install help",
		Long:  `TODO: Install help`,
	}

	cmd.AddCommand(install.NewOperatorsCmd())

	return cmd
}
