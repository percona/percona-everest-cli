// Package install holds logic for install command.
package install

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/percona/percona-everest-cli/pkg/install"
)

// NewOperatorsCmd returns a new operators command.
func NewOperatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operators",
		Short: "TODO: Operators help",
		Long:  `TODO: Operators help`,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := parseConfig()
			if err != nil {
				os.Exit(1)
			}
			op, err := install.NewOperators(c)
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
			if err := op.ProvisionCluster(); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
			if err := op.ConnectToEverest(); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
		},
	}

	initOperatorsFlags(cmd)

	return cmd
}

func initOperatorsFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("monitoring.enabled", "m", true, "Enable monitoring")
	cmd.Flags().StringP("monitoring.type", "", "pmm", "Monitoring type")
	cmd.Flags().StringP("monitoring.pmm.endpoint", "", "http://127.0.0.1", "PMM endpoint URL")
	cmd.Flags().StringP("monitoring.pmm.username", "", "admin", "PMM username")
	cmd.Flags().StringP("monitoring.pmm.password", "", "password", "PMM password")
	cmd.Flags().BoolP("enable_backup", "b", false, "Enable backups")
	cmd.Flags().BoolP("install_olm", "o", true, "Install OLM")
	cmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "specify kubeconfig")

	viper.BindPFlag("monitoring.enabled", cmd.Flags().Lookup("monitoring.enabled"))           //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.type", cmd.Flags().Lookup("monitoring.type"))                 //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.endpoint", cmd.Flags().Lookup("monitoring.pmm.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.username", cmd.Flags().Lookup("monitoring.pmm.username")) //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.password", cmd.Flags().Lookup("monitoring.pmm.password")) //nolint:errcheck,gosec
	viper.BindPFlag("enable_backup", cmd.Flags().Lookup("enable_backup"))                     //nolint:errcheck,gosec
	viper.BindPFlag("install_olm", cmd.Flags().Lookup("install_olm"))                         //nolint:errcheck,gosec
	viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig"))                           //nolint:errcheck,gosec
}

func parseConfig() (*install.OperatorsConfig, error) {
	c := &install.OperatorsConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
