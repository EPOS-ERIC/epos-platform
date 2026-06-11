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

func isAllowedUserStatus(status string) bool {
	switch status {
	case "ACTIVE", "INACTIVE", "BLOCKED":
		return true
	default:
		return false
	}
}

func isAllowedMembershipRole(role string) bool {
	switch role {
	case "VIEWER", "EDITOR", "REVIEWER", "ADMIN":
		return true
	default:
		return false
	}
}

func isAllowedMembershipStatus(status string) bool {
	switch status {
	case "PENDING", "ACTIVE", "REVOKED", "REJECTED":
		return true
	default:
		return false
	}
}

func parseUUID(id string) uuid.UUID {
	parsed, _ := uuid.Parse(id)
	return parsed
}
