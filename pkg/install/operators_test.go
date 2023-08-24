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
package install

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/percona/percona-everest-backend/client"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	iName    = "monitoring-instance"
	iDefault = "default"
)

func TestOperators_resolveMonitoringInstanceName(t *testing.T) {
	t.Parallel()

	l, err := zap.NewDevelopment()
	require.NoError(t, err)

	t.Run("shall work with monitoring disabled", func(t *testing.T) {
		t.Parallel()

		m := &mockEverestClientConnector{}
		defer m.AssertExpectations(t)

		o := &Operators{
			l: l.Sugar(),
			config: OperatorsConfig{
				Monitoring: MonitoringConfig{
					Enable: false,
				},
			},
			everestClient:          m,
			monitoringInstanceName: iDefault,
		}

		err := o.resolveMonitoringInstanceName(context.Background())
		require.NoError(t, err)
		require.Equal(t, iDefault, o.monitoringInstanceName)
	})

	t.Run("shall work with monitoring instance name", func(t *testing.T) {
		t.Parallel()

		m := &mockEverestClientConnector{}
		m.Mock.On("GetMonitoringInstance", mock.Anything, "123").Return(&client.MonitoringInstance{Name: iName}, nil)
		defer m.AssertExpectations(t)

		o := &Operators{
			l: l.Sugar(),
			config: OperatorsConfig{
				Monitoring: MonitoringConfig{
					Enable:       true,
					InstanceName: "123",
				},
			},
			everestClient: m,
		}

		err := o.resolveMonitoringInstanceName(context.Background())
		require.NoError(t, err)
		require.Equal(t, iName, o.monitoringInstanceName)
	})

	t.Run("shall fail with monitoring instance name not found", func(t *testing.T) {
		t.Parallel()

		m := &mockEverestClientConnector{}
		m.Mock.On("GetMonitoringInstance", mock.Anything, "123").Return(nil, errors.New("not-found"))
		defer m.AssertExpectations(t)

		o := &Operators{
			l: l.Sugar(),
			config: OperatorsConfig{
				Monitoring: MonitoringConfig{
					Enable:       true,
					InstanceName: "123",
				},
			},
			everestClient: m,
		}

		err := o.resolveMonitoringInstanceName(context.Background())
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "not-found"))
	})

	t.Run("shall prefer monitoring instance name", func(t *testing.T) {
		t.Parallel()

		m := &mockEverestClientConnector{}
		m.Mock.On("GetMonitoringInstance", mock.Anything, "123").Return(&client.MonitoringInstance{Name: iName}, nil)
		defer m.AssertExpectations(t)

		o := &Operators{
			l: l.Sugar(),
			config: OperatorsConfig{
				Monitoring: MonitoringConfig{
					Enable:       true,
					InstanceName: "123",
					PMM: &PMMConfig{
						Endpoint: "http://localhost",
						Username: "admin",
						Password: "admin",
					},
				},
			},
			everestClient: m,
		}

		err := o.resolveMonitoringInstanceName(context.Background())
		require.NoError(t, err)
		require.Equal(t, iName, o.monitoringInstanceName)
	})

	t.Run("shall fail without new instance name defined when creating a new instance", func(t *testing.T) {
		t.Parallel()

		m := &mockEverestClientConnector{}
		defer m.AssertExpectations(t)

		o := &Operators{
			l: l.Sugar(),
			config: OperatorsConfig{
				Monitoring: MonitoringConfig{
					Enable: true,
				},
			},
			everestClient: m,
		}

		err := o.resolveMonitoringInstanceName(context.Background())
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "monitoring.new-instance-name is required"))
	})

	t.Run("shall create a new PMM instance", func(t *testing.T) {
		t.Parallel()

		m := &mockEverestClientConnector{}
		m.Mock.On("CreateMonitoringInstance", mock.Anything, client.MonitoringInstanceCreateParams{
			Type: client.MonitoringInstanceCreateParamsTypePmm,
			Name: "new-instance",
			Url:  "http://monitoring-url",
			Pmm: &client.PMMMonitoringInstanceSpec{
				User:     "user",
				Password: "pass",
			},
		}).Return(&client.MonitoringInstance{}, nil)
		defer m.AssertExpectations(t)

		o := &Operators{
			l: l.Sugar(),
			config: OperatorsConfig{
				Monitoring: MonitoringConfig{
					Enable:          true,
					NewInstanceName: "new-instance",
					PMM: &PMMConfig{
						Endpoint: "http://monitoring-url",
						Username: "user",
						Password: "pass",
					},
				},
			},
			everestClient: m,
		}

		err := o.resolveMonitoringInstanceName(context.Background())
		require.NoError(t, err)
		require.Equal(t, "new-instance", o.monitoringInstanceName)
	})
}
