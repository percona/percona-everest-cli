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
// Package install holds the main logic for installation commands.

// Package version implements version reporting command to the end user.
package version

import (
	"encoding/json"
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"
)

const (
	devCatalogImage     = "docker.io/perconalab/everest-catalog:latest"
	releaseCatalogImage = "docker.io/percona/everest-catalog:%s"
	devManifestURL      = "https://raw.githubusercontent.com/percona/percona-everest-backend/main/deploy/quickstart-k8s.yaml"
	releaseManifestURL  = "https://raw.githubusercontent.com/percona/percona-everest-backend/v%s/deploy/quickstart-k8s.yaml"
)

var (
	// ProjectName is a component name, e.g. everestctl.
	ProjectName string //nolint:gochecknoglobals
	// Version is a component version e.g. v0.3.0-1-a93bef.
	Version string //nolint:gochecknoglobals
	// FullCommit is a git commit hash.
	FullCommit string //nolint:gochecknoglobals
	// CatalogImage is a image path for OLM catalog.
	catalogImage string //nolint:gochecknoglobals
)

// CatalogImage returns a catalog image needed for the build of everestctl
//
// for dev builds it returns https://raw.githubusercontent.com/percona/percona-everest-backend/main/deploy/quickstart-k8s.yaml
// for the release builds it returns https://raw.githubusercontent.com/percona/percona-everest-backend/vX.Y.Z/deploy/quickstart-k8s.yaml
func CatalogImage() string {
	catalogImage = devCatalogImage
	v, err := goversion.NewSemver(Version)
	if Version != "" && err == nil && v.Prerelease() == "" {
		catalogImage = fmt.Sprintf(releaseCatalogImage, Version)
	}
	return catalogImage
}

// ManifestURL returns a manifest URL to install everest
//
// for dev builds it returns everest-catalog:latest
// for the release it returns everest-catalog:X.Y.Z.
func ManifestURL() string {
	url := devManifestURL
	v, err := goversion.NewSemver(Version)
	if Version != "" && err == nil && v.Prerelease() == "" {
		url = fmt.Sprintf(releaseManifestURL, Version)
	}
	return url
}

// FullVersionInfo returns full version report.
func FullVersionInfo() string {
	out := []string{
		"ProjectName: " + ProjectName,
		"Version: " + Version,
		"FullCommit: " + FullCommit,
	}
	return strings.Join(out, "\n")
}

// FullVersionJSON returns version info as JSON.
func FullVersionJSON() (string, error) {
	res := map[string]string{
		"projectName": ProjectName,
		"version":     Version,
		"fullCommit":  FullCommit,
	}
	data, err := json.Marshal(res)
	return string(data), err
}
