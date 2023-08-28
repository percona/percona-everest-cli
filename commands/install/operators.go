// percona-everest-cli
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package install holds logic for install command.

// Package install ...
package install

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/install"
	"github.com/percona/percona-everest-cli/pkg/output"
)

// NewOperatorsCmd returns a new operators command.
func NewOperatorsCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "operators",
		Run: func(cmd *cobra.Command, args []string) {
			initOperatorsViperFlags(cmd)

			c, err := parseConfig()
			if err != nil {
				os.Exit(1)
			}

			op, err := install.NewOperators(*c, l)
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}

			if err := op.Run(cmd.Context()); err != nil {
				output.PrintError(err, l)
				os.Exit(1)
			}
		},
	}

	initOperatorsFlags(cmd)

	return cmd
}

func initOperatorsFlags(cmd *cobra.Command) {
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8080", "Everest endpoint URL")
	cmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "Path to a kubeconfig")
	cmd.Flags().StringP("name", "n", "", "Kubernetes cluster name")
	cmd.Flags().String("namespace", "percona-everest", "Namespace into which Percona Everest components are deployed to")
	cmd.Flags().Bool("skip-wizard", false, "Skip installation wizard")

	cmd.Flags().BoolP("monitoring.enable", "m", true, "Enable monitoring")
	cmd.Flags().String("monitoring.instance-name", "",
		"Monitoring instance name from Everest. If defined, other monitoring configuration is ignored",
	)
	cmd.Flags().String("monitoring.new-instance-name", "",
		"Name for a new monitoring instance if it's going to be created",
	)
	cmd.Flags().String("monitoring.type", "pmm", "Monitoring type")
	cmd.Flags().String("monitoring.pmm.endpoint", "http://127.0.0.1", "PMM endpoint URL")
	cmd.Flags().String("monitoring.pmm.username", "admin", "PMM username")
	cmd.Flags().String("monitoring.pmm.password", "", "PMM password")

	cmd.Flags().Bool("backup.enable", false, "Enable backups")
	cmd.Flags().String("backup.endpoint", "", "Backup endpoint URL")
	cmd.Flags().String("backup.region", "", "Backup region")
	cmd.Flags().String("backup.bucket", "", "Backup bucket")
	cmd.Flags().String("backup.access-key", "", "Backup access key")
	cmd.Flags().String("backup.secret-key", "", "Backup secret key")

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

	viper.BindPFlag("monitoring.enable", cmd.Flags().Lookup("monitoring.enable"))                       //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.instance-name", cmd.Flags().Lookup("monitoring.instance-name"))         //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.new-instance-name", cmd.Flags().Lookup("monitoring.new-instance-name")) //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.type", cmd.Flags().Lookup("monitoring.type"))                           //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.endpoint", cmd.Flags().Lookup("monitoring.pmm.endpoint"))           //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.username", cmd.Flags().Lookup("monitoring.pmm.username"))           //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.password", cmd.Flags().Lookup("monitoring.pmm.password"))           //nolint:errcheck,gosec

	viper.BindPFlag("backup.enable", cmd.Flags().Lookup("backup.enable"))         //nolint:errcheck,gosec
	viper.BindPFlag("backup.endpoint", cmd.Flags().Lookup("backup.endpoint"))     //nolint:errcheck,gosec
	viper.BindPFlag("backup.region", cmd.Flags().Lookup("backup.region"))         //nolint:errcheck,gosec
	viper.BindPFlag("backup.bucket", cmd.Flags().Lookup("backup.bucket"))         //nolint:errcheck,gosec
	viper.BindPFlag("backup.access-key", cmd.Flags().Lookup("backup.access-key")) //nolint:errcheck,gosec
	viper.BindPFlag("backup.secret-key", cmd.Flags().Lookup("backup.secret-key")) //nolint:errcheck,gosec

	viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig")) //nolint:errcheck,gosec
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))             //nolint:errcheck,gosec
	viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))   //nolint:errcheck,gosec

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
