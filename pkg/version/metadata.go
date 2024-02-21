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
package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	version "github.com/Percona-Lab/percona-version-service/versionpb"
	goversion "github.com/hashicorp/go-version"
)

// RecommendedVersion holds recommended versions per component.
type RecommendedVersion struct {
	Catalog         *goversion.Version
	EverestOperator *goversion.Version
	OLM             *goversion.Version
	PG              *goversion.Version
	PSMDB           *goversion.Version
	PXC             *goversion.Version
}

// Metadata returns metadata from a given metadata URL.
func Metadata(ctx context.Context, versionMetadataURL string) (*version.MetadataResponse, error) {
	p, err := url.Parse(versionMetadataURL)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse version metadata URL"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.JoinPath("metadata/v1/everest").String(), nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not create requirements request"))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not retrieve requirements"))
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response from requirements endpoint http %d", res.StatusCode)
	}
	requirements := &version.MetadataResponse{}
	if err = json.NewDecoder(res.Body).Decode(requirements); err != nil {
		return nil, errors.Join(err, errors.New("could not decode from requirements"))
	}

	return requirements, nil
}

// RecommendedVersions returns recommended version information based on metadata.
func RecommendedVersions(meta *version.MetadataVersion) (*RecommendedVersion, error) {
	recVer := &RecommendedVersion{}

	if olm, ok := meta.Recommended["olm"]; ok {
		v, err := goversion.NewSemver(olm)
		if err != nil {
			return nil, errors.Join(err, fmt.Errorf("invalid OLM version %s", olm))
		}
		recVer.OLM = v
	}

	if catalog, ok := meta.Recommended["catalog"]; ok {
		v, err := goversion.NewSemver(catalog)
		if err != nil {
			return nil, errors.Join(err, fmt.Errorf("invalid catalog version %s", catalog))
		}
		recVer.Catalog = v
	}

	return recVer, nil
}
