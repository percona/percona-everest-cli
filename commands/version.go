package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/version"
)

func newVersionCmd(l *zap.SugaredLogger) *cobra.Command {
	return &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			outputJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				l.Errorf("could not parse json global flag. Error: %s", err)
				return
			}
			if !outputJSON {
				fmt.Println(version.FullVersionInfo()) //nolint:forbidigo
			}
			version, err := version.FullVersionJSON()
			if err != nil {
				l.Errorf("could not print JSON. Error: %s", err)
			}
			fmt.Println(version) //nolint:forbidigo

		},
	}
}
