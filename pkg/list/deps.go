package list

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
)

type everestClientConnector interface {
	ListDatabaseEngines(ctx context.Context, kubernetesID string) (*client.DatabaseEngineList, error)
}
