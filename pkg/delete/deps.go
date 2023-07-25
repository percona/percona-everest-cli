// Package delete deletes database clusters.
package delete //nolint:predeclared

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
)

type everestClientConnector interface {
	ListKubernetesClusters(ctx context.Context) ([]client.KubernetesCluster, error)
	UnregisterKubernetesCluster(
		ctx context.Context,
		kubernetesID string,
		body client.UnregisterKubernetesClusterJSONRequestBody,
	) error

	DeleteDBCluster(
		ctx context.Context,
		kubernetesID string,
		name string,
	) (*client.IoK8sApimachineryPkgApisMetaV1StatusV2, error)
}
