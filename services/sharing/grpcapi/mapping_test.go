package grpcapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/sharing/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func TestToProtoConfigurationRoundTrip(t *testing.T) {
	id := uuid.New()
	title := "Vesuvius & Campi Flegrei volcanoes"
	now := time.Now().UTC().Truncate(time.Microsecond)
	record := repository.ConfigurationRecord{
		ID:             id,
		Configuration:  json.RawMessage(`{"configurables":[]}`),
		Title:          &title,
		ListOfServices: []string{"GNSS Stations"},
		IsOfficial:     true,
		IsPublic:       true,
		OwnerID:        "user-123",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	proto := toProtoConfiguration(record)
	if proto.GetId() != id.String() {
		t.Errorf("id mismatch: %s", proto.GetId())
	}
	if string(proto.GetConfiguration()) != `{"configurables":[]}` {
		t.Errorf("configuration mismatch: %s", proto.GetConfiguration())
	}
	if proto.GetTitle() != title {
		t.Errorf("title mismatch: %s", proto.GetTitle())
	}
	if proto.Description != nil {
		t.Errorf("absent description must stay nil")
	}
	if len(proto.GetListOfServices()) != 1 || proto.GetListOfServices()[0] != "GNSS Stations" {
		t.Errorf("services mismatch: %v", proto.GetListOfServices())
	}
	if !proto.GetIsOfficial() || !proto.GetIsPublic() {
		t.Errorf("flags mismatch: official=%v public=%v", proto.GetIsOfficial(), proto.GetIsPublic())
	}
	if proto.GetOwnerId() != "user-123" {
		t.Errorf("owner mismatch: %s", proto.GetOwnerId())
	}
	if !proto.GetCreatedAt().AsTime().Equal(now) {
		t.Errorf("created_at mismatch")
	}
}

func TestToDomainConfigurationInputRejectsInvalidJSON(t *testing.T) {
	if _, err := toDomainConfigurationInput([]byte(`{invalid`), nil, nil, nil, false, true); err == nil {
		t.Error("expected error for invalid configuration JSON")
	}
	if _, err := toDomainConfigurationInput([]byte(`{}`), nil, nil, nil, true, false); err == nil {
		t.Error("expected error for official non-public input")
	}
}

func TestCallerFromContext(t *testing.T) {
	if _, ok := callerFromContext(context.Background()); ok {
		t.Error("expected anonymous without metadata")
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDMetadataKey, "user-123"))
	caller, ok := callerFromContext(ctx)
	if !ok || caller.UserID != "user-123" || caller.Admin {
		t.Errorf("expected non-admin caller, got %+v, %v", caller, ok)
	}
	adminCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDMetadataKey, "admin-1", adminMetadataKey, "true"))
	caller, ok = callerFromContext(adminCtx)
	if !ok || !caller.Admin {
		t.Errorf("expected admin caller, got %+v, %v", caller, ok)
	}
}

func TestCanReadCanWrite(t *testing.T) {
	private := repository.ConfigurationRecord{IsPublic: false, OwnerID: "owner-1"}
	public := repository.ConfigurationRecord{IsPublic: true, OwnerID: "owner-1"}
	owner := Caller{UserID: "owner-1"}
	admin := Caller{UserID: "admin-1", Admin: true}
	other := Caller{UserID: "other-1"}

	if !canRead(Caller{}, public) || canRead(Caller{}, private) {
		t.Error("anonymous may read public only")
	}
	if !canRead(owner, private) || !canRead(admin, private) || canRead(other, private) {
		t.Error("private readable by owner or admin only")
	}
	if !canWrite(owner, private) || !canWrite(admin, private) || canWrite(other, private) {
		t.Error("writable by owner or admin only")
	}
}

func TestResolveIsPublic(t *testing.T) {
	if !resolveIsPublic(nil) {
		t.Error("absent visibility must default to public")
	}
	hidden := false
	if resolveIsPublic(&hidden) {
		t.Error("explicit false must stay private")
	}
}
