package domain

import (
	"strings"
	"time"
)

func optionalTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func integerInRange(value int64, minimum, maximum *int64) bool {
	return (minimum == nil || value >= *minimum) && (maximum == nil || value <= *maximum)
}

func floatInRange(value float64, minimum, maximum *float64) bool {
	return (minimum == nil || value >= *minimum) && (maximum == nil || value <= *maximum)
}

func timeInRange(value time.Time, minimum, maximum *time.Time) bool {
	return (minimum == nil || !value.Before(*minimum)) && (maximum == nil || !value.After(*maximum))
}
