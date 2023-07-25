// Package delete holds logic for delete command.
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

// NewClusterCmd returns a new cluster command.
func NewClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cluster",
		Run: func(cmd *cobra.Command, args []string) {
			initClusterViperFlags(cmd)
			c, err := parseClusterConfig()
			if err != nil {
				os.Exit(1)
			}

			everestCl, err := client.NewClient(fmt.Sprintf("%s/v1", c.Everest.Endpoint))
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			op := delete.NewCluster(*c, everestClConnector)
			if err := op.Run(cmd.Context()); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
		},
	}

	initClusterFlags(cmd)

	return cmd
}

func initClusterFlags(cmd *cobra.Command) {
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8081", "Everest endpoint URL")
	cmd.Flags().String("name", "", "Kubernetes cluster name in Everest")
	cmd.Flags().BoolP("force", "f", false, "Force removal in case there are database clusters running")
}

func initClusterViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("everest.endpoint", cmd.Flags().Lookup("everest.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))                         //nolint:errcheck,gosec
	viper.BindPFlag("force", cmd.Flags().Lookup("force"))                       //nolint:errcheck,gosec
}

func parseClusterConfig() (*delete.ClusterConfig, error) {
	c := &delete.ClusterConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
