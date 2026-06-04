package metadataaccess

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
)

func TestFetchDetailMapsBadRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"BAD_REQUEST","message":"Missing shape or id","details":null}`))
	}))
	t.Cleanup(server.Close)

	client := testfixtures.NewMetadataClient(server.URL)

	_, err := FetchDetail(context.Background(), client, "distribution", "\f")
	if err == nil {
		t.Fatal("FetchDetail() error = nil")
	}
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("FetchDetail() error = %v, want ErrBadRequest", err)
	}
}
