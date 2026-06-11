package service

import (
	"database/sql"
	"strings"

	"github.com/google/uuid"
)

func nullString(v string) sql.NullString {
	trimmed := strings.TrimSpace(v)

	if trimmed == "" {
		return sql.NullString{
			String: "",
			Valid:  false,
		}
	}

	return sql.NullString{
		String: trimmed,
		Valid:  true,
	}
}

func isUniqueViolation(err error, constraint string) bool {
	return err != nil && strings.Contains(err.Error(), constraint)
}

func parseUUID(id string) uuid.UUID {
	parsed, _ := uuid.Parse(id)
	return parsed
}
