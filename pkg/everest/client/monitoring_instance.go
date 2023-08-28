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

// CreateMonitoringInstance creates a new monitoring instance.
func (e *Everest) CreateMonitoringInstance(
	ctx context.Context,
	body client.CreateMonitoringInstanceJSONRequestBody,
) (*client.MonitoringInstance, error) {
	res := &client.MonitoringInstance{}
	err := makeRequest(
		ctx, e.cl.CreateMonitoringInstance,
		body, res, errors.New("cannot create monitoring instance due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetMonitoringInstance retrieves a monitoring instance by its name.
func (e *Everest) GetMonitoringInstance(ctx context.Context, name string) (*client.MonitoringInstance, error) {
	res := &client.MonitoringInstance{}
	err := makeRequest(
		ctx, e.cl.GetMonitoringInstance,
		name, res, errors.New("cannot retrieve monitoring instance due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListMonitoringInstances lists monitoring instances.
func (e *Everest) ListMonitoringInstances(ctx context.Context) ([]client.MonitoringInstance, error) {
	res := []client.MonitoringInstance{}
	err := makeRequest(
		ctx, func(
			ctx context.Context,
			_ struct{},
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.ListMonitoringInstances(ctx, r...)
		},
		struct{}{}, &res, errors.New("cannot list monitoring instances due to Everest error"),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
