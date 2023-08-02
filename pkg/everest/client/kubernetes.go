package client

import (
	"context"
	"net/http"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
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
	var res interface{}
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
