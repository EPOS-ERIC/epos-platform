package grpcapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func TestToProtoConfigurationRoundTrip(t *testing.T) {
	id := uuid.New()
	title := "Vesuvius & Campi Flegrei volcanoes"
	now := time.Now().UTC().Truncate(time.Microsecond)
	public := true
	record, err := domain.NewConfiguration(id, domain.ConfigurationInput{
		Configuration:  json.RawMessage(`{"configurables":[]}`),
		Title:          title,
		ListOfServices: []string{"GNSS Stations"},
		IsPublic:       &public,
	}, true, "user-123", now, now)
	if err != nil {
		t.Fatal(err)
	}

	proto := toProtoConfiguration(record)
	if proto.GetId() != id.String() || string(proto.GetConfiguration()) != `{"configurables":[]}` || proto.GetTitle() != title {
		t.Fatalf("mapped configuration = %+v", proto)
	}
	if proto.Description != nil {
		t.Fatal("absent description must stay nil")
	}
	if proto.GetIsOfficial() != true || !proto.GetIsPublic() || proto.GetOwnerId() != "user-123" {
		t.Fatalf("status/owner mismatch: %+v", proto)
	}
	if len(proto.GetListOfServices()) != 1 || proto.GetListOfServices()[0] != "GNSS Stations" {
		t.Fatalf("services mismatch: %v", proto.GetListOfServices())
	}
	if !proto.GetCreatedAt().AsTime().Equal(now) {
		t.Fatal("created_at mismatch")
	}
}

func TestSummaryOwnershipIsCapabilityGated(t *testing.T) {
	now := time.Now().UTC()
	record, err := domain.NewConfigurationSummary(uuid.New(), "Summary", nil, nil, true, true, "owner", now, now)
	if err != nil {
		t.Fatal(err)
	}
	withoutOwner := toProtoConfigurationSummary(record, false)
	if withoutOwner.OwnerId != nil {
		t.Fatal("summary must omit owner when caller lacks capability")
	}
	withOwner := toProtoConfigurationSummary(record, true)
	if withOwner.GetOwnerId() != "owner" {
		t.Fatalf("manager summary owner = %q", withOwner.GetOwnerId())
	}
}

func TestToDomainConfigurationInputValidation(t *testing.T) {
	if _, err := toDomainConfigurationInput([]byte(`{invalid`), "Title", nil, nil, nil); err == nil {
		t.Fatal("expected error for invalid configuration JSON")
	}
	raw := []byte(" { \"value\" : 1 } ")
	input, err := toDomainConfigurationInput(raw, "Title", nil, nil, nil)
	if err != nil || string(input.Configuration) != string(raw) {
		t.Fatalf("configuration = %q, error = %v", input.Configuration, err)
	}
}

func TestCallerFromContext(t *testing.T) {
	if caller, err := callerFromContext(context.Background()); err != nil || caller.Actor != "" {
		t.Fatal("expected anonymous without metadata")
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDMetadataKey, "user-123"))
	caller, err := callerFromContext(ctx)
	if err != nil || caller.Actor != "user-123" || caller.CanManage {
		t.Fatalf("expected non-manager caller, got %+v, %v", caller, err)
	}
	managerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDMetadataKey, "manager-1", manageMetadataKey, "true"))
	caller, err = callerFromContext(managerCtx)
	if err != nil || !caller.CanManage {
		t.Fatalf("expected manager caller, got %+v, %v", caller, err)
	}
	invalidCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDMetadataKey, "user-1", userIDMetadataKey, "user-2"))
	if _, err := callerFromContext(invalidCtx); err == nil {
		t.Fatal("expected error for duplicate actor metadata")
	}
}

func TestListScopeMapping(t *testing.T) {
	for _, test := range []struct {
		proto configurationv1.ConfigurationListScope
		want  domain.ListScope
	}{
		{configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN, domain.ListScopeOwn},
		{configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER, domain.ListScopeOwner},
		{configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_ALL, domain.ListScopeAll},
		{configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OFFICIAL, domain.ListScopeOfficial},
	} {
		got, err := toDomainListScope(test.proto)
		if err != nil || got != test.want {
			t.Fatalf("scope %v mapped to %q, %v", test.proto, got, err)
		}
	}
	if _, err := toDomainListScope(configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified list scope")
	}
}
