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

// RegisterKubernetesCluster registers a new Kubernetes cluster.
func (e *Everest) RegisterKubernetesCluster(
	ctx context.Context,
	body client.RegisterKubernetesClusterJSONRequestBody,
) (*client.KubernetesCluster, error) {
	res := &client.KubernetesCluster{}
	err := makeRequest(
		ctx, e.cl.RegisterKubernetesCluster,
		body, res, errors.New("cannot register Kubernetes cluster due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListKubernetesClusters lists all Kubernetes clusters.
func (e *Everest) ListKubernetesClusters(ctx context.Context) ([]client.KubernetesCluster, error) {
	res := []client.KubernetesCluster{}
	err := makeRequest(
		ctx, func(
			ctx context.Context,
			_ struct{},
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.ListKubernetesClusters(ctx, r...)
		},
		struct{}{}, &res, errors.New("cannot list Kubernetes clusters due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// UnregisterKubernetesCluster unregisters a Kubernetes cluster.
func (e *Everest) UnregisterKubernetesCluster(
	ctx context.Context,
	kubernetesID string,
	body client.UnregisterKubernetesClusterJSONRequestBody,
) error {
	res := &struct{}{}
	err := makeRequest(
		ctx, func(
			ctx context.Context,
			kubernetesID string,
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.UnregisterKubernetesCluster(ctx, kubernetesID, body, r...)
		},
		kubernetesID, res, errors.New("cannot unregister Kubernetes cluster due to Everest error"),
	)
	return err
}

// SetKubernetesClusterMonitoring configures Kubernetes cluster monitoring.
func (e *Everest) SetKubernetesClusterMonitoring(
	ctx context.Context,
	kubernetesID string,
	body client.SetKubernetesClusterMonitoringJSONRequestBody,
) error {
	res := &struct{}{}
	err := makeRequest(
		ctx, func(
			ctx context.Context,
			kubernetesID string,
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.SetKubernetesClusterMonitoring(ctx, kubernetesID, body, r...)
		},
		kubernetesID, res, errors.New("cannot configure Kubernetes cluster monitoring due to Everest error"),
	)
	return err
}
