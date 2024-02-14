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

// Package upgrade implements upgrade logic for the CLI.
package upgrade

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	version "github.com/Percona-Lab/percona-version-service/versionpb"
	goversion "github.com/hashicorp/go-version"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"

	"github.com/percona/percona-everest-cli/pkg/kubernetes"
	cliVersion "github.com/percona/percona-everest-cli/pkg/version"
)

type (
	// UpgradeConfig defines configuration required for upgrade command.
	UpgradeConfig struct {
		Everest struct {
			// Endpoint stores URL to Everest.
			Endpoint string
			// Token stores Everest token.
			Token string
		}

		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`
		// Namespace holds namespace where Everest is installed.
		Namespace string
		// VersionMetadataURL stores hostname to retrieve version metadata information from.
		VersionMetadataURL string `mapstructure:"version-metadata-url"`
	}

	// Upgrade struct implements upgrade command.
	Upgrade struct {
		l *zap.SugaredLogger

		config        *UpgradeConfig
		everestClient everestClientConnector
		kubeClient    *kubernetes.Kubernetes
	}

	minimumVersion struct {
		catalog *goversion.Version
		olm     *goversion.Version
	}
)

var ErrNoUpdateAvailable = errors.New("no update available")

// NewUpgrade returns a new Upgrade struct.
func NewUpgrade(cfg *UpgradeConfig, everestClient everestClientConnector, l *zap.SugaredLogger) (*Upgrade, error) {
	cli := &Upgrade{
		config:        cfg,
		everestClient: everestClient,
		l:             l.With("component", "upgrade"),
	}

	k, err := kubernetes.New(cfg.KubeconfigPath, cli.l)
	if err != nil {
		var u *url.Error
		if errors.As(err, &u) {
			cli.l.Error("Could not connect to Kubernetes. " +
				"Make sure Kubernetes is running and is accessible from this computer/server.")
		}
		return nil, err
	}
	cli.kubeClient = k
	return cli, nil
}

// Run runs the operators installation process.
func (u *Upgrade) Run(ctx context.Context) error {
	// Check prerequisites
	upgradeEverestTo, minVer, err := u.canUpgrade(ctx)
	if err != nil {
		if errors.Is(err, ErrNoUpdateAvailable) {
			u.l.Info("You're running the latest version of Everest")
			return nil
		}
		return err
	}

	// Start upgrade.
	if err := u.upgradeOLM(ctx, minVer.olm); err != nil {
		return err
	}

	// TODO: figure out catalog version
	u.l.Infof("Upgrading Percona Catalog to %s", upgradeEverestTo)
	if err := u.kubeClient.InstallPerconaCatalog(ctx, upgradeEverestTo); err != nil {
		return err
	}

	u.l.Infof("Upgrading Everest to %s in namespace %s", upgradeEverestTo, u.config.Namespace)
	if err := u.kubeClient.InstallEverest(ctx, u.config.Namespace, upgradeEverestTo); err != nil {
		return err
	}

	u.l.Infof("Everest has been upgraded to version %s", upgradeEverestTo)

	return nil
}

// canUpgrade checks if there's a new Everest version available and if we can upgrade to it
// based on minimum requirements.
func (u *Upgrade) canUpgrade(ctx context.Context) (*goversion.Version, *minimumVersion, error) {
	// Get Everest version.
	eVer, err := u.everestClient.Version(ctx)
	if err != nil {
		return nil, nil, errors.Join(err, errors.New("could not retrieve Everest version"))
	}
	everestVersion, err := goversion.NewSemver(eVer.Version)
	if err != nil {
		return nil, nil, errors.Join(err, fmt.Errorf("invalid Everest version %s", eVer.Version))
	}

	u.l.Infof("Current Everest version is %s", everestVersion)

	// Determine version to upgrade to.
	upgradeEverestTo, meta, err := u.versionToUpgradeTo(ctx, everestVersion)
	if err != nil {
		return nil, nil, err
	}

	// Check minimum requirements.
	minVer, err := u.verifyMinimumRequirements(ctx, meta)
	if err != nil {
		return nil, nil, err
	}

	return upgradeEverestTo, minVer, nil
}

// versionToUpgradeTo returns version to which the current Everest version can be upgraded to.
func (u *Upgrade) versionToUpgradeTo(
	ctx context.Context, currentEverestVersion *goversion.Version,
) (*goversion.Version, *version.MetadataVersion, error) {
	req, err := cliVersion.Metadata(ctx, u.config.VersionMetadataURL)
	if err != nil {
		return nil, nil, err
	}

	var (
		upgradeTo *goversion.Version
		meta      *version.MetadataVersion
	)
	for _, v := range req.Versions {
		ver, err := goversion.NewVersion(v.Version)
		if err != nil {
			u.l.Debugf("Could not parse version %s. Error: %s", v.Version, err)
			continue
		}

		if currentEverestVersion.GreaterThanOrEqual(ver) {
			continue
		}

		if upgradeTo == nil {
			upgradeTo = ver
			meta = v
			continue
		}

		// Select the latest patch version for the same major and minor version.
		verSeg := ver.Segments()
		uSeg := upgradeTo.Segments()
		if len(verSeg) >= 3 && len(uSeg) >= 3 && verSeg[0] == uSeg[0] && verSeg[1] == uSeg[1] && verSeg[2] > uSeg[2] {
			upgradeTo = ver
			meta = v
			continue
		}

		if upgradeTo.GreaterThan(ver) {
			upgradeTo = ver
			meta = v
			continue
		}
	}

	if upgradeTo == nil {
		return nil, nil, ErrNoUpdateAvailable
	}

	return upgradeTo, meta, nil
}

func (u *Upgrade) verifyMinimumRequirements(ctx context.Context, meta *version.MetadataVersion) (*minimumVersion, error) {
	minVer, err := u.minimumVersion(meta)
	if err != nil {
		return nil, err
	}

	if err := u.checkRequirements(minVer); err != nil {
		return nil, err
	}

	return minVer, nil
}

func (u *Upgrade) minimumVersion(meta *version.MetadataVersion) (*minimumVersion, error) {
	olm, ok := meta.Requirements["olm"]
	if !ok {
		olm = "0.0.0"
	}
	catalog, ok := meta.Requirements["catalog"]
	if !ok {
		catalog = "0.0.0"
	}

	vOLM, err := goversion.NewSemver(olm)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("invalid OLM version %s", olm))
	}

	vCatalog, err := goversion.NewSemver(catalog)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("invalid catalog version %s", catalog))
	}

	return &minimumVersion{
		olm:     vOLM,
		catalog: vCatalog,
	}, nil
}

func (u *Upgrade) checkRequirements(minVer *minimumVersion) error {
	// TODO: to be implemented.
	return nil
}

func (u *Upgrade) upgradeOLM(ctx context.Context, minimumVersion *goversion.Version) error {
	u.l.Info("Checking OLM version")
	csv, err := u.kubeClient.GetClusterServiceVersion(ctx, types.NamespacedName{
		Name:      "packageserver",
		Namespace: kubernetes.OLMNamespace,
	})
	if err != nil {
		return errors.Join(err, errors.New("could not retrieve Cluster Service Version"))
	}
	foundVersion, err := goversion.NewSemver(csv.Spec.Version.String())
	if err != nil {
		return err
	}
	u.l.Infof("OLM version is %s. Minimum version is %s", foundVersion, minimumVersion)
	if !foundVersion.LessThan(minimumVersion) {
		u.l.Info("OLM version is supported. No action is required.")
		return nil
	}

	u.l.Info("Upgrading OLM to version %s", minimumVersion)
	// TODO: actually upgrade OLM operator instead of installation/skip.
	if err := u.kubeClient.InstallOLMOperator(ctx, true); err != nil {
		return errors.Join(err, errors.New("could not upgrade OLM"))
	}
	u.l.Info("OLM has been upgraded")

	return nil
}
