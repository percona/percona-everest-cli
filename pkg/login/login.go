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
	"time"

	"github.com/percona/percona-everest-cli/pkg/cache"
	"github.com/pkg/browser"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
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

// TODO: figure out what to do about this. Maybe store in cache or make it random?
var key = []byte("random_key")

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
	// TODO: figure out how to get url
	issuer := "http://localhost:8080"

	if l.config.PersonalAccessToken != "" {
		l.l.Info("Storing token in cache")
		token := &oidc.AccessTokenResponse{
			TokenType:   "bearer",
			AccessToken: l.config.PersonalAccessToken,
			ExpiresIn:   0,
		}
		err := cache.StoreToken(issuer, token)
		return err
	}

	// TODO: figure out how to get client ID. Maybe registration endpoint?
	clientID := "235018918891159554@everest-app"
	clientSecret := ""
	// TODO: what scopes do we need?
	scopes := []string{"openid"}

	cookieHandler := httphelper.NewCookieHandler(key, key, httphelper.WithUnsecure())
	options := []rp.Option{rp.WithPKCE(cookieHandler)}
	// options := []rp.Option{}

	provider, err := rp.NewRelyingPartyOIDC(issuer, clientID, clientSecret, "", scopes, options...)
	if err != nil {
		l.l.Fatalf("error creating provider %s", err.Error())
	}

	l.l.Info("starting device authorization flow")
	resp, err := rp.DeviceAuthorization(scopes, provider)
	if err != nil {
		l.l.Fatal(err)
	}
	l.l.Info("resp", resp)

	// TODO: figure out a better way. This prints unnecessary errors to stderr.
	if err := browser.OpenURL(resp.VerificationURIComplete); err != nil {
		l.l.Errorf("Could not open browser. Error: %s", err)
	}
	l.l.Infof("Please browse to %s and enter code %s\n", resp.VerificationURI, resp.UserCode)

	l.l.Info("start polling")
	token, err := rp.DeviceAccessToken(ctx, resp.DeviceCode, time.Duration(resp.Interval)*time.Second, provider)
	if err != nil {
		l.l.Fatal(err)
	}
	l.l.Infof("successfully obtained token: %v", token)

	l.l.Info("Storing token in cache")
	err = cache.StoreToken(issuer, token)

	l.l.Info("Sucessfully signed in")

	return err
}
