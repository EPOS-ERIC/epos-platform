// Package swagger exposes embedded OpenAPI spec and Swagger UI assets.
package swagger

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
)

var log = logging.Get("swagger")

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed swagger-ui
var swaggerUI embed.FS

func GetSpec() []byte {
	log.Debug("returning openapi spec")
	return openapiSpec
}

func GetUI() fs.FS {
	uiFS := subFSOrPanic(swaggerUI, "swagger-ui")
	log.Debug("returning swagger ui filesystem")
	return uiFS
}

func subFSOrPanic(sourceFS fs.FS, dir string) fs.FS {
	uiFS, err := fs.Sub(sourceFS, dir)
	if err != nil {
		panic(fmt.Errorf("load swagger ui filesystem: %w", err))
	}

	return uiFS
}
