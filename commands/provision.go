package commands

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/commands/provision"
)

func newProvisionCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "provision",
	}

	cmd.AddCommand(provision.NewMySQLCmd(l))

	return cmd
}
