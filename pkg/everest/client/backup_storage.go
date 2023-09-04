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

// Package client ...
package client

import (
	"context"
	"errors"

	"github.com/percona/percona-everest-backend/client"
)

// CreateBackupStorage creates a new backup storage.
func (e *Everest) CreateBackupStorage(
	ctx context.Context,
	body client.CreateBackupStorageJSONRequestBody,
) (*client.BackupStorage, error) {
	res := &client.BackupStorage{}
	err := makeRequest(
		ctx, e.cl.CreateBackupStorage,
		body, res, errors.New("cannot create backup storage due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
