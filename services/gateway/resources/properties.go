package resources

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/gatewayhref"
)

type propertyItem struct {
	ID    string
	Title string
	Href  string
	Value string
}

type propertyReader struct {
	properties map[string]any
	lookup     map[string]any
}

func newPropertyReader(properties map[string]any) propertyReader {
	lookup := make(map[string]any, len(properties))
	for key, value := range properties {
		lookup[normalizePropertyKey(key)] = value
	}

	return propertyReader{
		properties: properties,
		lookup:     lookup,
	}
}

func (r propertyReader) items(key string) ([]propertyItem, error) {
	raw, ok := r.properties[key]
	if !ok {
		raw, ok = r.lookup[normalizePropertyKey(key)]
	}
	if !ok {
		return nil, nil
	}

	values := []any{raw}
	if raw == nil {
		values = nil
	} else if typedValues, ok := raw.([]any); ok {
		values = typedValues
	}

	items := make([]propertyItem, 0, len(values))
	for i, value := range values {
		switch typed := value.(type) {
		case string:
			items = append(items, propertyItem{Value: typed})
		case map[string]any:
			item := propertyItem{
				ID:    r.mapString(typed, "id"),
				Title: firstNonEmpty(r.mapString(typed, "title"), r.mapString(typed, "name")),
				Href:  r.mapString(typed, "href"),
				Value: r.mapString(typed, "value"),
			}
			items = append(items, item)
		default:
			return nil, fmt.Errorf("item %d: expected string or object, got %T", i, value)
		}
	}

	return items, nil
}

func (r propertyReader) resourceRefs(key string, resourceType api.ResourceType) ([]api.ResourceRef, error) {
	items, err := r.items(key)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	refs := make([]api.ResourceRef, 0, len(items))
	for i, item := range items {
		if item.ID == "" {
			return nil, fmt.Errorf("item %d: missing id", i)
		}

		refs = append(refs, api.ResourceRef{
			ID:   item.ID,
			Name: item.Title,
			Href: gatewayhref.Build(resourceType, item.ID),
		})
	}

	return refs, nil
}

func (r propertyReader) passthroughRefs(key string) ([]api.ResourceRef, error) {
	items, err := r.items(key)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	refs := make([]api.ResourceRef, 0, len(items))
	for i, item := range items {
		if item.ID == "" {
			return nil, fmt.Errorf("item %d: missing id", i)
		}

		refs = append(refs, api.ResourceRef{
			ID:   item.ID,
			Name: item.Title,
			Href: firstNonEmpty(item.Href, item.ID),
		})
	}

	return refs, nil
}

func (r propertyReader) optionalResourceRef(key string, resourceType api.ResourceType) (*api.ResourceRef, error) {
	refs, err := r.resourceRefs(key, resourceType)
	if err != nil || len(refs) == 0 {
		return nil, err
	}

	return &refs[0], nil
}

func (r propertyReader) requiredString(key string) (string, error) {
	value := r.optionalString(key)
	if value == "" {
		return "", fmt.Errorf("missing field '%s'", key)
	}

	return value, nil
}

func (r propertyReader) optionalString(key string) string {
	items, err := r.items(key)
	if err != nil || len(items) == 0 {
		return ""
	}

	first := items[0]
	return firstNonEmpty(first.Value, first.Title, first.ID)
}

func (r propertyReader) strings(key string) ([]string, error) {
	items, err := r.items(key)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	values := make([]string, 0, len(items))
	for _, item := range items {
		value := firstNonEmpty(item.Value, item.Title, item.ID)
		if value == "" {
			continue
		}
		values = append(values, value)
	}

	return values, nil
}

func (r propertyReader) optionalTime(key string) (*time.Time, error) {
	value := r.optionalString(key)
	if value == "" {
		return nil, nil
	}

	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, time.DateOnly} {
		parsedTime, err := time.Parse(layout, value)
		if err == nil {
			return &parsedTime, nil
		}
	}

	return nil, fmt.Errorf("parse '%s': unsupported time format '%s'", key, value)
}

func (r propertyReader) optionalVersioningStatus(key string) (api.VersioningStatus, error) {
	value := r.optionalString(key)
	if value == "" {
		return "", nil
	}

	status, err := api.NewVersioningStatusFromValue(value)
	if err != nil {
		return "", fmt.Errorf("parse '%s' versioning status: %w", key, err)
	}

	return status, nil
}

func (r propertyReader) optionalBool(key string) (*bool, error) {
	value := r.optionalString(key)
	if value == "" {
		return nil, nil
	}

	parsedValue, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("parse '%s' bool: %w", key, err)
	}

	return &parsedValue, nil
}

func (r propertyReader) requiredBool(key string) (bool, error) {
	value, err := r.optionalBool(key)
	if err != nil {
		return false, err
	}
	if value == nil {
		return false, fmt.Errorf("missing field '%s'", key)
	}

	return *value, nil
}

func (r propertyReader) mapString(values map[string]any, key string) string {
	raw, ok := values[key]
	if !ok {
		return ""
	}

	stringValue, ok := raw.(string)
	if !ok {
		return ""
	}

	return stringValue
}

// normalizePropertyKey strips non-alphanumeric characters and lowercases the key for tolerant upstream lookups.
func normalizePropertyKey(key string) string {
	var normalized strings.Builder
	normalized.Grow(len(key))

	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(unicode.ToLower(r))
		}
	}

	return normalized.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
