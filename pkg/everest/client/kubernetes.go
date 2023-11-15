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

// SetKubernetesClusterMonitoring configures Kubernetes cluster monitoring.
func (e *Everest) SetKubernetesClusterMonitoring(
	ctx context.Context,
	body client.SetKubernetesClusterMonitoringJSONRequestBody,
) error {
	res := &struct{}{}
	err := makeRequest(
		ctx, func(
			ctx context.Context,
			r ...client.RequestEditorFn,
		) (*http.Response, error) {
			return e.cl.SetKubernetesClusterMonitoring(ctx, body, r...)
		},
		kubernetesID, res, errors.New("cannot configure Kubernetes cluster monitoring due to Everest error"),
	)
	return err
}
