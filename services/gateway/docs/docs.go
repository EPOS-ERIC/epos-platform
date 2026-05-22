// Package docs exposes embedded OpenAPI spec and API documentation UI assets.
package docs

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
)

var log = logging.Get("docs")

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed swagger-ui
var swaggerUI embed.FS

//go:embed scalar
var scalarUI embed.FS

func GetSpec() []byte {
	log.Debug("returning openapi spec")
	return openapiSpec
}

func GetUI() fs.FS {
	uiFS := subFSOrPanic(swaggerUI, "swagger-ui")
	log.Debug("returning swagger ui filesystem")
	return uiFS
}

func GetScalarUI() fs.FS {
	uiFS := subFSOrPanic(scalarUI, "scalar")
	log.Debug("returning scalar ui filesystem")
	return uiFS
}

func subFSOrPanic(sourceFS fs.FS, dir string) fs.FS {
	uiFS, err := fs.Sub(sourceFS, dir)
	if err != nil {
		panic(fmt.Errorf("load docs ui filesystem: %w", err))
	}

	return uiFS
}
