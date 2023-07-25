// Package delete deletes database clusters.
package delete //nolint:predeclared

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
)

type everestClientConnector interface {
	DeleteDBCluster(
		ctx context.Context,
		kubernetesID string,
		name string,
	) (*client.IoK8sApimachineryPkgApisMetaV1StatusV2, error)
}
