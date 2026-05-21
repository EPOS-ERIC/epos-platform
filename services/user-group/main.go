package main

import (
	"log"
	"net/http"
	"os"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/server"
)

func main() {
	addr := os.Getenv("USER_GROUP_LISTEN_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	openapiPath := os.Getenv("USER_GROUP_OPENAPI_PATH")
	if openapiPath == "" {
		openapiPath = "openapi.yaml"
	}

	handler := server.NewRouter(openapiPath)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
