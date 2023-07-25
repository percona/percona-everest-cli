package client

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
)

// ListDatabaseEngines lists database engines.
func (e *Everest) ListDatabaseEngines(ctx context.Context, kubernetesID string) (*client.DatabaseEngineList, error) {
	res := &client.DatabaseEngineList{}
	err := makeRequest(
		ctx, e.cl.ListDatabaseEngines,
		kubernetesID, res, errors.New("cannot list database engines due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
