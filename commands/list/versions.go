package list

import (
	"fmt"
	"os"

	"github.com/percona/percona-everest-backend/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/list"
	"github.com/percona/percona-everest-cli/pkg/output"
)

// NewVersionsCmd returns a new versions command.
func NewVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "versions",
		Aliases: []string{"version"},
		Run: func(cmd *cobra.Command, args []string) {
			initVersionsViperFlags(cmd)

			c, err := parseVersionsConfig()
			if err != nil {
				os.Exit(1)
			}

			everestCl, err := client.NewClient(fmt.Sprintf("%s/v1", c.Everest.Endpoint))
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			command := list.NewVersions(*c, everestClConnector)
			res, err := command.Run(cmd.Context())
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}

			output.PrintOutput(cmd, res)
		},
	}

	initVersionsFlags(cmd)

	return cmd
}

func initVersionsFlags(cmd *cobra.Command) {
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8081", "Everest endpoint URL")
	cmd.Flags().String("kubernetes-id", "", "Kubernetes cluster ID")

	cmd.Flags().String("type", "", "Database engine type")
}

func initVersionsViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("everest.endpoint", cmd.Flags().Lookup("everest.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("kubernetes-id", cmd.Flags().Lookup("kubernetes-id"))       //nolint:errcheck,gosec

	viper.BindPFlag("type", cmd.Flags().Lookup("type")) //nolint:errcheck,gosec
}

func parseVersionsConfig() (*list.VersionsConfig, error) {
	c := &list.VersionsConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
