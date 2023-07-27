// Package list holds logic for list commands.
package list

import (
	"fmt"
	"os"

	"github.com/percona/percona-everest-backend/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/list"
	"github.com/percona/percona-everest-cli/pkg/output"
)

// NewDatabaseEnginesCmd returns a new database engines command.
func NewDatabaseEnginesCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "databaseengines",
		Aliases: []string{"databaseengine", "dbengines", "dbengine"},
		Run: func(cmd *cobra.Command, args []string) {
			initDatabaseEnginesViperFlags(cmd)

			c, err := parseDatabaseEnginesConfig()
			if err != nil {
				os.Exit(1)
			}

			everestCl, err := client.NewClient(fmt.Sprintf("%s/v1", c.Everest.Endpoint))
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			command := list.NewDatabaseEngines(c, everestClConnector, l)
			dbEngines, err := command.Run(cmd.Context())
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}

			output.PrintOutput(cmd, l, dbEngines)
		},
	}

	initDatabaseEnginesFlags(cmd)

	return cmd
}

func initDatabaseEnginesFlags(cmd *cobra.Command) {
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8081", "Everest endpoint URL")
	cmd.Flags().String("kubernetes-id", "", "Kubernetes cluster ID")
	cmd.MarkFlagRequired("kubernetes-id") //nolint:errcheck,gosec
}

func initDatabaseEnginesViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("everest.endpoint", cmd.Flags().Lookup("everest.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("kubernetes-id", cmd.Flags().Lookup("kubernetes-id"))       //nolint:errcheck,gosec
}

func parseDatabaseEnginesConfig() (*list.DBEnginesConfig, error) {
	c := &list.DBEnginesConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
