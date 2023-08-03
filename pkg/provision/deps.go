// Package provision provisions database clusters.
package provision

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
)

type everestClientConnector interface {
	CreateDBCluster(
		ctx context.Context,
		kubernetesID string,
		body client.CreateDatabaseClusterJSONRequestBody,
	) (*client.DatabaseCluster, error)
}
