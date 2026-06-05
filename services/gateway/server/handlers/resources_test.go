package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func TestWriteResourceErrorMapsBadRequest(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	(&Resources{}).writeResourceError(context.Background(), rec, domain.ErrBadRequest, "distribution")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}
