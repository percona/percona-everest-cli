package commands

import (
	"github.com/spf13/cobra"

	"github.com/percona/percona-everest-cli/commands/delete"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete",
	}

	cmd.AddCommand(delete.NewMySQLCmd())

	return cmd
}
