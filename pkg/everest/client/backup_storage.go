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
	bs := &client.BackupStorage{}
	err := do(
		ctx, e.cl.CreateBackupStorage,
		body, bs, errors.New("cannot create backup storage due to Everest error"),
	)
	return bs, err
}
