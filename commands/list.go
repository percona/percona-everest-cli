package commands

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/commands/list"
)

func newListCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
	}

	cmd.AddCommand(list.NewDatabaseEnginesCmd(l))
	cmd.AddCommand(list.NewVersionsCmd(l))

	return cmd
}
