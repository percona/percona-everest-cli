package commands

import (
	"fmt"

	"github.com/percona/percona-everest-cli/pkg/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newVersionCmd(l *zap.SugaredLogger) *cobra.Command {
	return &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.FullVersionInfo())
		},
	}
}
