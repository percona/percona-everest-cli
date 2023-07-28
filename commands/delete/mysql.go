// Package delete provides delete sub-commands.
package delete //nolint:predeclared

import (
	"fmt"
	"os"

	"github.com/percona/percona-everest-backend/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/percona/percona-everest-cli/pkg/delete"
	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
)

// NewMySQLCmd returns a new MySQL command.
func NewMySQLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "mysql",
		Run: func(cmd *cobra.Command, args []string) {
			initMySQLViperFlags(cmd)

			c, err := parseMySQLConfig()
			if err != nil {
				os.Exit(1)
			}

			everestCl, err := client.NewClient(fmt.Sprintf("%s/v1", c.Everest.Endpoint))
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			command := delete.NewMySQL(*c, everestClConnector)

			if err := command.Run(cmd.Context()); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
		},
	}

	initMySQLFlags(cmd)

	return cmd
}

func initMySQLFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Cluster name")
	cmd.MarkFlagRequired("name")
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8081", "Everest endpoint URL")
	cmd.Flags().String("kubernetes-id", "", "Kubernetes cluster ID in Everest")
	cmd.MarkFlagRequired("kubernetes-id")

	cmd.Flags().BoolP("force", "f", false, "Do not prompt to confirm removal")
}

func initMySQLViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))                         //nolint:errcheck,gosec
	viper.BindPFlag("everest.endpoint", cmd.Flags().Lookup("everest.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("kubernetes-id", cmd.Flags().Lookup("kubernetes-id"))       //nolint:errcheck,gosec

	viper.BindPFlag("force", cmd.Flags().Lookup("force")) //nolint:errcheck,gosec
}

func parseMySQLConfig() (*delete.MySQLConfig, error) {
	c := &delete.MySQLConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
