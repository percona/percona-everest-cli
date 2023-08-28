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

// Package list ...
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

// NewVersionsCmd returns a new versions command.
func NewVersionsCmd(l *zap.SugaredLogger) *cobra.Command {
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
				l.Error(err)
				os.Exit(1)
			}

			everestClConnector := everestClient.NewEverest(everestCl)
			command := list.NewVersions(*c, everestClConnector, l)
			res, err := command.Run(cmd.Context())
			if err != nil {
				output.PrintError(err, l)
				os.Exit(1)
			}

			output.PrintOutput(cmd, l, res)
		},
	}

	initVersionsFlags(cmd)

	return cmd
}

func initVersionsFlags(cmd *cobra.Command) {
	cmd.Flags().String("everest.endpoint", "http://127.0.0.1:8080", "Everest endpoint URL")
	cmd.Flags().String("kubernetes-id", "", "Kubernetes cluster ID")
	cmd.MarkFlagRequired("kubernetes-id") //nolint:errcheck,gosec

	cmd.Flags().String("type", "", "Filter by database engine type")
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
