package client

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
)

// RegisterKubernetesCluster registers a new Kubernetes cluster.
func (e *Everest) RegisterKubernetesCluster(
	ctx context.Context,
	body client.RegisterKubernetesClusterJSONRequestBody,
) (*client.KubernetesCluster, error) {
	cluster := &client.KubernetesCluster{}
	err := do(
		ctx, e.cl.RegisterKubernetesCluster,
		body, cluster, errors.New("cannot register Kubernetes cluster due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

// CreateBackupStorage creates a new backup storage.
func (e *Everest) CreateBackupStorage(
	ctx context.Context,
	body client.CreateBackupStorageJSONRequestBody,
) (*client.BackupStorage, error) {
	bs := &client.BackupStorage{}
	err := do(
		ctx, e.cl.CreateBackupStorage,
		body, bs, errors.New("cannot create backup storage due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return bs, nil
}
