package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

var log = logging.Get("handlers")

const (
	defaultPageSize   int32 = 10
	defaultPageNumber int32 = 1
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error("failed to encode JSON response", "status", status, "error", err)
	}
}

func writeError(ctx context.Context, w http.ResponseWriter, status int, code, message string, internalErr error) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)

	problem := api.Problem{
		Title:       code,
		Status:      status,
		Detail:      message,
		DebugDetail: "",
		Instance:    "",
	}
	if internalErr != nil && log.Enabled(ctx, slog.LevelDebug) {
		problem.DebugDetail = internalErr.Error()
	}

	if err := json.NewEncoder(w).Encode(problem); err != nil {
		log.ErrorContext(ctx, "failed to encode problem response", "status", status, "error", err)
	}
}

func writeBadRequest(ctx context.Context, w http.ResponseWriter, err error) {
	writeError(ctx, w, http.StatusBadRequest, "invalid_query_parameter", err.Error(), nil)
}

func writeNotImplemented(ctx context.Context, w http.ResponseWriter) {
	writeError(ctx, w, http.StatusNotImplemented, "not_implemented", "endpoint not implemented", nil)
}

func parsePageSize(values url.Values) (int32, error) {
	minPageValue := int32(1)
	maxPageSize := int32(100)
	return parseInt32(values, "pageSize", defaultPageSize, &minPageValue, &maxPageSize)
}

func parsePageNumber(values url.Values) (int32, error) {
	minPageValue := int32(1)
	return parseInt32(values, "pageNumber", defaultPageNumber, &minPageValue, nil)
}

func parseInt32(values url.Values, name string, defaultValue int32, minValue, maxValue *int32) (int32, error) {
	if !values.Has(name) {
		return defaultValue, nil
	}

	parsedValue, err := strconv.ParseInt(values.Get(name), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", queryParamLabel(name))
	}

	value := int32(parsedValue)
	if minValue != nil && value < *minValue {
		return 0, fmt.Errorf("%s must be >= %d", queryParamLabel(name), *minValue)
	}
	if maxValue != nil && value > *maxValue {
		return 0, fmt.Errorf("%s must be <= %d", queryParamLabel(name), *maxValue)
	}

	return value, nil
}

func parseDateTime(values url.Values, name string) (time.Time, error) {
	if !values.Has(name) {
		return time.Time{}, nil
	}

	value, err := time.Parse(time.RFC3339, values.Get(name))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be rfc3339", queryParamLabel(name))
	}

	return value, nil
}

func parseCSVStrings(values url.Values, name string) []string {
	if !values.Has(name) {
		return nil
	}

	parts := strings.Split(values.Get(name), ",")
	parsed := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parsed = append(parsed, trimmed)
	}

	if len(parsed) == 0 {
		return nil
	}

	return parsed
}

func parseCSVResourceTypes(values url.Values, name string) ([]api.ResourceType, error) {
	rawValues := parseCSVStrings(values, name)
	if len(rawValues) == 0 {
		return nil, nil
	}

	parsed := make([]api.ResourceType, 0, len(rawValues))
	for _, rawValue := range rawValues {
		value, err := api.NewResourceTypeFromValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("unsupported search type: %s", rawValue)
		}
		parsed = append(parsed, value)
	}

	return parsed, nil
}

func parseCSVVersioningStatuses(values url.Values, name string, defaultValues []api.VersioningStatus) ([]api.VersioningStatus, error) {
	rawValues := parseCSVStrings(values, name)
	if len(rawValues) == 0 {
		if defaultValues == nil {
			return nil, nil
		}

		defaults := make([]api.VersioningStatus, len(defaultValues))
		copy(defaults, defaultValues)
		return defaults, nil
	}

	parsed := make([]api.VersioningStatus, 0, len(rawValues))
	for _, rawValue := range rawValues {
		value, err := api.NewVersioningStatusFromValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("versioning status must be one of draft, submitted, published, archived")
		}
		parsed = append(parsed, value)
	}

	return parsed, nil
}

func queryParamLabel(name string) string {
	switch name {
	case "pageSize":
		return "page size"
	case "pageNumber":
		return "page number"
	case "dateFrom":
		return "date from"
	case "dateTo":
		return "date to"
	case "versioningStatus":
		return "versioning status"
	default:
		return name
	}
}
