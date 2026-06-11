// Package docs exposes embedded OpenAPI spec and API documentation UI assets.
package docs

import (
	_ "embed"
	"io/fs"

	apidocs "github.com/EPOS-ERIC/epos-platform/packages/go/apidocs"
)

//go:embed openapi.yaml
var openapiSpec []byte

func GetSpec() []byte {
	return openapiSpec
}

func GetUI() fs.FS {
	return apidocs.GetUI()
}

func GetScalarUI() fs.FS {
	return apidocs.GetScalarUI()
}
