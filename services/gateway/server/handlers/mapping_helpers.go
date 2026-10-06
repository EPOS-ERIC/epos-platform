package handlers

import (
	"fmt"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func toClientStringPtr(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return &v.Value
}

func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid upstream %s: %w", field, err)
	}
	return id, nil
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
