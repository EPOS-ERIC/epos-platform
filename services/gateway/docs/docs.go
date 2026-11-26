// Package docs exposes embedded OpenAPI spec and API documentation UI assets.
package docs

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed scalar swagger-ui
var ui embed.FS

func GetSpec() []byte {
	return openapiSpec
}

func GetUI() fs.FS {
	return uiFS("swagger-ui")
}

func GetScalarUI() fs.FS {
	return uiFS("scalar")
}

func uiFS(dir string) fs.FS {
	sub, err := fs.Sub(ui, dir)
	if err != nil {
		panic(fmt.Errorf("load docs ui filesystem: %w", err))
	}
	return sub
}
