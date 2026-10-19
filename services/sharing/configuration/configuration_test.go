package configuration

import (
	"context"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"github.com/google/uuid"
)

type listRepository struct {
	Repository
	filter ListFilter
}

func (r *listRepository) List(_ context.Context, filter ListFilter) ([]domain.ConfigurationSummary, error) {
	r.filter = filter
	return nil, nil
}

func TestReadWritePolicy(t *testing.T) {
	private, err := domain.NewConfiguration(uuid.New(), domain.ConfigurationInput{
		Configuration: []byte(`{}`), Title: "Test",
	}, false, "owner", time.Now(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	owner := domain.Caller{Actor: "owner"}
	manager := domain.Caller{Actor: "manager", CanManage: true}
	other := domain.Caller{Actor: "other"}
	if !canRead(owner, private) || !canRead(manager, private) || canRead(other, private) || canRead(domain.Caller{}, private) {
		t.Fatal("private reads must require owner or manager")
	}
	if !canWrite(owner, private) || !canWrite(manager, private) || canWrite(other, private) {
		t.Fatal("personal writes must require owner or manager")
	}
	public := private
	public.IsPublic = true
	if !canRead(domain.Caller{}, public) {
		t.Fatal("public reads must allow anonymous callers")
	}
	public.IsOfficial = true
	if canWrite(owner, public) || !canWrite(manager, public) {
		t.Fatal("official writes must require manager, even for creator")
	}
}

func TestListAuthorizationAndScope(t *testing.T) {
	for _, test := range []struct {
		name      string
		caller    domain.Caller
		scope     domain.ListScope
		ownerID   domain.Actor
		wantError apperror.Code
	}{
		{name: "anonymous own list", scope: domain.ListScopeOwn, wantError: apperror.CodeUnauthenticated},
		{name: "own list", caller: domain.Caller{Actor: "owner"}, scope: domain.ListScopeOwn},
		{name: "official list is public", scope: domain.ListScopeOfficial},
		{name: "owner filter requires manager", caller: domain.Caller{Actor: "owner"}, scope: domain.ListScopeOwner, ownerID: "other", wantError: apperror.CodePermissionDenied},
		{name: "all filter requires manager", caller: domain.Caller{Actor: "owner"}, scope: domain.ListScopeAll, wantError: apperror.CodePermissionDenied},
		{name: "manager owner filter", caller: domain.Caller{Actor: "manager", CanManage: true}, scope: domain.ListScopeOwner, ownerID: "other"},
		{name: "manager all filter", caller: domain.Caller{Actor: "manager", CanManage: true}, scope: domain.ListScopeAll},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &listRepository{}
			service := New(repo)
			_, err := service.List(t.Context(), test.caller, ListFilter{Scope: test.scope, OwnerID: test.ownerID})
			if test.wantError != "" {
				if !apperror.IsCode(err, test.wantError) {
					t.Fatalf("error = %v, want %s", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repo.filter.Scope != test.scope || repo.filter.OwnerID != test.ownerID || repo.filter.Caller != test.caller {
				t.Fatalf("filter = %+v", repo.filter)
			}
		})
	}
}
