package swagger

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed swagger-ui
var swaggerUI embed.FS

func GetSpec() []byte {
	return openapiSpec
}

func GetUI() fs.FS {
	uiFS, err := fs.Sub(swaggerUI, "swagger-ui")
	if err != nil {
		panic(fmt.Errorf("load swagger ui filesystem: %w", err))
	}

	return uiFS
}
