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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/percona/percona-everest-backend/client"
)

// ListDatabaseEngines lists database engines.
func (e *Everest) ListDatabaseEngines(ctx context.Context) (*client.DatabaseEngineList, error) {
	ret := &client.DatabaseEngineList{}
	res, err := e.cl.ListDatabaseEngines(ctx)
	if err != nil {
		return ret, errors.Join(err, errors.New("cannot list database engines due to Everest error"))
	}
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return ret, processErrorResponse(res, errors.New("cannot list database engines doe to Everest error"))
	}
	err = json.NewDecoder(res.Body).Decode(ret)
	return ret, err
}
