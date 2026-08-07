// Package metadatareader parses metadata-lifecycle property maps into typed values used by gateway mappers.
package metadatareader

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/resourcehref"
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
			Href: resourcehref.Build(resourceType, item.ID),
		})
	}

	return refs, nil
}

// RequiredString returns non-empty string value for key.
func (r Reader) RequiredString(key string) (string, error) {
	value := r.OptionalString(key)
	if value == "" {
		return "", fmt.Errorf("missing field '%s'", key)
	}

	return value, nil
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
