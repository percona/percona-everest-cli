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

// Package uninstall ...
package uninstall //nolint:predeclared

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

// Cluster implements logic for the cluster command.
type Cluster struct {
	config     ClusterConfig
	kubeClient *kubernetes.Kubernetes
	l          *zap.SugaredLogger
}

type k8sCluster struct {
	// id stores ID of the Kubernetes cluster to be removed.
	id string
	// namespace stores everest namespace in the k8s cluster.
	namespace string
	// uid stores k8s UID of the namespace.
	uid string
}

// ClusterConfig stores configuration for the Cluster command.
type ClusterConfig struct {
	// KubeconfigPath is a path to a kubeconfig
	KubeconfigPath string `mapstructure:"kubeconfig"`
	// Namespace defines the namespace operators shall be installed to.
	Namespace string
	// AssumeYes is true when all questions can be skipped.
	AssumeYes bool `mapstructure:"assume-yes"`
	// Force is true when we shall not prompt for removal.
	Force bool
	// IgnoreK8sUnavailable is true when unavailable Kubernetes can be ignored.
	IgnoreK8sUnavailable bool `mapstructure:"ignore-kubernetes-unavailable"`
}

// NewCluster returns a new Cluster struct.
func NewCluster(c ClusterConfig, l *zap.SugaredLogger) (*Cluster, error) {
	kubeClient, err := kubernetes.New(c.KubeconfigPath, l)
	if err != nil {
		if !c.IgnoreK8sUnavailable {
			return nil, err
		}
	}

	cli := &Cluster{
		config:     c,
		kubeClient: kubeClient,
		l:          l,
	}
	return cli, nil
}

func (c *Cluster) runEverestWizard(ctx context.Context) error {
	pNamespace := &survey.Input{
		Message: "Please select namespace",
		Default: c.config.Namespace,
	}
	if err := survey.AskOne(
		pNamespace,
		&c.config.Namespace,
	); err != nil {
		return err
	}

	return nil
}

// Run runs the cluster command.
func (c *Cluster) Run(ctx context.Context) error {
	if err := c.runEverestWizard(ctx); err != nil {
		return err
	}

	if !c.config.AssumeYes {
		msg := `You are about to uninstall a Kubernetes cluster from Everest.
This will uninstall all monitoring resources deployed by Everest from the Kubernetes cluster. All other resources such as Database Clusters will not be affected.`
		fmt.Printf("\n%s\n\n", msg) //nolint:forbidigo
		confirm := &survey.Confirm{
			Message: "Are you sure you want to uninstall Everest?",
		}
		prompt := false
		if err := survey.AskOne(confirm, &prompt); err != nil {
			return err
		}

		if !prompt {
			c.l.Info("Exiting")
			return nil
		}
	}

	if err := c.uninstallK8sResources(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) uninstallK8sResources(ctx context.Context) error {
	c.l.Info("Deleting all Kubernetes monitoring resources in Kubernetes cluster")
	if err := c.kubeClient.DeleteAllMonitoringResources(ctx, c.config.Namespace); err != nil {
		return errors.Join(err, errors.New("could not uninstall monitoring resources from the Kubernetes cluster"))
	}

	return nil
}
