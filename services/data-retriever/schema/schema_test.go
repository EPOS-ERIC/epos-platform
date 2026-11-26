package schema

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
)

func parameterMap(parameters []domain.OutputParameter) map[string]string {
	result := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		result[parameter.Name] = parameter.Kind
	}
	return result
}

func TestExtractJSONInfersKinds(t *testing.T) {
	payload := []byte(`{
		"name": "Etna",
		"active": true,
		"height": 3357,
		"prominence": 3357.5,
		"last_eruption": "2026-09-14",
		"location": {"lat": 37.75, "lon": 14.99},
		"tags": ["volcano", "sicily"],
		"nothing": null
	}`)
	parameters, parsed := Extract("application/json; charset=utf-8", payload)
	if !parsed {
		t.Fatal("Extract() reported valid JSON as unparsable")
	}
	got := parameterMap(parameters)
	want := map[string]string{
		"name":          "STRING",
		"active":        "BOOLEAN",
		"height":        "INTEGER",
		"prominence":    "FLOAT",
		"last_eruption": "DATETIME",
		"location":      "JSON",
		"location.lat":  "FLOAT",
		"location.lon":  "FLOAT",
		"tags":          "JSON",
	}
	if len(got) != len(want) {
		t.Fatalf("parameters = %v, want %v", got, want)
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Errorf("parameter %q = %q, want %q", name, got[name], kind)
		}
	}
}

func TestExtractMergesArrayElementKeys(t *testing.T) {
	parameters, parsed := Extract("application/json", []byte(`{"results":[{"a":1},{"b":"x","a":2}]}`))
	got := parameterMap(parameters)
	if !parsed || got["results"] != "JSON" || got["results.a"] != "INTEGER" || got["results.b"] != "STRING" {
		t.Fatalf("parameters = %v, parsed = %t", got, parsed)
	}
}

func TestExtractDistinguishesInvalidFromValidEmpty(t *testing.T) {
	for _, test := range []struct {
		name        string
		contentType string
		payload     []byte
		wantParsed  bool
	}{
		{"malformed JSON", "application/json", []byte(`{"a":`), false},
		{"valid empty JSON", "application/json", []byte(`{}`), true},
		{"malformed XML", "application/xml", []byte(`<root>`), false},
		{"unmatched XML close", "application/xml", []byte(`</root>`), false},
		{"unsupported format", "image/tiff", []byte("binary"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			parameters, parsed := Extract(test.contentType, test.payload)
			if parsed != test.wantParsed {
				t.Fatalf("parsed = %t, want %t", parsed, test.wantParsed)
			}
			if test.name == "valid empty JSON" && len(parameters) != 0 {
				t.Fatalf("parameters = %v, want empty", parameters)
			}
		})
	}
}

func TestExtractJSONContentTypeSuffix(t *testing.T) {
	parameters, parsed := Extract("application/vnd.api+json", []byte(`{"a":1}`))
	if !parsed || parameterMap(parameters)["a"] != "INTEGER" {
		t.Fatalf("parameters = %v, parsed = %t", parameters, parsed)
	}
}

func TestExtractXMLCollectsElementsAndAttributes(t *testing.T) {
	payload := []byte(`<search><results total="2"><item id="1"><title>Etna</title></item></results></search>`)
	parameters, parsed := Extract("application/xml", payload)
	if !parsed {
		t.Fatal("Extract() reported valid XML as unparsable")
	}
	got := parameterMap(parameters)
	want := map[string]string{
		"search":                "STRING",
		"search.results":        "STRING",
		"search.results.@total": "STRING",
		"search.results.item":   "STRING",
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Errorf("parameter %q = %q, want %q", name, got[name], kind)
		}
	}
}

func TestExtractCSVUsesHeaderRow(t *testing.T) {
	parameters, parsed := Extract("text/csv", []byte("name,height,active\nEtna,3357,true\n"))
	got := parameterMap(parameters)
	if !parsed || len(got) != 3 || got["name"] != "STRING" || got["height"] != "STRING" || got["active"] != "STRING" {
		t.Fatalf("parameters = %v, parsed = %t", got, parsed)
	}
}

func TestExtractSkipsUnsupportedFormats(t *testing.T) {
	for _, contentType := range []string{"image/tiff", "application/zip", "application/octet-stream", ""} {
		if parameters, parsed := Extract(contentType, []byte("binary")); parsed || parameters != nil {
			t.Fatalf("content type %q: parameters = %v, parsed = %t", contentType, parameters, parsed)
		}
	}
}
