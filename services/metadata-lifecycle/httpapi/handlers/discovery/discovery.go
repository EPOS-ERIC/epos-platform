// Package discovery exposes resource discovery HTTP handlers.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
)

type Service interface {
	Search(context.Context, *string, discovery.Filter) ([]discovery.Item, error)
}

type Handler struct{ service Service }

var log = logging.Get("discovery-handler")

func New(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	actorID, present, ok := transport.OptionalActor(w, r)
	if !ok {
		return
	}
	var actor *string
	if present {
		actor = &actorID
	}

	filter, err := filterFromRequest(r)
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	items, err := h.service.Search(r.Context(), actor, filter)
	if err != nil {
		status := http.StatusInternalServerError
		title := "internal_server_error"
		detail := "discovery search failed"
		switch {
		case errors.Is(err, discovery.ErrInvalid):
			log.DebugContext(r.Context(), "discovery search invalid", "error", err)
			status, title, detail = http.StatusBadRequest, "invalid_request", err.Error()
		case errors.Is(err, discovery.ErrDenied):
			log.DebugContext(r.Context(), "discovery search denied", "error", err)
			status, title, detail = http.StatusUnauthorized, "unauthorized", "authentication is required for non-published discovery results"
		default:
			log.ErrorContext(r.Context(), "discovery search failed", "error", err)
		}
		transport.WriteProblem(w, status, title, detail)

		return
	}

	transport.WriteJSON(w, http.StatusOK, items)
}

func filterFromRequest(r *http.Request) (discovery.Filter, error) {
	values := r.URL.Query()
	statuses := []string{"PUBLISHED"}
	if raw := split(values["versioningStatus"]); len(raw) != 0 {
		statuses = raw
	}
	parsedStatuses, err := parseStatuses(statuses)
	if err != nil {
		return discovery.Filter{}, fmt.Errorf("parse versioningStatus: %w", err)
	}
	types, err := parseTypes(split(values["type"]))
	if err != nil {
		return discovery.Filter{}, fmt.Errorf("parse type: %w", err)
	}
	providers, err := parseUUIDs(split(values["provider"]))
	if err != nil {
		return discovery.Filter{}, fmt.Errorf("parse provider: %w", err)
	}
	from, to, err := parseTimes(values)
	if err != nil {
		return discovery.Filter{}, fmt.Errorf("parse date range: %w", err)
	}
	spatialFilter, err := parseSpatialFilter(values.Get("spatialFilter"))
	if err != nil {
		return discovery.Filter{}, err
	}
	return discovery.Filter{
		Query:              strings.TrimSpace(values.Get("q")),
		Types:              types,
		SpatialFilter:      spatialFilter,
		DateFrom:           from,
		DateTo:             to,
		Providers:          providers,
		Keywords:           split(values["keywords"]),
		VersioningStatuses: parsedStatuses,
	}, nil
}

func parseSpatialFilter(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	geometry, err := wkt.Unmarshal(value)
	if err != nil {
		return "", fmt.Errorf("invalid spatialFilter: %w", discovery.ErrInvalid)
	}
	switch geometry.(type) {
	case *geom.Point, *geom.Polygon:
		return value, nil
	default:
		return "", fmt.Errorf("spatialFilter must be POINT or POLYGON: %w", discovery.ErrInvalid)
	}
}

func parseStatuses(values []string) ([]discovery.Status, error) {
	result := make([]discovery.Status, 0, len(values))
	seen := map[discovery.Status]bool{}
	for _, value := range values {
		status := discovery.Status(strings.ToUpper(strings.TrimSpace(value)))
		switch status {
		case discovery.Published, discovery.Draft, discovery.Submitted, discovery.Archived, discovery.Rejected:
		default:
			return nil, fmt.Errorf("unsupported versioningStatus %q: %w", value, discovery.ErrInvalid)
		}
		if !seen[status] {
			result = append(result, status)
			seen[status] = true
		}
	}
	return result, nil
}

func parseTypes(values []string) ([]discovery.ResourceType, error) {
	result := make([]discovery.ResourceType, 0, len(values))
	seen := map[discovery.ResourceType]bool{}
	for _, value := range values {
		typ := discovery.ResourceType(strings.TrimSpace(value))
		if typ == "" {
			continue
		}
		switch typ {
		case discovery.Dataset, discovery.SoftwareApplication, discovery.SoftwareSourceCode:
		default:
			return nil, fmt.Errorf("unsupported resource type %q: %w", value, discovery.ErrInvalid)
		}
		if !seen[typ] {
			result = append(result, typ)
			seen[typ] = true
		}
	}
	return result, nil
}

func parseUUIDs(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("invalid UUID %q: %w", value, discovery.ErrInvalid)
		}
		result = append(result, id)
	}
	return result, nil
}

func parseTimes(values url.Values) (*time.Time, *time.Time, error) {
	parse := func(key string) (*time.Time, error) {
		value := strings.TrimSpace(values.Get(key))
		if value == "" {
			return nil, nil
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", key, discovery.ErrInvalid)
		}
		return &parsed, nil
	}
	from, err := parse("dateFrom")
	if err != nil {
		return nil, nil, err
	}
	to, err := parse("dateTo")
	if err != nil {
		return nil, nil, err
	}
	return from, to, nil
}

func split(values []string) []string {
	var result []string
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			if value := strings.TrimSpace(part); value != "" {
				result = append(result, value)
			}
		}
	}
	return result
}
