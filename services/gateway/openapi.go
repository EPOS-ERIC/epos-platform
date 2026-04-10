package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed swagger-ui
var swaggerUI embed.FS

func serve() {
	mux := http.NewServeMux()

	// Serve the spec
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(openapiSpec)
	})

	// Serve Swagger UI static assets
	swaggerFS, _ := fs.Sub(swaggerUI, "swagger-ui")
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS))))

	http.ListenAndServe(":8088", mux)
}
