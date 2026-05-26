// Package docs exposes embedded OpenAPI spec and API documentation UI assets.
package docs

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed swagger-ui
var swaggerUI embed.FS

//go:embed scalar
var scalarUI embed.FS

func GetSpec() []byte {
	return openapiSpec
}

func GetUI() fs.FS {
	uiFS := subFSOrPanic(swaggerUI, "swagger-ui")
	return uiFS
}

func GetScalarUI() fs.FS {
	uiFS := subFSOrPanic(scalarUI, "scalar")
	return uiFS
}

func subFSOrPanic(sourceFS fs.FS, dir string) fs.FS {
	uiFS, err := fs.Sub(sourceFS, dir)
	if err != nil {
		panic(fmt.Errorf("load docs ui filesystem: %w", err))
	}

	return uiFS
}
