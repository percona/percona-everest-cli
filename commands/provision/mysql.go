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

// Package provision provides provision sub-commands.
package provision

import (
	"fmt"
	"os"

	"github.com/percona/percona-everest-backend/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/output"
	"github.com/percona/percona-everest-cli/pkg/provision"
)

// NewMySQLCmd returns a new MySQL command.
func NewMySQLCmd(l *zap.SugaredLogger) *cobra.Command {
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
				l.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			command := provision.NewMySQL(*c, everestClConnector, l)

			if err := command.Run(cmd.Context()); err != nil {
				output.PrintError(err, l)
				os.Exit(1)
			}
		},
	}

	initMySQLFlags(cmd)

	return cmd
}

func initMySQLFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Cluster name")
	cmd.MarkFlagRequired("name") //nolint:errcheck,gosec
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8080", "Everest endpoint URL")
	cmd.Flags().String("kubernetes-id", "", "Kubernetes cluster ID in Everest")
	cmd.MarkFlagRequired("kubernetes-id") //nolint:errcheck,gosec

	cmd.Flags().String("db.version", "latest", "MySQL version")

	cmd.Flags().Int("nodes", 1, "Number of cluster nodes")
	cmd.Flags().String("cpu", "1", "CPUs to assign to the cluster")
	cmd.Flags().String("memory", "2G", "Memory to assign to the cluster (4G, 512M, etc.)")
	cmd.Flags().String("disk", "15G", "MB of disk to assign to the cluster (8G, 500M, etc.)")

	cmd.Flags().Bool("external-access", false, "Make this cluster available outside of Kubernetes")
}

func initMySQLViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))                         //nolint:errcheck,gosec
	viper.BindPFlag("everest.endpoint", cmd.Flags().Lookup("everest.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("kubernetes-id", cmd.Flags().Lookup("kubernetes-id"))       //nolint:errcheck,gosec

	viper.BindPFlag("db.version", cmd.Flags().Lookup("db.version")) //nolint:errcheck,gosec

	viper.BindPFlag("nodes", cmd.Flags().Lookup("nodes"))   //nolint:errcheck,gosec
	viper.BindPFlag("cpu", cmd.Flags().Lookup("cpu"))       //nolint:errcheck,gosec
	viper.BindPFlag("memory", cmd.Flags().Lookup("memory")) //nolint:errcheck,gosec
	viper.BindPFlag("disk", cmd.Flags().Lookup("disk"))     //nolint:errcheck,gosec

	viper.BindPFlag("external-access", cmd.Flags().Lookup("external-access")) //nolint:errcheck,gosec
}

func parseMySQLConfig() (*provision.MySQLConfig, error) {
	c := &provision.MySQLConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
