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

// Package monitoring holds commands for monitoring command.
package monitoring

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/monitoring"
	"github.com/percona/percona-everest-cli/pkg/output"
)

// NewMonitoringCmd returns a new enable monitoring command.
func NewMonitoringCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "enable",
		Run: func(cmd *cobra.Command, args []string) {
			initMonitoringViperFlags(cmd)

			c, err := parseResetConfig()
			if err != nil {
				os.Exit(1)
			}

			command, err := monitoring.NewMonitoring(*c, l)
			if err != nil {
				output.PrintError(err, l)
				os.Exit(1)
			}

			err = command.Run(cmd.Context())
			if err != nil {
				output.PrintError(err, l)
				os.Exit(1)
			}
		},
	}

	initMonitoringFlags(cmd)

	return cmd
}

func initMonitoringFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "Path to a kubeconfig")
	cmd.Flags().String("namespace", "percona-everest", "Namespace where Percona Everest is deployed")
	cmd.Flags().Bool("skip-wizard", false, "Skip configuration wizard")
	cmd.Flags().String("everest-url", "", "A URL to connect to Everest")
	cmd.Flags().String("everest-token", "", "A Token to authenticate in Everest")
	cmd.Flags().String("instance-name", "",
		"Monitoring instance name from Everest. If defined, other monitoring configuration is ignored",
	)
	cmd.Flags().String("new-instance-name", "",
		"Name for a new monitoring instance if it's going to be created",
	)
	cmd.Flags().String("type", "pmm", "Monitoring type")
	cmd.Flags().String("pmm.endpoint", "http://127.0.0.1", "PMM endpoint URL")
	cmd.Flags().String("pmm.username", "admin", "PMM username")
	cmd.Flags().String("pmm.password", "", "PMM password")
}

func initMonitoringViperFlags(cmd *cobra.Command) {
	viper.BindEnv("kubeconfig")                                                   //nolint:errcheck,gosec
	viper.BindPFlag("skip-wizard", cmd.Flags().Lookup("skip-wizard"))             //nolint:errcheck,gosec
	viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig"))               //nolint:errcheck,gosec
	viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))                 //nolint:errcheck,gosec
	viper.BindPFlag("everest-url", cmd.Flags().Lookup("everest-url"))             //nolint:errcheck,gosec
	viper.BindPFlag("everest-token", cmd.Flags().Lookup("everest-token"))         //nolint:errcheck,gosec
	viper.BindPFlag("instance-name", cmd.Flags().Lookup("instance-name"))         //nolint:errcheck,gosec
	viper.BindPFlag("new-instance-name", cmd.Flags().Lookup("new-instance-name")) //nolint:errcheck,gosec
	viper.BindPFlag("type", cmd.Flags().Lookup("type"))                           //nolint:errcheck,gosec
	viper.BindPFlag("pmm.endpoint", cmd.Flags().Lookup("pmm.endpoint"))           //nolint:errcheck,gosec
	viper.BindPFlag("pmm.username", cmd.Flags().Lookup("pmm.username"))           //nolint:errcheck,gosec
	viper.BindPFlag("pmm.password", cmd.Flags().Lookup("pmm.password"))           //nolint:errcheck,gosec
}

func parseResetConfig() (*monitoring.Config, error) {
	c := &monitoring.Config{}
	err := viper.Unmarshal(c)
	return c, err
}
