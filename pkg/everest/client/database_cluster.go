package client

import (
	"context"
	"net/http"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
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
