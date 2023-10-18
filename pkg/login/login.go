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

// Package login holds the main logic for login command.
package login

import (
	"context"

	"github.com/percona/percona-everest-cli/pkg/cache"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"go.uber.org/zap"
)

// Login implements the main logic for commands.
type Login struct {
	config        LoginConfig
	everestClient everestClientConnector
	l             *zap.SugaredLogger
}

type (
	// LoginConfig stores configuration for the versions command.
	LoginConfig struct {
		Everest struct {
			// Endpoint stores URL to Everest.
			Endpoint string
		}
		// PersonalAccessToken stores personal access token.
		PersonalAccessToken string `mapstructure:"personal-access-token"`
	}
)

// NewLogin returns a new Login struct.
func NewLogin(c LoginConfig, everestClient everestClientConnector, l *zap.SugaredLogger) *Login {
	cli := &Login{
		config:        c,
		everestClient: everestClient,
		l:             l.With("component", "login"),
	}

	return cli
}

// Run runs the login command.
func (l *Login) Run(ctx context.Context) error {
	l.l.Info("Storing token in cache")
	token := &oidc.AccessTokenResponse{
		TokenType:   "bearer",
		AccessToken: l.config.PersonalAccessToken,
		ExpiresIn:   0,
	}
	err := cache.StoreToken(l.config.Everest.Endpoint, token)
	return err
}
