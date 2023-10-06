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

// Package cache holds common logic for cache.
package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/zitadel/oidc/v2/pkg/oidc"
)

// File holds cached data and in the format as stored on the filesystem.
type File struct {
	// Credentials hold cached tokens. The key is the full url to the issuer.
	Credentials map[string]*oidc.AccessTokenResponse `json:"credentials,omitempty"`
}

// ReadFile returns cache stored in a file.
func ReadFile() (*File, error) {
	cache, err := readCacheFile()
	if err != nil {
		return nil, err
	}

	return cache, nil
}

// StoreToken stores the provided token in file cache.
func StoreToken(issuer string, token *oidc.AccessTokenResponse) error {
	cache, err := ReadFile()
	if err != nil {
		return err
	}

	if cache.Credentials == nil {
		cache.Credentials = make(map[string]*oidc.AccessTokenResponse)
	}
	cache.Credentials[issuer] = token

	if err := writeCacheFile(cache); err != nil {
		return err
	}

	return nil
}

func filePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Join(err, errors.New("could not find user's home directory"))
	}

	cacheDir := path.Join(homeDir, ".everest")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", errors.Join(err, errors.New("could not create config directory"))
	}

	cacheFile := path.Join(cacheDir, "cache")

	return cacheFile, nil
}

func readCacheFile() (*File, error) {
	cacheFile, err := filePath()
	if err != nil {
		return nil, err
	}

	cacheData, err := os.ReadFile(cacheFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.Join(err, errors.New("could not read cache file"))
	}

	parsedCache := &File{}
	if len(cacheData) != 0 {
		if err := json.Unmarshal(cacheData, &parsedCache); err != nil {
			return nil, errors.Join(err, errors.New("could not parse cache file"))
		}
	}

	return parsedCache, nil
}

func writeCacheFile(f *File) error {
	cacheJson, err := json.Marshal(f)
	if err != nil {
		return errors.Join(err, errors.New("could not marshal cache data"))
	}

	cacheFile, err := filePath()
	if err != nil {
		return err
	}

	err = os.WriteFile(cacheFile, cacheJson, 0600)
	if err != nil {
		return errors.Join(err, errors.New("could not update cache file"))
	}

	return nil
}
