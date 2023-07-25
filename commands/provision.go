package commands

import (
	"github.com/spf13/cobra"

	"github.com/percona/percona-everest-cli/commands/provision"
)

func newProvisionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "provision",
	}

	cmd.AddCommand(provision.NewMySQLCmd())

	return cmd
}
