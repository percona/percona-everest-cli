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
	"net/http"

	"github.com/percona/percona-everest-backend/client"
)

// CreateDBCluster creates a new database cluster.
func (e *Everest) CreateDBCluster(
	ctx context.Context,
	kubernetesID string,
	body client.CreateDatabaseClusterJSONRequestBody,
) (*client.DatabaseCluster, error) {
	res := &client.DatabaseCluster{}
	err := makeRequest(
		ctx,
		func(
			ctx context.Context,
			body client.CreateDatabaseClusterJSONRequestBody,
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.CreateDatabaseCluster(ctx, kubernetesID, body, r...)
		},
		body, res, errors.New("cannot create database cluster due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DeleteDBCluster deletes a database cluster.
func (e *Everest) DeleteDBCluster(
	ctx context.Context,
	kubernetesID string,
	name string,
) (*client.IoK8sApimachineryPkgApisMetaV1StatusV2, error) {
	res := &client.IoK8sApimachineryPkgApisMetaV1StatusV2{}
	err := makeRequest(
		ctx,
		func(
			ctx context.Context,
			_ struct{},
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.DeleteDatabaseCluster(ctx, kubernetesID, name, r...)
		},
		struct{}{}, res, errors.New("cannot delete database cluster due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
