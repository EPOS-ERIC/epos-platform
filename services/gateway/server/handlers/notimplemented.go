package handlers

import "net/http"

// NotImplemented marks Gateway routes awaiting Backoffice API coverage.
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	WriteError(r.Context(), w, http.StatusNotImplemented, "not_implemented", "endpoint is not available through Backoffice yet", nil)
}
