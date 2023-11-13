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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// NewClusterCmd returns a new cluster command.
func NewClusterCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Deprecated. Please use everestctl uninstall instead",
		Run: func(cmd *cobra.Command, args []string) {
			initClusterViperFlags(cmd)
			l.Info("delete cluster is deprecated. Please use everestctl uninstall")
			os.Exit(1)
		},
	}

	initClusterFlags(cmd)

	return cmd
}

func initClusterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "Path to a kubeconfig")
	cmd.Flags().String("namespace", "percona-everest", "Namespace into which Percona Everest components are deployed to")
	cmd.Flags().String("name", "", "Kubernetes cluster name in Everest")
	cmd.Flags().BoolP("assume-yes", "y", false, "Assume yes to all questions")
	cmd.Flags().BoolP("force", "f", false, "Force removal in case there are database clusters running")
	cmd.Flags().Bool("ignore-kubernetes-unavailable", false, "Remove cluster even if Kubernetes is not available")
}

func initClusterViperFlags(cmd *cobra.Command) {
	viper.BindEnv("kubeconfig")                                     //nolint:errcheck,gosec
	viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig")) //nolint:errcheck,gosec
	viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))   //nolint:errcheck,gosec
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))             //nolint:errcheck,gosec
	viper.BindPFlag("assume-yes", cmd.Flags().Lookup("assume-yes")) //nolint:errcheck,gosec
	viper.BindPFlag("force", cmd.Flags().Lookup("force"))           //nolint:errcheck,gosec
	viper.BindPFlag(                                                //nolint:errcheck,gosec
		"ignore-kubernetes-unavailable", cmd.Flags().Lookup("ignore-kubernetes-unavailable"),
	)
}
