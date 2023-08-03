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
package install

import (
	"context"
	"strings"
	"testing"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const apiSecret = "api-secret"

func TestOperators_validateConfig(t *testing.T) {
	t.Parallel()

	type fields struct {
		config         OperatorsConfig
		everestClient  everestClientConnector
		apiKeySecretID string
	}

	m := &mockEverestClientConnector{}
	m.Mock.On("GetPMMInstance", mock.Anything, "123").Return(&client.PMMInstance{ApiKeySecretId: apiSecret}, nil)
	m.Mock.On("GetPMMInstance", mock.Anything, "not-found").Return(nil, errors.New("not found"))

	tests := []struct {
		name               string
		fields             fields
		errContains        string
		wantAPIKeySecretID string
	}{
		{
			name:               "shall work with PMM instance-id",
			wantAPIKeySecretID: apiSecret,
			fields: fields{
				everestClient: m,
				config: OperatorsConfig{
					Monitoring: MonitoringConfig{
						Enable: true,
						PMM: &PMMConfig{
							InstanceID: "123",
						},
					},
				},
			},
		},
		{
			name:               "shall prefer PMM instance-id",
			wantAPIKeySecretID: apiSecret,
			fields: fields{
				everestClient: m,
				config: OperatorsConfig{
					Monitoring: MonitoringConfig{
						Enable: true,
						PMM: &PMMConfig{
							InstanceID: "123",
							Endpoint:   "http://localhost",
							Username:   "admin",
							Password:   "admin",
						},
					},
				},
			},
		},
		{
			name: "shall not throw on monitoring enabled with no API key or instance ID",
			fields: fields{
				config: OperatorsConfig{
					Monitoring: MonitoringConfig{
						Enable: true,
						PMM:    &PMMConfig{},
					},
				},
			},
		},
		{
			name:        "shall throw on instance ID not found",
			errContains: "could not retrieve PMM instance by its ID",
			fields: fields{
				everestClient: m,
				config: OperatorsConfig{
					Monitoring: MonitoringConfig{
						Enable: true,
						PMM: &PMMConfig{
							InstanceID: "not-found",
						},
					},
				},
			},
		},
	}

	l, err := zap.NewDevelopment()
	require.NoError(t, err)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o := &Operators{
				l:              l.Sugar(),
				config:         tt.fields.config,
				everestClient:  tt.fields.everestClient,
				apiKeySecretID: tt.fields.apiKeySecretID,
			}
			err := o.validateConfig(context.Background())
			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Operators.validateConfig() error = %v, errContains %v", err, tt.errContains)
			}

			if tt.wantAPIKeySecretID != "" && o.apiKeySecretID != tt.wantAPIKeySecretID {
				t.Errorf("Operators.apiKeySecretID = %v, expected %v", o.apiKeySecretID, tt.wantAPIKeySecretID)
			}
		})
	}
}
