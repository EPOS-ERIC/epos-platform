package handlers

import (
	"fmt"
	"time"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toClientStringPtr(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return &v.Value
}

func toAPIRequiredNilString(value *string) gatewayapi.NilString {
	if value == nil {
		return gatewayapi.NilString{Null: true}
	}
	return gatewayapi.NewNilString(*value)
}

func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid upstream %s: %w", field, err)
	}
	return id, nil
}

func fromProtoTimestamp(field string, value *timestamppb.Timestamp) (time.Time, error) {
	if value == nil {
		return time.Time{}, fmt.Errorf("decode upstream %s: timestamp is required", field)
	}
	if err := value.CheckValid(); err != nil {
		return time.Time{}, fmt.Errorf("decode upstream %s: %w", field, err)
	}
	return value.AsTime(), nil
}

func toProtoKeywords(values []gatewayapi.Keyword) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}

func toAPIKeywords(values []string) []gatewayapi.Keyword {
	result := make([]gatewayapi.Keyword, len(values))
	for i, value := range values {
		result[i] = gatewayapi.Keyword(value)
	}
	return result
}
