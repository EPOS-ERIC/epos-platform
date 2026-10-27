// Package schema extracts output parameters from JSON, XML, and CSV payloads.
package schema

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
)

const (
	maxDepth       = 4
	maxArraySample = 10
	maxParameters  = 256
	maxCSVColumns  = 200
)

var datetimeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Extract reports whether payload parsed as a supported format separately
// from its parameters, so valid empty schemas can replace prior values.
func Extract(contentType string, payload []byte) ([]domain.OutputParameter, bool) {
	base, _, _ := strings.Cut(contentType, ";")
	base = strings.ToLower(strings.TrimSpace(base))
	switch {
	case base == "application/json" || strings.HasSuffix(base, "+json"):
		return extractJSON(payload)
	case base == "application/xml" || base == "text/xml" || strings.HasSuffix(base, "+xml"):
		return extractXML(payload)
	case base == "text/csv":
		return extractCSV(payload)
	default:
		return nil, false
	}
}

type collector struct {
	parameters []domain.OutputParameter
	seen       map[string]struct{}
}

func (c *collector) add(name, kind string) {
	if kind == "" || len(c.parameters) >= maxParameters {
		return
	}
	if _, exists := c.seen[name]; exists {
		return
	}
	c.seen[name] = struct{}{}
	c.parameters = append(c.parameters, domain.OutputParameter{Name: name, Kind: kind})
}

func extractJSON(payload []byte) ([]domain.OutputParameter, bool) {
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false
	}
	output := &collector{seen: map[string]struct{}{}}
	walkJSON(decoded, "", 0, output)
	return output.parameters, true
}

func walkJSON(value any, prefix string, depth int, output *collector) {
	if len(output.parameters) >= maxParameters {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(typed)) {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			field := typed[key]
			output.add(path, jsonKind(field))
			if depth < maxDepth {
				walkJSON(field, path, depth+1, output)
			}
		}
	case []any:
		for index, element := range typed {
			if index >= maxArraySample {
				break
			}
			if _, ok := element.(map[string]any); ok {
				walkJSON(element, prefix, depth, output)
			}
		}
	}
}

func jsonKind(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case bool:
		return "BOOLEAN"
	case string:
		if isDatetime(typed) {
			return "DATETIME"
		}
		return "STRING"
	case float64:
		if typed == math.Trunc(typed) && math.Abs(typed) < 9e15 {
			return "INTEGER"
		}
		return "FLOAT"
	case map[string]any, []any:
		return "JSON"
	default:
		return ""
	}
}

func isDatetime(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, layout := range datetimeLayouts {
		if _, err := time.Parse(layout, value); err == nil {
			return true
		}
	}
	return false
}

func extractXML(payload []byte) ([]domain.OutputParameter, bool) {
	decoder := xml.NewDecoder(strings.NewReader(string(payload)))
	output := &collector{seen: map[string]struct{}{}}
	var stack []string
	rootSeen := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return output.parameters, rootSeen
			}
			return nil, false
		}
		switch typed := token.(type) {
		case xml.StartElement:
			if len(stack) == 0 {
				rootSeen = true
			}
			stack = append(stack, typed.Name.Local)
			if len(stack) <= maxDepth {
				path := strings.Join(stack, ".")
				output.add(path, "STRING")
				for _, attribute := range typed.Attr {
					if attribute.Name.Local == "xmlns" || attribute.Name.Space == "xmlns" {
						continue
					}
					output.add(path+".@"+attribute.Name.Local, "STRING")
				}
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
}

func extractCSV(payload []byte) ([]domain.OutputParameter, bool) {
	reader := csv.NewReader(strings.NewReader(string(payload)))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, false
	}
	output := &collector{seen: map[string]struct{}{}}
	for _, field := range header {
		if len(output.parameters) >= maxCSVColumns {
			break
		}
		name := strings.TrimSpace(field)
		if name != "" {
			output.add(name, "STRING")
		}
	}
	return output.parameters, true
}
