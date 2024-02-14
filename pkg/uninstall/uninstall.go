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
	"fmt"
	"time"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	everestv1alpha1 "github.com/percona/everest-operator/api/v1alpha1"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/percona/percona-everest-cli/pkg/install"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
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

// Run runs the cluster command.
func (u *Uninstall) Run(ctx context.Context) error {
	if !u.config.AssumeYes {
		msg := `You are about to uninstall Everest from the Kubernetes cluster.
This will uninstall Everest and all its components from the cluster.`
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

	dbsExist, err := u.dbsExist(ctx)
	if err != nil {
		return err
	}
	if dbsExist {
		if !u.config.Force {
			confirm := &survey.Confirm{
				Message: "There are still database clusters managed by Everest. Do you want to delete them?",
			}
			prompt := false
			if err := survey.AskOne(confirm, &prompt); err != nil {
				return err
			}

			if !prompt {
				u.l.Info("Can't proceed without deleting database clusters")
				return nil
			}
		}

		if err := u.deleteDBs(ctx); err != nil {
			return err
		}
	}

	// BackupStorages have finalizers, so we need to delete them first
	if err := u.deleteBackupStorages(ctx); err != nil {
		return err
	}

	if err := u.deleteDBNamespaces(ctx); err != nil {
		return err
	}

	// There are no resources with finalizers in the monitoring namespace, so
	// we can delete it directly
	if err := u.deleteNamespaces(ctx, []string{install.MonitoringNamespace}); err != nil {
		return err
	}

	// All resources with finalizers in the system namespace (DBCs and
	// BackupStorages) have already been deleted, so we can delete the
	// namespace directly
	if err := u.deleteNamespaces(ctx, []string{install.SystemNamespace}); err != nil {
		return err
	}

	// There are no resources with finalizers in the monitoring namespace, so
	// we can delete it directly
	if err := u.deleteNamespaces(ctx, []string{kubernetes.OLMNamespace}); err != nil {
		return err
	}

	return nil
}

func (u *Uninstall) getDBs(ctx context.Context) (map[string]*everestv1alpha1.DatabaseClusterList, error) {
	namespaces, err := u.kubeClient.GetDBNamespaces(ctx, install.SystemNamespace)
	if err != nil {
		return nil, err
	}

	allDBs := map[string]*everestv1alpha1.DatabaseClusterList{}
	for _, ns := range namespaces {
		dbs, err := u.kubeClient.ListDatabaseClusters(ctx, ns)
		if err != nil {
			return nil, err
		}

		allDBs[ns] = dbs
	}

	return allDBs, nil
}

func (u *Uninstall) dbsExist(ctx context.Context) (bool, error) {
	allDBs, err := u.getDBs(ctx)
	if err != nil {
		return false, err
	}

	exist := false
	for ns, dbs := range allDBs {
		if len(dbs.Items) == 0 {
			continue
		}

		exist = true
		u.l.Warnf("Database clusters in namespace '%s':", ns)
		for _, db := range dbs.Items {
			u.l.Warnf("  - %s", db.Name)
		}
	}

	return exist, nil
}

func (u *Uninstall) deleteDBs(ctx context.Context) error {
	allDBs, err := u.getDBs(ctx)
	if err != nil {
		return err
	}

	for ns, dbs := range allDBs {
		for _, db := range dbs.Items {
			u.l.Infof("Deleting database cluster '%s' in namespace '%s'", db.Name, ns)
			if err := u.kubeClient.DeleteDatabaseCluster(ctx, ns, db.Name); err != nil {
				return err
			}
		}
	}

	// Wait for all database clusters to be deleted, or timeout after 5 minutes.
	u.l.Info("Waiting for database clusters to be deleted")
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, false, func(ctx context.Context) (bool, error) {
		allDBs, err := u.getDBs(ctx)
		if err != nil {
			return false, err
		}

		for _, dbs := range allDBs {
			if len(dbs.Items) != 0 {
				return false, nil
			}
		}

		u.l.Info("All database clusters have been deleted")

		return true, nil
	})
}

func (u *Uninstall) deleteNamespaces(ctx context.Context, namespaces []string) error {
	for _, ns := range namespaces {
		u.l.Infof("Deleting namespace '%s'", ns)
		if err := u.kubeClient.DeleteNamespace(ctx, ns); err != nil {
			return err
		}
	}

	// Wait for all namespaces to be deleted, or timeout after 5 minutes.
	u.l.Infof("Waiting for namespace(s) '%s' to be deleted", strings.Join(namespaces, "', '"))
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, false, func(ctx context.Context) (bool, error) {
		for _, ns := range namespaces {
			_, err := u.kubeClient.GetNamespace(ctx, ns)
			if err != nil && !k8serrors.IsNotFound(err) {
				return false, err
			}
			if err == nil {
				return false, nil
			}
		}

		u.l.Infof("Namespace(s) '%s' have been deleted", strings.Join(namespaces, "', '"))

		return true, nil
	})
}

func (u *Uninstall) deleteDBNamespaces(ctx context.Context) error {
	namespaces, err := u.kubeClient.GetDBNamespaces(ctx, install.SystemNamespace)
	if err != nil {
		return err
	}

	return u.deleteNamespaces(ctx, namespaces)
}

func (u *Uninstall) deleteBackupStorages(ctx context.Context) error {
	storages, err := u.kubeClient.ListBackupStorages(ctx, install.SystemNamespace)
	if err != nil {
		return err
	}

	if len(storages.Items) == 0 {
		return nil
	}

	for _, storage := range storages.Items {
		u.l.Infof("Deleting backup storage '%s'", storage.Name)
		if err := u.kubeClient.DeleteBackupStorage(ctx, install.SystemNamespace, storage.Name); err != nil {
			return err
		}
	}

	// Wait for all backup storages to be deleted, or timeout after 5 minutes.
	u.l.Infof("Waiting for backup storages to be deleted")
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, false, func(ctx context.Context) (bool, error) {
		storages, err := u.kubeClient.ListBackupStorages(ctx, install.SystemNamespace)
		if err != nil {
			return false, err
		}

		if len(storages.Items) != 0 {
			return false, nil
		}

		u.l.Info("All backup storages have been deleted")

		return true, nil
	})
}
