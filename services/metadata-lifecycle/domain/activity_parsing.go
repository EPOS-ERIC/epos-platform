package domain

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func NewActivityFilter(value ActivityFilter) (ActivityFilter, error) {
	if value.GroupID != nil && *value.GroupID == uuid.Nil {
		return ActivityFilter{}, apperror.Field("groupId", "Must not be the nil UUID.", nil)
	}

	if value.DateFrom != nil && value.DateTo != nil && value.DateFrom.After(*value.DateTo) {
		return ActivityFilter{}, apperror.Field("dateFrom", "Must be on or before `dateTo`.", nil)
	}

	return value, nil
}
