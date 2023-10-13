package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/version"
)

func newVersionCmd(_ *zap.SugaredLogger) *cobra.Command {
	return &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.FullVersionInfo()) //nolint:forbidigo
		},
	}
}
