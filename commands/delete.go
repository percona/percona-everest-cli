package commands

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/commands/delete"
)

func newDeleteCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete",
	}

	cmd.AddCommand(delete.NewMySQLCmd(l))

	return cmd
}
