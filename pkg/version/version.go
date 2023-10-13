package version

import (
	"fmt"
	"strings"
)

const (
	devCatalogImage     = "docker.io/percona/everest-catalog:latest"
	releaseCatalogImage = "docker.io/percona/everest-catalog:%s"
)

var (
	// ProjectName is a component name, e.g. everestctl.
	ProjectName string
	// Version is a component version e.g. v0.3.0-1-a93bef.
	Version string
	// FullCommit is a git commit hash.
	FullCommit string
	// CatalogImage is a image path for OLM catalog
	catalogImage string
)

func CatalogImage() string {
	catalogImage = fmt.Sprintf(releaseCatalogImage, Version)
	if strings.Contains(Version, "dirty") {
		catalogImage = devCatalogImage
	}
	return catalogImage
}

func FullVersionInfo() string {
	out := []string{
		"ProjectName: " + ProjectName,
		"Version: " + Version,
		"FullCommit: " + FullCommit,
	}
	return strings.Join(out, "\n")
}
