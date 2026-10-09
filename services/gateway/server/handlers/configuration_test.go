package handlers

import (
	"encoding/json"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/go-faster/jx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToAPIConfigurationOwnerIsPermissionGated(t *testing.T) {
	record := &configurationv1.Configuration{
		Id:             "1e2c03fb-5541-447e-aeae-8c630110bf24",
		Configuration:  []byte(`{"configurables":[]}`),
		Title:          "Workspace",
		ListOfServices: nil,
		IsOfficial:     true,
		IsPublic:       true,
		OwnerId:        "user-123",
		CreatedAt:      timestamppb.Now(),
		UpdatedAt:      timestamppb.Now(),
	}

	mapped, err := toAPIConfiguration(record, false)
	if err != nil {
		t.Fatal(err)
	}
	if mapped.OwnerID.Set {
		t.Fatal("owner ID must be omitted for ordinary callers")
	}
	if mapped.Description.Null != true || mapped.ListOfServices == nil {
		t.Fatalf("nullable/empty fields = %+v", mapped)
	}
	if !mapped.IsOfficial || !mapped.IsPublic || string(mapped.Configuration) != `{"configurables":[]}` {
		t.Fatalf("mapped detail = %+v", mapped)
	}

	mapped, err = toAPIConfiguration(record, true)
	if err != nil {
		t.Fatal(err)
	}
	if ownerID, ok := mapped.OwnerID.Get(); !ok || ownerID != "user-123" {
		t.Fatalf("owner ID = %q, %v", ownerID, ok)
	}
}

func TestToAPIConfigurationRejectsMalformedUpstream(t *testing.T) {
	for _, record := range []*configurationv1.Configuration{
		nil,
		{Id: "not-a-uuid"},
		{Id: "1e2c03fb-5541-447e-aeae-8c630110bf24", Configuration: []byte("invalid")},
		{Id: "1e2c03fb-5541-447e-aeae-8c630110bf24", Configuration: []byte(`{}`)},
	} {
		if _, err := toAPIConfiguration(record, false); err == nil {
			t.Fatalf("expected mapping error for %+v", record)
		}
	}
}

func TestToAPIConfigurationSummaryOwnerIsPermissionGated(t *testing.T) {
	record := &configurationv1.ConfigurationSummary{
		Id:         "1e2c03fb-5541-447e-aeae-8c630110bf24",
		Title:      "Official",
		IsOfficial: true,
		IsPublic:   true,
		OwnerId:    new("owner"),
		CreatedAt:  timestamppb.Now(),
		UpdatedAt:  timestamppb.Now(),
	}
	public, err := toAPIConfigurationSummary(record, false)
	if err != nil {
		t.Fatal(err)
	}
	if public.OwnerID.Set || public.ListOfServices == nil || !public.Description.Null {
		t.Fatalf("public summary = %+v", public)
	}
	managed, err := toAPIConfigurationSummary(record, true)
	if err != nil {
		t.Fatal(err)
	}
	if ownerID, ok := managed.OwnerID.Get(); !ok || ownerID != "owner" {
		t.Fatalf("managed owner ID = %q, %v", ownerID, ok)
	}
}

func TestToAPIOfficialConfigurationSummaryOmitsOwner(t *testing.T) {
	record := &configurationv1.ConfigurationSummary{
		Id:         "1e2c03fb-5541-447e-aeae-8c630110bf24",
		Title:      "Official",
		IsOfficial: true,
		IsPublic:   true,
		OwnerId:    new("private-owner"),
		CreatedAt:  timestamppb.Now(),
		UpdatedAt:  timestamppb.Now(),
	}
	mapped, err := toAPIOfficialConfigurationSummary(record)
	if err != nil {
		t.Fatal(err)
	}
	if !mapped.IsOfficial || !mapped.IsPublic || mapped.ListOfServices == nil {
		t.Fatalf("official summary = %+v", mapped)
	}
	encoded, err := json.Marshal(mapped)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, included := fields["owner_id"]; included {
		t.Fatalf("official summary leaked owner_id: %s", encoded)
	}
	for _, field := range []string{"is_official", "is_public"} {
		if _, included := fields[field]; !included {
			t.Fatalf("official summary omitted %s: %s", field, encoded)
		}
	}
}

func TestToProtoConfigurationRequestsPreserveVisibilityPresence(t *testing.T) {
	created, err := toProtoCreateConfigurationRequest(&gatewayapi.ConfigurationWriteRequest{
		Configuration: jx.Raw(`{}`),
		Title:         "Test",
	})
	if err != nil || created.IsPublic {
		t.Fatalf("create request = %+v, error = %v", created, err)
	}

	for _, visibility := range []gatewayapi.OptNilBool{
		{},
		{Set: true, Null: true},
		{Set: true, Value: false},
		{Set: true, Value: true},
	} {
		request, err := toProtoReplaceConfigurationRequest("1e2c03fb-5541-447e-aeae-8c630110bf24", &gatewayapi.ConfigurationWriteRequest{
			Configuration: jx.Raw(`{}`),
			Title:         "Test",
			IsPublic:      visibility,
		})
		if err != nil {
			t.Fatal(err)
		}
		wantPresent := visibility.Set && !visibility.Null
		if (request.IsPublic != nil) != wantPresent {
			t.Fatalf("visibility %+v mapped to %v", visibility, request.IsPublic)
		}
		if wantPresent && *request.IsPublic != visibility.Value {
			t.Fatalf("visibility %+v mapped to %v", visibility, *request.IsPublic)
		}
	}
}

func TestToProtoSetOfficialStatus(t *testing.T) {
	request, err := toProtoSetConfigurationOfficialStatusRequest("1e2c03fb-5541-447e-aeae-8c630110bf24", &gatewayapi.SetConfigurationOfficialStatusRequest{IsOfficial: false})
	if err != nil || request.IsOfficial == nil || *request.IsOfficial {
		t.Fatalf("status request = %+v, error = %v", request, err)
	}
}

func TestToProtoListConfigurationsRequest(t *testing.T) {
	request, err := toProtoListConfigurationsRequest(gatewayapi.ListConfigurationsParams{})
	if err != nil || request.GetScope() != configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN {
		t.Fatalf("default list = %+v, error = %v", request, err)
	}

	owner := gatewayapi.ListConfigurationsParams{OwnerID: gatewayapi.NewOptString("user-2")}
	request, err = toProtoListConfigurationsRequest(owner)
	if err != nil || request.GetScope() != configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER || request.GetOwnerId() != "user-2" {
		t.Fatalf("owner list = %+v, error = %v", request, err)
	}

	all := gatewayapi.ListConfigurationsParams{Scope: gatewayapi.OptListConfigurationsScope{Set: true, Value: gatewayapi.ListConfigurationsScopeAll}}
	request, err = toProtoListConfigurationsRequest(all)
	if err != nil || request.GetScope() != configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_ALL {
		t.Fatalf("all list = %+v, error = %v", request, err)
	}

	conflict := all
	conflict.OwnerID.SetTo("user-2")
	if _, err := toProtoListConfigurationsRequest(conflict); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("conflicting filters error = %v", err)
	}
}
