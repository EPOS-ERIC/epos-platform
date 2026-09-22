// Package schema extracts output parameter schemas from upstream payloads.
// Supported formats are JSON documents, XML documents and CSV tables; every
// other content type yields no parameters. Extraction is deterministic:
// object keys are visited in sorted order and discovery stops at fixed depth
// and parameter caps.
package schema

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io"
	"math"
	"sort"
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
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Extract discovers output parameters from an upstream payload based on its
// content type. It returns nil when the format is unsupported or carries no
// discoverable fields.
func Extract(contentType string, payload []byte) []domain.OutputParameter {
	base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case base == "application/json" || strings.HasSuffix(base, "+json"):
		return extractJSON(payload)
	case base == "application/xml" || base == "text/xml" || strings.HasSuffix(base, "+xml"):
		return extractXML(payload)
	case base == "text/csv":
		return extractCSV(payload)
	default:
		return nil
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

func extractJSON(payload []byte) []domain.OutputParameter {
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	collector := &collector{seen: map[string]struct{}{}}
	walkJSON(decoded, "", 0, collector)
	return collector.parameters
}

func walkJSON(value any, prefix string, depth int, collector *collector) {
	if len(collector.parameters) >= maxParameters {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			field := typed[key]
			if kind, ok := jsonKind(field); ok {
				collector.add(path, kind)
			}
			if depth < maxDepth {
				walkJSON(field, path, depth+1, collector)
			}
		}
	case []any:
		for i, element := range typed {
			if i >= maxArraySample {
				break
			}
			if _, ok := element.(map[string]any); ok {
				walkJSON(element, prefix, depth, collector)
			}
		}
	}
}

func jsonKind(value any) (string, bool) {
	switch typed := value.(type) {
	case nil:
		return "", false
	case bool:
		return "BOOLEAN", true
	case string:
		if isDatetime(typed) {
			return "DATETIME", true
		}
		return "STRING", true
	case float64:
		if typed == math.Trunc(typed) && math.Abs(typed) < 9e15 {
			return "INTEGER", true
		}
		return "FLOAT", true
	case map[string]any:
		return "JSON", true
	case []any:
		return "JSON", true
	default:
		return "", false
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

func extractXML(payload []byte) []domain.OutputParameter {
	decoder := xml.NewDecoder(strings.NewReader(string(payload)))
	collector := &collector{seen: map[string]struct{}{}}
	var stack []string
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil
		}
		switch typed := token.(type) {
		case xml.StartElement:
			stack = append(stack, typed.Name.Local)
			if len(stack) <= maxDepth {
				path := strings.Join(stack, ".")
				collector.add(path, "STRING")
				for _, attr := range typed.Attr {
					if attr.Name.Local == "xmlns" || attr.Name.Space == "xmlns" {
						continue
					}
					collector.add(path+".@"+attr.Name.Local, "STRING")
				}
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return collector.parameters
}

func extractCSV(payload []byte) []domain.OutputParameter {
	reader := csv.NewReader(strings.NewReader(string(payload)))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil
	}
	collector := &collector{seen: map[string]struct{}{}}
	for _, field := range header {
		if len(collector.parameters) >= maxCSVColumns {
			break
		}
		name := strings.TrimSpace(field)
		if name == "" {
			continue
		}
		collector.add(name, "STRING")
	}
	return collector.parameters
}
