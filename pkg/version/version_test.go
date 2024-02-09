package version

import (
	"fmt"
	"testing"

	goversion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestCatalogImage(t *testing.T) {
	t.Parallel()
	v, err := goversion.NewVersion("v0.3.0")
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(releaseCatalogImage, v.String()), CatalogImage(v))

	v, err = goversion.NewVersion("v0.3.0-1-asd-dirty")
	assert.NoError(t, err)
	assert.Equal(t, devCatalogImage, CatalogImage(v))

	v, err = goversion.NewVersion("c09550")
	assert.NoError(t, err)
	assert.Equal(t, devCatalogImage, CatalogImage(v))

	v, err = goversion.NewVersion("0.3.0-37-gf1f07f6")
	assert.NoError(t, err)
	assert.Equal(t, devCatalogImage, CatalogImage(v))
}
