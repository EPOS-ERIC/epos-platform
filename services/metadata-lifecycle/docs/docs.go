// Package docs exposes embedded OpenAPI spec and API documentation UI assets.
package docs

import (
	_ "embed"
	"io/fs"

	apidocs "github.com/EPOS-ERIC/epos-platform/packages/go/apidocs"
)

//go:embed openapi.yaml
var openapiSpec []byte

// GetSpec returns embedded OpenAPI specification bytes.
func GetSpec() []byte {
	return openapiSpec
}

// GetUI returns embedded API documentation UI files.
func GetUI() fs.FS {
	return apidocs.GetUI()
}

// GetScalarUI returns embedded Scalar UI files.
func GetScalarUI() fs.FS {
	return apidocs.GetScalarUI()
}
