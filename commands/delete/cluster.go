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

// Package delete holds logic for delete command.
package delete //nolint:predeclared

import (
	"fmt"
	"os"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/commands/common"
	"github.com/percona/percona-everest-cli/pkg/delete"
	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
)

// NewClusterCmd returns a new cluster command.
func NewClusterCmd(l *zap.SugaredLogger) *cobra.Command {
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
				l.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			op, err := delete.NewCluster(*c, everestClConnector, l)
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}

			if err := op.Run(cmd.Context()); err != nil {
				if !errors.Is(err, common.ErrExitWithError) {
					l.Error(err)
				}
				os.Exit(1)
			}
		},
	}

	initClusterFlags(cmd)

	return cmd
}

func initClusterFlags(cmd *cobra.Command) {
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8080", "Everest endpoint URL")
	cmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "Path to a kubeconfig")
	cmd.Flags().String("name", "", "Kubernetes cluster name in Everest")
	cmd.Flags().BoolP("force", "f", false, "Force removal in case there are database clusters running")
}

func initClusterViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("everest.endpoint", cmd.Flags().Lookup("everest.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig"))             //nolint:errcheck,gosec
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))                         //nolint:errcheck,gosec
	viper.BindPFlag("force", cmd.Flags().Lookup("force"))                       //nolint:errcheck,gosec
}

func parseClusterConfig() (*delete.ClusterConfig, error) {
	c := &delete.ClusterConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
