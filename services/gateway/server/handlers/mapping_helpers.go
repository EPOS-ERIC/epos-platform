package handlers

import (
	"fmt"

	"github.com/google/uuid"
)

func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid upstream %s: %w", field, err)
	}
	return id, nil
}
