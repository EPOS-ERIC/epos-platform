// Package metadatareader parses metadata-lifecycle property maps into typed values used by gateway mappers.
package metadatareader

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/gatewayhref"
)

var log = logging.Get("metadata-reader")

// Item represents one normalized property value from a metadata-lifecycle property map.
type Item struct {
	ID    string
	Type  string
	Title string
	Href  string
	Value string
}

// Reader provides tolerant accessors over metadata-lifecycle property maps.
type Reader struct {
	properties map[string]any
	lookup     map[string]any
}

// New builds a Reader from raw metadata-lifecycle properties.
func New(properties map[string]any) Reader {
	lookup := make(map[string]any, len(properties))
	for key, value := range properties {
		lookup[normalizePropertyKey(key)] = value
	}

	return Reader{properties: properties, lookup: lookup}
}

// Items returns normalized property items for key.
func (r Reader) Items(key string) ([]Item, error) {
	raw, ok := r.properties[key]
	if !ok {
		log.Debug("items direct key miss, trying normalized lookup", "key", key, "properties", r.properties)
		raw, ok = r.lookup[normalizePropertyKey(key)]
	}
	if !ok {
		log.Debug("items key not found", "key", key, "properties", r.properties)
		return nil, nil
	}

	values := []any{raw}
	if raw == nil {
		values = nil
	} else if typedValues, ok := raw.([]any); ok {
		values = typedValues
	}

	items := make([]Item, 0, len(values))
	for i, value := range values {
		switch typed := value.(type) {
		case string:
			items = append(items, Item{ID: "", Type: "", Title: "", Href: "", Value: typed})
		case map[string]any:
			items = append(items, Item{
				ID:    mapString(typed, "id"),
				Type:  mapString(typed, "type"),
				Title: FirstNonEmpty(mapString(typed, "title"), mapString(typed, "name")),
				Href:  mapString(typed, "href"),
				Value: mapString(typed, "value"),
			})
		default:
			log.Debug("items parse failed: unexpected value type", "key", key, "index", i, "type", fmt.Sprintf("%T", value))
			return nil, fmt.Errorf("item %d: expected string or object, got %T", i, value)
		}
	}

	return items, nil
}

// ResourceRefs returns gateway resource references for key.
func (r Reader) ResourceRefs(key string, resourceType api.ResourceType) ([]api.ResourceRef, error) {
	items, err := r.Items(key)
	if err != nil || len(items) == 0 {
		if err != nil {
			log.Debug("resource refs parse failed", "key", key, "resourceType", resourceType, "error", err)
		} else {
			log.Debug("resource refs empty", "key", key, "resourceType", resourceType)
		}
		return nil, err
	}

	refs := make([]api.ResourceRef, 0, len(items))
	for i, item := range items {
		if item.ID == "" {
			log.Debug("resource refs item missing id", "key", key, "index", i)
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

// PassthroughRefs returns references while preserving upstream href values when available.
func (r Reader) PassthroughRefs(key string) ([]api.ResourceRef, error) {
	items, err := r.Items(key)
	if err != nil || len(items) == 0 {
		if err != nil {
			log.Debug("passthrough refs parse failed", "key", key, "error", err)
		} else {
			log.Debug("passthrough refs empty", "key", key)
		}
		return nil, err
	}

	refs := make([]api.ResourceRef, 0, len(items))
	for i, item := range items {
		if item.ID == "" {
			log.Debug("passthrough refs item missing id", "key", key, "index", i)
			return nil, fmt.Errorf("item %d: missing id", i)
		}

		refs = append(refs, api.ResourceRef{
			ID:   item.ID,
			Name: item.Title,
			Href: FirstNonEmpty(item.Href, item.ID),
		})
	}

	return refs, nil
}

// OptionalResourceRef returns first resource reference for key, if present.
func (r Reader) OptionalResourceRef(key string, resourceType api.ResourceType) (*api.ResourceRef, error) {
	refs, err := r.ResourceRefs(key, resourceType)
	if err != nil || len(refs) == 0 {
		return nil, err
	}

	return &refs[0], nil
}

// RequiredString returns non-empty string value for key.
func (r Reader) RequiredString(key string) (string, error) {
	value := r.OptionalString(key)
	if value == "" {
		return "", fmt.Errorf("missing field '%s'", key)
	}

	return value, nil
}

// RequiredStringOneOf returns first non-empty value among keys.
func (r Reader) RequiredStringOneOf(keys ...string) (string, error) {
	for _, key := range keys {
		value := r.OptionalString(key)
		if value != "" {
			return value, nil
		}
	}

	return "", fmt.Errorf("missing fields %v", keys)
}

// OptionalStringOneOf returns first non-empty value among keys.
func (r Reader) OptionalStringOneOf(keys ...string) string {
	for _, key := range keys {
		value := r.OptionalString(key)
		if value != "" {
			return value
		}
	}

	return ""
}

// OptionalString returns first normalized string value for key.
func (r Reader) OptionalString(key string) string {
	items, err := r.Items(key)
	if err != nil || len(items) == 0 {
		return ""
	}

	first := items[0]
	return FirstNonEmpty(first.Value, first.Title, first.ID)
}

// Strings returns all non-empty normalized string values for key.
func (r Reader) Strings(key string) ([]string, error) {
	items, err := r.Items(key)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	values := make([]string, 0, len(items))
	for _, item := range items {
		value := FirstNonEmpty(item.Value, item.Title, item.ID)
		if value == "" {
			continue
		}
		values = append(values, value)
	}

	return values, nil
}

// StringsOneOf returns first non-empty string slice among keys.
func (r Reader) StringsOneOf(keys ...string) ([]string, error) {
	for _, key := range keys {
		values, err := r.Strings(key)
		if err != nil {
			return nil, err
		}
		if len(values) > 0 {
			return values, nil
		}
	}

	return nil, nil
}

// OptionalTime parses an optional time value for key.
func (r Reader) OptionalTime(key string) (*time.Time, error) {
	value := r.OptionalString(key)
	if value == "" {
		return nil, nil
	}

	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, time.DateOnly} {
		parsedTime, err := time.Parse(layout, value)
		if err == nil {
			return &parsedTime, nil
		}
	}

	log.Debug("optional time parse failed", "key", key, "value", value)
	return nil, fmt.Errorf("parse '%s': unsupported time format '%s'", key, value)
}

// OptionalTimeOneOf parses the first present time value among keys.
func (r Reader) OptionalTimeOneOf(keys ...string) (*time.Time, error) {
	for _, key := range keys {
		value, err := r.OptionalTime(key)
		if err != nil {
			return nil, err
		}
		if value != nil {
			return value, nil
		}
	}

	return nil, nil
}

// OptionalVersioningStatus parses an optional versioning status for key.
func (r Reader) OptionalVersioningStatus(key string) (api.VersioningStatus, error) {
	value := r.OptionalString(key)
	if value == "" {
		return "", nil
	}

	status, err := api.NewVersioningStatusFromValue(value)
	if err != nil {
		log.Debug("optional versioning status parse failed", "key", key, "value", value, "error", err)
		return "", fmt.Errorf("parse '%s' versioning status: %w", key, err)
	}

	return status, nil
}

// OptionalBool parses an optional boolean value for key.
func (r Reader) OptionalBool(key string) (*bool, error) {
	value := r.OptionalString(key)
	if value == "" {
		return nil, nil
	}

	parsedValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Debug("optional bool parse failed", "key", key, "value", value, "error", err)
		return nil, fmt.Errorf("parse '%s' bool: %w", key, err)
	}

	return &parsedValue, nil
}

// RequiredBool parses a required boolean value for key.
func (r Reader) RequiredBool(key string) (bool, error) {
	value, err := r.OptionalBool(key)
	if err != nil {
		return false, err
	}
	if value == nil {
		return false, fmt.Errorf("missing field '%s'", key)
	}

	return *value, nil
}

// OptionalBoolOneOf parses first present boolean value among keys.
func (r Reader) OptionalBoolOneOf(keys ...string) (*bool, error) {
	for _, key := range keys {
		value, err := r.OptionalBool(key)
		if err != nil {
			return nil, err
		}
		if value != nil {
			return value, nil
		}
	}

	return nil, nil
}

// RequiredBoolOneOf parses first present required boolean value among keys.
func (r Reader) RequiredBoolOneOf(keys ...string) (bool, error) {
	value, err := r.OptionalBoolOneOf(keys...)
	if err != nil {
		return false, err
	}
	if value == nil {
		return false, fmt.Errorf("missing fields %v", keys)
	}

	return *value, nil
}

// ItemsOneOf returns first non-empty item list among keys.
func (r Reader) ItemsOneOf(keys ...string) ([]Item, error) {
	for _, key := range keys {
		items, err := r.Items(key)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return items, nil
		}
	}

	return nil, nil
}

// ParseOptionalInt32 parses an optional int32 string value.
func ParseOptionalInt32(value string) (int32, error) {
	if value == "" {
		return 0, nil
	}

	parsedValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		log.Debug("optional int32 parse failed", "value", value, "error", err)
		return 0, fmt.Errorf("parse int32 '%s': %w", value, err)
	}

	return int32(parsedValue), nil
}

// FirstNonEmpty returns the first non-empty string from values.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func mapString(values map[string]any, key string) string {
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
