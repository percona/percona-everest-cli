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
// Package list holds the main logic for list commands.
package list

import (
	"context"
	"sort"
	"strings"

	goversion "github.com/hashicorp/go-version"
	everestv1alpha1 "github.com/percona/everest-operator/api/v1alpha1"
	"github.com/percona/percona-everest-backend/client"
	"go.uber.org/zap"
)

// Versions implements the main logic for commands.
type Versions struct {
	config        VersionsConfig
	everestClient everestClientConnector
	l             *zap.SugaredLogger
}

type (
	// VersionsConfig stores configuration for the versions command.
	VersionsConfig struct {
		KubernetesID string `mapstructure:"kubernetes-id"`
		Everest      EverestConfig

		// Type represents a database engine type.
		Type string
	}
)

type (
	// VersionsList stores a list of versions per engine type.
	VersionsList map[everestv1alpha1.EngineType]goversion.Collection
)

// String returns string result of database engines list.
func (v VersionsList) String() string {
	out := make([]string, 0, len(v))
	for engine, versions := range v {
		out = append(out, "-----", string(engine), "-----")

		sort.Sort(sort.Reverse(versions))
		for _, ver := range versions {
			out = append(out, ver.Original())
		}
	}

	return strings.Join(out, "\n")
}

// NewVersions returns a new Versions struct.
func NewVersions(c VersionsConfig, everestClient everestClientConnector, l *zap.SugaredLogger) *Versions {
	cli := &Versions{
		config:        c,
		everestClient: everestClient,
		l:             l.With("component", "list/versions"),
	}

	return cli
}

// Run runs the versions list command.
func (v *Versions) Run(ctx context.Context) (VersionsList, error) {
	dbEngines, err := v.everestClient.ListDatabaseEngines(ctx, v.config.KubernetesID)
	if err != nil {
		return nil, err
	}

	if dbEngines.Items == nil {
		res := make(VersionsList)
		return res, nil
	}

	return v.parseVersions(*dbEngines.Items)
}

func (v *Versions) parseVersions(items []client.DatabaseEngine) (VersionsList, error) {
	res := make(VersionsList)
	for _, db := range items {
		if v.checkIfSkip(db) {
			continue
		}

		engineType := everestv1alpha1.EngineType(db.Spec.Type)
		if _, ok := res[engineType]; !ok {
			res[engineType] = make(goversion.Collection, 0, len(*db.Status.AvailableVersions.Engine))
		}

		for version := range *db.Status.AvailableVersions.Engine {
			ver, err := goversion.NewVersion(version)
			if err != nil {
				return nil, err
			}
			res[engineType] = append(res[engineType], ver)
		}
	}

	return res, nil
}

func (v *Versions) checkIfSkip(db client.DatabaseEngine) bool {
	if db.Spec == nil {
		return true
	}

	if v.config.Type != "" && db.Spec.Type != v.config.Type {
		return true
	}

	if db.Status == nil || db.Status.AvailableVersions == nil || db.Status.AvailableVersions.Engine == nil {
		return true
	}

	return false
}
