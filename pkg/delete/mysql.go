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

// Package delete ...
package delete //nolint:predeclared

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"go.uber.org/zap"
)

// MySQL implements logic for the MySQL command.
type MySQL struct {
	config        MySQLConfig
	everestClient everestClientConnector
	l             *zap.SugaredLogger
}

// MySQLConfig stores configuration for the MySQL command.
type MySQLConfig struct {
	Name string

	Everest struct {
		// Endpoint stores URL to Everest.
		Endpoint string
	}

	// Force is true when we shall not prompt for removal.
	Force bool
}

// NewMySQL returns a new MySQL struct.
func NewMySQL(c MySQLConfig, everestClient everestClientConnector, l *zap.SugaredLogger) *MySQL {
	cli := &MySQL{
		config:        c,
		everestClient: everestClient,
		l:             l.With("component", "delete/mysql"),
	}

	return cli
}

// Run runs the MySQL command.
func (m *MySQL) Run(ctx context.Context) error {
	if !m.config.Force {
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("Are you sure you want to remove the %q database cluster?", m.config.Name),
		}
		prompt := false
		err := survey.AskOne(confirm, &prompt)
		if err != nil {
			return err
		}

		if !prompt {
			m.l.Info("Exiting")
			return nil
		}
	}

	m.l.Infof("Deleting %q cluster", m.config.Name)
	_, err := m.everestClient.DeleteDBCluster(ctx, m.config.Name)
	if err != nil {
		return err
	}

	m.l.Infof("Cluster %q successfully deleted", m.config.Name)

	return nil
}
