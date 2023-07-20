package install

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
)

type everestClientConnector interface {
	RegisterKubernetesCluster(
		ctx context.Context,
		body client.RegisterKubernetesClusterJSONRequestBody,
	) (*client.KubernetesCluster, error)
	CreateBackupStorage(
		ctx context.Context,
		body client.CreateBackupStorageJSONRequestBody,
	) (*client.BackupStorage, error)

	CreatePMMInstance(
		ctx context.Context,
		body client.CreatePMMInstanceJSONRequestBody,
	) (*client.PMMInstance, error)
	GetPMMInstance(
		ctx context.Context,
		pmmInstanceID string,
	) (*client.PMMInstance, error)
	ListPMMInstances(ctx context.Context) ([]client.PMMInstance, error)
}
