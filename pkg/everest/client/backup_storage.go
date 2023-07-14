package client

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
)

// CreateBackupStorage creates a new backup storage.
func (e *Everest) CreateBackupStorage(
	ctx context.Context,
	body client.CreateBackupStorageJSONRequestBody,
) (*client.BackupStorage, error) {
	res := &client.BackupStorage{}
	err := do(
		ctx, e.cl.CreateBackupStorage,
		body, res, errors.New("cannot create backup storage due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
