package handlers

import (
	"testing"

	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/go-faster/jx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToAPIConfiguration(t *testing.T) {
	title := "Vesuvius & Campi Flegrei volcanoes"
	mapped, err := toAPIConfiguration(&configurationv1.Configuration{
		Id:             "1e2c03fb-5541-447e-aeae-8c630110bf24",
		Configuration:  []byte(`{"configurables":[]}`),
		Title:          &title,
		ListOfServices: []string{"GNSS Stations"},
		IsOfficial:     true,
		IsPublic:       true,
		OwnerId:        "user-123",
		CreatedAt:      timestamppb.Now(),
		UpdatedAt:      timestamppb.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mapped.ID.String() != "1e2c03fb-5541-447e-aeae-8c630110bf24" {
		t.Errorf("id mismatch: %s", mapped.ID)
	}
	if string(mapped.Configuration) != `{"configurables":[]}` {
		t.Errorf("configuration mismatch: %s", mapped.Configuration)
	}
	if !mapped.Title.Set || mapped.Title.Null || mapped.Title.Value != title {
		t.Errorf("title mismatch: %+v", mapped.Title)
	}
	if mapped.Description.Set {
		t.Errorf("absent description must stay unset")
	}
	if !mapped.IsOfficial || !mapped.IsPublic {
		t.Errorf("flags mismatch: official=%v public=%v", mapped.IsOfficial, mapped.IsPublic)
	}
	if mapped.OwnerID != "user-123" {
		t.Errorf("owner mismatch: %s", mapped.OwnerID)
	}
}

func TestToAPIConfigurationRejectsEmpty(t *testing.T) {
	if _, err := toAPIConfiguration(nil); err == nil {
		t.Error("expected error for nil record")
	}
	if _, err := toAPIConfiguration(&configurationv1.Configuration{Id: "not-a-uuid"}); err == nil {
		t.Error("expected error for malformed id")
	}
}

func TestToProtoCreateConfigurationRequest(t *testing.T) {
	hidden := false
	proto, err := toProtoCreateConfigurationRequest(&gatewayapi.CreateConfigurationRequest{
		Configuration:  jx.Raw(`{"configurables":[]}`),
		ListOfServices: []string{"InSAR"},
		IsOfficial:     gatewayapi.OptBool{Value: true, Set: true},
		IsPublic:       gatewayapi.OptNilBool{Value: hidden, Set: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proto.Id != nil {
		t.Errorf("absent id must stay nil")
	}
	if string(proto.GetConfiguration()) != `{"configurables":[]}` {
		t.Errorf("configuration mismatch: %s", proto.GetConfiguration())
	}
	if len(proto.GetListOfServices()) != 1 {
		t.Errorf("services mismatch: %v", proto.GetListOfServices())
	}
	if proto.Title != nil {
		t.Errorf("absent metadata must stay nil")
	}
	if !proto.GetIsOfficial() {
		t.Errorf("official flag must be forwarded")
	}
	if proto.IsPublic == nil || proto.GetIsPublic() {
		t.Errorf("explicit private flag must be forwarded")
	}
}

func TestToProtoCreateConfigurationRequestDefaults(t *testing.T) {
	proto, err := toProtoCreateConfigurationRequest(&gatewayapi.CreateConfigurationRequest{
		Configuration: jx.Raw(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proto.GetIsOfficial() {
		t.Errorf("absent official must default to false")
	}
	if proto.IsPublic != nil {
		t.Errorf("absent visibility must stay nil (server defaults to public)")
	}
}

func TestToProtoListConfigurationsRequest(t *testing.T) {
	proto := toProtoListConfigurationsRequest(gatewayapi.ListConfigurationsParams{})
	if proto.PageSize != nil || proto.Offset != nil || proto.Official != nil {
		t.Errorf("absent pagination must stay nil, got %+v", proto)
	}
	official := gatewayapi.ListConfigurationsParams{
		Official: gatewayapi.OptBool{Value: true, Set: true},
	}
	if got := toProtoListConfigurationsRequest(official); got.Official == nil || !got.GetOfficial() {
		t.Errorf("official filter must be forwarded, got %+v", got)
	}
}
