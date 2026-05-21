package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizeValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{"authIdentifier":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing required fields",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid shape returns not implemented",
			body:       `{"authIdentifier":"a","group":"g","service":"metadata-lifecycle","action":"publish"}`,
			wantStatus: http.StatusNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v2/authorize", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			Authorize(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
