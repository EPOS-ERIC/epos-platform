package handlers

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/swagger"
)

func SwaggerRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
}

func OpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(swagger.GetSpec())
}
