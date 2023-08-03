// percona-everest-backend
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
// Package delete deletes database clusters.
package delete //nolint:predeclared

import (
	"context"

	"github.com/percona/percona-everest-backend/client"
)

type everestClientConnector interface {
	ListKubernetesClusters(ctx context.Context) ([]client.KubernetesCluster, error)
	UnregisterKubernetesCluster(
		ctx context.Context,
		kubernetesID string,
		body client.UnregisterKubernetesClusterJSONRequestBody,
	) error

	DeleteDBCluster(
		ctx context.Context,
		kubernetesID string,
		name string,
	) (*client.IoK8sApimachineryPkgApisMetaV1StatusV2, error)
}
