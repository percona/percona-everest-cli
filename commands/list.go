package commands

import (
	"github.com/spf13/cobra"

	"github.com/percona/percona-everest-cli/commands/list"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
	}

	cmd.AddCommand(list.NewDatabaseEnginesCmd())

	return cmd
}
