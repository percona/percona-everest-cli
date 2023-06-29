// Package install holds logic for install command.
package install

import (
	"fmt"
	"os"

	"github.com/percona/percona-everest-backend/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/install"
)

// NewOperatorsCmd returns a new operators command.
func NewOperatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "operators",
		Run: func(cmd *cobra.Command, args []string) {
			initOperatorsViperFlags(cmd)

			c, err := parseConfig()
			if err != nil {
				os.Exit(1)
			}

			everestCl, err := client.NewClient(fmt.Sprintf("%s/v1", c.Everest.Endpoint))
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			op, err := install.NewOperators(c, everestClConnector)
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}

			if err := op.Run(cmd.Context()); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
		},
	}

	initOperatorsFlags(cmd)

	return cmd
}

func initOperatorsFlags(cmd *cobra.Command) {
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8081", "Everest endpoint URL")
	cmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "Path to a kubeconfig")
	cmd.Flags().StringP("name", "n", "", "Kubernetes cluster name")
	cmd.Flags().Bool("skip-wizard", false, "Skip installation wizard")

	cmd.Flags().BoolP("monitoring.enable", "m", true, "Enable monitoring")
	cmd.Flags().String("monitoring.type", "pmm", "Monitoring type")
	cmd.Flags().String("monitoring.pmm.endpoint", "http://127.0.0.1", "PMM endpoint URL")
	cmd.Flags().String("monitoring.pmm.username", "admin", "PMM username")
	cmd.Flags().String("monitoring.pmm.password", "password", "PMM password")

	cmd.Flags().Bool("backup.enable", false, "Enable backups")
	cmd.Flags().String("backup.endpoint", "", "Backup endpoint URL")
	cmd.Flags().String("backup.region", "", "Backup region")
	cmd.Flags().String("backup.bucket", "", "Backup bucket")
	cmd.Flags().String("backup.access-key", "", "Backup access key")
	cmd.Flags().String("backup.secret-key", "", "Backup secret key")

	cmd.Flags().String("operator.namespace", "percona-everest", "Namespace where operators are deployed to")
	cmd.Flags().Bool("operator.mongodb", true, "Install MongoDB operator")
	cmd.Flags().Bool("operator.postgresql", true, "Install PostgreSQL operator")
	cmd.Flags().Bool("operator.xtradb-cluster", true, "Install XtraDB Cluster operator")

	cmd.Flags().String("channel.everest", "stable-v0", "Channel for Everest operator")
	cmd.Flags().String("channel.victoria-metrics", "stable-v0", "Channel for VictoriaMetrics operator")
	cmd.Flags().String("channel.xtradb-cluster", "stable-v1", "Channel for XtraDB Cluster operator")
	cmd.Flags().String("channel.mongodb", "stable-v1", "Channel for MongoDB operator")
	cmd.Flags().String("channel.postgresql", "fast-v2", "Channel for PostgreSQL operator")
}

func initOperatorsViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("everest.endpoint", cmd.Flags().Lookup("everest.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("skip-wizard", cmd.Flags().Lookup("skip-wizard"))           //nolint:errcheck,gosec

	viper.BindPFlag("monitoring.enable", cmd.Flags().Lookup("monitoring.enable"))             //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.type", cmd.Flags().Lookup("monitoring.type"))                 //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.endpoint", cmd.Flags().Lookup("monitoring.pmm.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.username", cmd.Flags().Lookup("monitoring.pmm.username")) //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.password", cmd.Flags().Lookup("monitoring.pmm.password")) //nolint:errcheck,gosec

	viper.BindPFlag("backup.enable", cmd.Flags().Lookup("backup.enable"))         //nolint:errcheck,gosec
	viper.BindPFlag("backup.endpoint", cmd.Flags().Lookup("backup.endpoint"))     //nolint:errcheck,gosec
	viper.BindPFlag("backup.region", cmd.Flags().Lookup("backup.region"))         //nolint:errcheck,gosec
	viper.BindPFlag("backup.bucket", cmd.Flags().Lookup("backup.bucket"))         //nolint:errcheck,gosec
	viper.BindPFlag("backup.access-key", cmd.Flags().Lookup("backup.access-key")) //nolint:errcheck,gosec
	viper.BindPFlag("backup.secret-key", cmd.Flags().Lookup("backup.secret-key")) //nolint:errcheck,gosec

	viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig")) //nolint:errcheck,gosec
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))             //nolint:errcheck,gosec

	viper.BindPFlag("operator.namespace", cmd.Flags().Lookup("operator.namespace"))           //nolint:errcheck,gosec
	viper.BindPFlag("operator.mongodb", cmd.Flags().Lookup("operator.mongodb"))               //nolint:errcheck,gosec
	viper.BindPFlag("operator.postgresql", cmd.Flags().Lookup("operator.postgresql"))         //nolint:errcheck,gosec
	viper.BindPFlag("operator.xtradb-cluster", cmd.Flags().Lookup("operator.xtradb-cluster")) //nolint:errcheck,gosec

	viper.BindPFlag("channel.victoria-metrics", cmd.Flags().Lookup("channel.victoria-metrics")) //nolint:errcheck,gosec
	viper.BindPFlag("channel.xtradb-cluster", cmd.Flags().Lookup("channel.xtradb-cluster"))     //nolint:errcheck,gosec
	viper.BindPFlag("channel.mongodb", cmd.Flags().Lookup("channel.mongodb"))                   //nolint:errcheck,gosec
	viper.BindPFlag("channel.postgresql", cmd.Flags().Lookup("channel.postgresql"))             //nolint:errcheck,gosec
	viper.BindPFlag("channel.everest", cmd.Flags().Lookup("channel.everest"))                   //nolint:errcheck,gosec
}

func parseConfig() (*install.OperatorsConfig, error) {
	c := &install.OperatorsConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
