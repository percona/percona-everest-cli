package client

import (
	"context"
	"net/http"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
)

// CreatePMMInstance creates a new PMM instance.
func (e *Everest) CreatePMMInstance(
	ctx context.Context,
	body client.CreatePMMInstanceJSONRequestBody,
) (*client.PMMInstance, error) {
	res := &client.PMMInstance{}
	err := do(
		ctx, e.cl.CreatePMMInstance,
		body, res, errors.New("cannot create PMM instance due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetPMMInstance retrieves a PMM instance by its ID.
func (e *Everest) GetPMMInstance(ctx context.Context, pmmInstanceID string) (*client.PMMInstance, error) {
	res := &client.PMMInstance{}
	err := do(
		ctx, e.cl.GetPMMInstance,
		pmmInstanceID, res, errors.New("cannot retrieve PMM instance due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListPMMInstances lists PMM instances.
func (e *Everest) ListPMMInstances(ctx context.Context) ([]client.PMMInstance, error) {
	res := []client.PMMInstance{}
	err := do(
		ctx, func(
			ctx context.Context,
			_ struct{},
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.ListPMMInstances(ctx, r...)
		},
		struct{}{}, &res, errors.New("cannot list PMM instances due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
