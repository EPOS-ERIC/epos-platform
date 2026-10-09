package grpcapi

import (
	"context"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/configuration"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type policyRepository struct {
	configuration.Repository
	filter  configuration.ListFilter
	records map[uuid.UUID]domain.Configuration
}

func newPolicyRepository() *policyRepository {
	return &policyRepository{records: make(map[uuid.UUID]domain.Configuration)}
}

func (r *policyRepository) Create(_ context.Context, ownerID domain.Actor, input domain.ConfigurationInput) (domain.Configuration, error) {
	record, err := domain.NewConfiguration(uuid.New(), input, false, string(ownerID), time.Now(), time.Now())
	if err == nil {
		r.records[record.ID] = record
	}
	return record, err
}

func (r *policyRepository) GetByID(_ context.Context, id uuid.UUID) (domain.Configuration, error) {
	record, ok := r.records[id]
	if !ok {
		return domain.Configuration{}, apperror.Default(apperror.CodeNotFound)
	}
	return record, nil
}

func (r *policyRepository) List(_ context.Context, filter configuration.ListFilter) ([]domain.ConfigurationSummary, error) {
	r.filter = filter
	return nil, nil
}

func (r *policyRepository) Update(_ context.Context, id uuid.UUID, caller domain.Caller, input domain.ConfigurationInput) (domain.Configuration, error) {
	record, ok := r.records[id]
	if !ok || (!caller.CanManage && (record.IsOfficial || caller.Actor != record.OwnerID)) || (record.IsOfficial && input.IsPublic != nil && !*input.IsPublic) {
		return domain.Configuration{}, apperror.Default(apperror.CodeNotFound)
	}
	public := false
	if record.IsOfficial {
		public = true
	} else if input.IsPublic != nil {
		public = *input.IsPublic
	}
	input.IsPublic = &public
	updated, err := domain.NewConfiguration(record.ID, input, record.IsOfficial, string(record.OwnerID), record.CreatedAt, time.Now())
	if err == nil {
		r.records[id] = updated
	}
	return updated, err
}

func (r *policyRepository) Delete(_ context.Context, id uuid.UUID, caller domain.Caller) error {
	record, ok := r.records[id]
	if !ok || (!caller.CanManage && (record.IsOfficial || caller.Actor != record.OwnerID)) {
		return apperror.Default(apperror.CodeNotFound)
	}
	delete(r.records, id)
	return nil
}

func (r *policyRepository) SetOfficialStatus(_ context.Context, id uuid.UUID, caller domain.Caller, isOfficial bool) (domain.Configuration, error) {
	record, ok := r.records[id]
	if !ok || !caller.CanManage {
		return domain.Configuration{}, apperror.Default(apperror.CodeNotFound)
	}
	if record.IsOfficial != isOfficial {
		record.IsOfficial = isOfficial
		record.IsPublic = true
		record.UpdatedAt = time.Now()
		r.records[id] = record
	}
	return record, nil
}

func testContext(actor string, manage bool) context.Context {
	if actor == "" {
		return context.Background()
	}
	pairs := []string{userIDMetadataKey, actor}
	if manage {
		pairs = append(pairs, manageMetadataKey, "true")
	}
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(pairs...))
}

func TestListConfigurationAuthorizationAndScope(t *testing.T) {
	for _, test := range []struct {
		name    string
		actor   string
		manage  bool
		scope   configurationv1.ConfigurationListScope
		ownerID string
		code    apperror.Code
	}{
		{name: "anonymous own", scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN, code: apperror.CodeUnauthenticated},
		{name: "own", actor: "owner", scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN},
		{name: "manager own remains own", actor: "manager", manage: true, scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN},
		{name: "anonymous official", scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OFFICIAL},
		{name: "anonymous owner filter", scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER, ownerID: "owner", code: apperror.CodeUnauthenticated},
		{name: "user owner filter", actor: "user", scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER, ownerID: "owner", code: apperror.CodePermissionDenied},
		{name: "manager owner filter", actor: "manager", manage: true, scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER, ownerID: "owner"},
		{name: "manager all", actor: "manager", manage: true, scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_ALL},
		{name: "unspecified scope", actor: "owner", code: apperror.CodeValidationFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newPolicyRepository()
			request := &configurationv1.ListConfigurationsRequest{Scope: test.scope}
			if test.ownerID != "" {
				request.OwnerId = &test.ownerID
			}
			_, err := New(configuration.New(repo)).ListConfigurations(testContext(test.actor, test.manage), request)
			if test.code != "" {
				if !apperror.IsCode(err, test.code) {
					t.Fatalf("error = %v, want %s", err, test.code)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(repo.filter.Scope) != testScopeName(test.scope) || repo.filter.OwnerID != domain.Actor(test.ownerID) || repo.filter.Caller.Actor != domain.Actor(test.actor) || repo.filter.Caller.CanManage != test.manage {
				t.Fatalf("filter = %+v", repo.filter)
			}
			if test.scope == configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OFFICIAL && len(repo.records) > 0 {
				t.Fatal("official list should not mutate records")
			}
		})
	}
}

func testScopeName(scope configurationv1.ConfigurationListScope) string {
	switch scope {
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN:
		return string(domain.ListScopeOwn)
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER:
		return string(domain.ListScopeOwner)
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_ALL:
		return string(domain.ListScopeAll)
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OFFICIAL:
		return string(domain.ListScopeOfficial)
	default:
		return ""
	}
}

func TestUnifiedReadAndMutationAuthorization(t *testing.T) {
	repo := newPolicyRepository()
	service := configuration.New(repo)
	owner := testContext("owner", false)
	created, err := New(service).CreateConfiguration(owner, &configurationv1.CreateConfigurationRequest{Configuration: []byte(`{}`), Title: "Private"})
	if err != nil {
		t.Fatal(err)
	}
	id := created.GetConfiguration().GetId()
	privateID, _ := uuid.Parse(id)
	stranger := testContext("stranger", false)
	if _, err := New(service).GetConfiguration(stranger, &configurationv1.GetConfigurationRequest{Id: id}); !apperror.IsCode(err, apperror.CodeNotFound) {
		t.Fatalf("private read error = %v", err)
	}
	_, err = New(service).UpdateConfiguration(stranger, &configurationv1.UpdateConfigurationRequest{Id: id, Configuration: []byte(`{}`), Title: "Blocked"})
	if !apperror.IsCode(err, apperror.CodeNotFound) {
		t.Fatalf("private update error = %v", err)
	}

	manager := testContext("manager", true)
	promoted, err := New(service).SetConfigurationOfficialStatus(manager, &configurationv1.SetConfigurationOfficialStatusRequest{Id: id, IsOfficial: new(true)})
	if err != nil || !promoted.GetConfiguration().GetIsOfficial() || !promoted.GetConfiguration().GetIsPublic() || promoted.GetConfiguration().GetOwnerId() != "owner" {
		t.Fatalf("promoted = %+v, error = %v", promoted, err)
	}
	if _, err := New(service).UpdateConfiguration(owner, &configurationv1.UpdateConfigurationRequest{Id: id, Configuration: []byte(`{}`), Title: "Blocked"}); !apperror.IsCode(err, apperror.CodePermissionDenied) {
		t.Fatalf("official creator update error = %v", err)
	}
	if _, err := New(service).DeleteConfiguration(owner, &configurationv1.DeleteConfigurationRequest{Id: id}); !apperror.IsCode(err, apperror.CodePermissionDenied) {
		t.Fatalf("official creator delete error = %v", err)
	}
	demoted, err := New(service).SetConfigurationOfficialStatus(manager, &configurationv1.SetConfigurationOfficialStatusRequest{Id: id, IsOfficial: new(false)})
	if err != nil || demoted.GetConfiguration().GetIsOfficial() || !demoted.GetConfiguration().GetIsPublic() {
		t.Fatalf("demoted = %+v, error = %v", demoted, err)
	}
	if repo.records[privateID].OwnerID != "owner" {
		t.Fatal("promotion/demotion must preserve creator")
	}
}

func TestStatusPermissionAndRequiredField(t *testing.T) {
	server := New(configuration.New(newPolicyRepository()))
	id := uuid.New().String()
	if _, err := server.SetConfigurationOfficialStatus(testContext("user", false), &configurationv1.SetConfigurationOfficialStatusRequest{Id: id, IsOfficial: new(true)}); !apperror.IsCode(err, apperror.CodePermissionDenied) {
		t.Fatalf("non-manager error = %v", err)
	}
	if _, err := server.SetConfigurationOfficialStatus(testContext("manager", true), &configurationv1.SetConfigurationOfficialStatusRequest{Id: id}); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("missing status error = %v", err)
	}
}
