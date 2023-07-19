package client

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
)

// RegisterKubernetesCluster registers a new Kubernetes cluster.
func (e *Everest) RegisterKubernetesCluster(
	ctx context.Context,
	body client.RegisterKubernetesClusterJSONRequestBody,
) (*client.KubernetesCluster, error) {
	res := &client.KubernetesCluster{}
	err := do(
		ctx, e.cl.RegisterKubernetesCluster,
		body, res, errors.New("cannot register Kubernetes cluster due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
