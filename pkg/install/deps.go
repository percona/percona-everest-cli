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

// Package install ...
package install

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
)

//go:generate ../../bin/mockery --name=everestClientConnector --case=snake --inpackage --testonly

type everestClientConnector interface {
	CreateMonitoringInstance(
		ctx context.Context,
		body client.CreateMonitoringInstanceJSONRequestBody,
	) (*client.MonitoringInstance, error)
	GetMonitoringInstance(
		ctx context.Context,
		pmmInstanceID string,
	) (*client.MonitoringInstance, error)
	ListMonitoringInstances(ctx context.Context) ([]client.MonitoringInstance, error)

	SetKubernetesClusterMonitoring(
		ctx context.Context,
		body client.SetKubernetesClusterMonitoringJSONRequestBody,
	) error
}
