package identity

import "strings"

func nullString(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isUniqueViolation(err error, constraint string) bool {
	return err != nil && strings.Contains(err.Error(), constraint)
}
