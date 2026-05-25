package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

func TestAuthorizeValidationAndRule(t *testing.T) {
	usersRepo := repository.NewUserMemoryRepository()
	groupsRepo := repository.NewGroupMemoryRepository()
	membershipsRepo := repository.NewMembershipMemoryRepository()
	friendshipsRepo := repository.NewFriendshipMemoryRepository()
	h := NewAuthorizeHandler(usersRepo, membershipsRepo, friendshipsRepo)

	requester, _ := usersRepo.Create("requester-auth", "", "", "", "ACTIVE")
	owner, _ := usersRepo.Create("owner-auth", "", "", "", "ACTIVE")
	group, _ := groupsRepo.Create("g1", "Group 1", "", false)
	_, _ = membershipsRepo.Upsert(requester.ID, group.ID, "EDITOR", "ACTIVE")
	_, _ = membershipsRepo.Upsert(owner.ID, group.ID, "EDITOR", "ACTIVE")
	_, _ = friendshipsRepo.Create(requester.ID, owner.ID)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantAllowed *bool
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
			body:       `{"authIdentifier":"requester-auth","group":"` + group.ID + `","service":"metadata-lifecycle","action":"publish"}`,
			wantStatus: http.StatusOK,
			wantAllowed: ptrBool(false),
		},
		{
			name:       "friend in same group can read draft",
			body:       `{"authIdentifier":"requester-auth","group":"` + group.ID + `","service":"metadata-lifecycle","action":"read","resourceType":"draft","resourceOwnerIdentifier":"owner-auth"}`,
			wantStatus: http.StatusOK,
			wantAllowed: ptrBool(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v2/authorize", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			h.Authorize(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantAllowed != nil {
				var resp AuthorizeResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Allowed != *tt.wantAllowed {
					t.Fatalf("expected allowed=%v, got %v", *tt.wantAllowed, resp.Allowed)
				}
			}
		})
	}
}

func ptrBool(v bool) *bool {
	return &v
}
