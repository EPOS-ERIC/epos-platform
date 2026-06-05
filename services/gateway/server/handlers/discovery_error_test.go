package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func TestWriteDiscoveryErrorMapsBadRequest(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	(&Discovery{}).writeDiscoveryError(context.Background(), rec, domain.ErrBadRequest, "metadata search")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}
