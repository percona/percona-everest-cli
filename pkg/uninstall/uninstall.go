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
package uninstall

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

const (
	everestNamespace = "percona-everest"
)

// Uninstall implements logic for the cluster command.
type Uninstall struct {
	config     Config
	kubeClient *kubernetes.Kubernetes
	l          *zap.SugaredLogger
}

// Config stores configuration for the Uninstall command.
type Config struct {
	// KubeconfigPath is a path to a kubeconfig
	KubeconfigPath string `mapstructure:"kubeconfig"`
	// Namespaces defines namespaces that everest can operate in.
	Namespaces []string `mapstructure:"namespace"`
	// AssumeYes is true when all questions can be skipped.
	AssumeYes bool `mapstructure:"assume-yes"`
	// Force is true when we shall not prompt for removal.
	Force bool
}

// NewUninstall returns a new Uninstall struct.
func NewUninstall(c Config, l *zap.SugaredLogger) (*Uninstall, error) {
	kubeClient, err := kubernetes.New(c.KubeconfigPath, l)
	if err != nil {
		return nil, err
	}

	cli := &Uninstall{
		config:     c,
		kubeClient: kubeClient,
		l:          l,
	}
	return cli, nil
}

func (u *Uninstall) runEverestWizard(ctx context.Context) error {
	if !u.config.AssumeYes {
		namespaces, err := u.kubeClient.GetPersistedNamespaces(ctx, everestNamespace)
		if err != nil {
			return err
		}
		if !u.config.AssumeYes {
			pNamespace := &survey.MultiSelect{
				Message: "Please select namespaces",
				Options: namespaces,
			}
			if err := survey.AskOne(
				pNamespace,
				&u.config.Namespaces,
				survey.WithValidator(survey.MinItems(1)),
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// Run runs the cluster command.
func (u *Uninstall) Run(ctx context.Context) error {
	if err := u.runEverestWizard(ctx); err != nil {
		return err
	}
	if len(u.config.Namespaces) == 0 {
		return errors.New("namespace list is empty")
	}

	if !u.config.AssumeYes {
		msg := `You are about to uninstall Everest from the Kubernetes cluster.
This will uninstall Everest and all monitoring resources deployed by it. All other resources such as Databases and Database Backups will not be affected.`
		fmt.Printf("\n%s\n\n", msg) //nolint:forbidigo
		confirm := &survey.Confirm{
			Message: "Are you sure you want to uninstall Everest?",
		}
		prompt := false
		if err := survey.AskOne(confirm, &prompt); err != nil {
			return err
		}

		if !prompt {
			u.l.Info("Exiting")
			return nil
		}
	}

	if err := u.uninstallK8sResources(ctx); err != nil {
		return err
	}
	if err := u.kubeClient.DeleteEverest(ctx, everestNamespace); err != nil {
		return err
	}

	return nil
}

func (u *Uninstall) uninstallK8sResources(ctx context.Context) error {
	u.l.Info("Deleting all Kubernetes monitoring resources in Kubernetes cluster")
	for _, namespace := range u.config.Namespaces {
		namespace := namespace
		if err := u.kubeClient.DeleteAllMonitoringResources(ctx, namespace); err != nil {
			return errors.Join(err, errors.New("could not uninstall monitoring resources from the Kubernetes cluster"))
		}
	}

	return nil
}
