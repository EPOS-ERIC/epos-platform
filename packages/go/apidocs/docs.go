// Package apidocs exposes shared API documentation UI assets.
package apidocs

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed scalar
var scalarUI embed.FS

//go:embed swagger-ui
var swaggerUI embed.FS

func GetUI() fs.FS {
	return subFSOrPanic(swaggerUI, "swagger-ui")
}

func GetScalarUI() fs.FS {
	return subFSOrPanic(scalarUI, "scalar")
}

func subFSOrPanic(sourceFS fs.FS, dir string) fs.FS {
	uiFS, err := fs.Sub(sourceFS, dir)
	if err != nil {
		panic(fmt.Errorf("load docs ui filesystem: %w", err))
	}

	return uiFS
}
