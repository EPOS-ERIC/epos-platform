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
	got := parameterMap(Extract("application/json; charset=utf-8", payload))
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

func TestExtractJSONMergesArrayElementKeys(t *testing.T) {
	payload := []byte(`{"results": [{"a": 1}, {"b": "x", "a": 2}]}`)
	got := parameterMap(Extract("application/json", payload))
	if got["results"] != "JSON" || got["results.a"] != "INTEGER" || got["results.b"] != "STRING" {
		t.Fatalf("parameters = %v", got)
	}
}

func TestExtractJSONRejectsMalformed(t *testing.T) {
	if parameters := Extract("application/json", []byte(`{"a":`)); parameters != nil {
		t.Fatalf("parameters = %v, want nil", parameters)
	}
}

func TestExtractJSONSuffixContentType(t *testing.T) {
	got := parameterMap(Extract("application/vnd.api+json", []byte(`{"a": 1}`)))
	if got["a"] != "INTEGER" {
		t.Fatalf("parameters = %v", got)
	}
}

func TestExtractXMLCollectsElementsAndAttributes(t *testing.T) {
	payload := []byte(`<search><results total="2"><item id="1"><title>Etna</title></item></results></search>`)
	got := parameterMap(Extract("application/xml", payload))
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
	payload := []byte("name,height,active\nEtna,3357,true\n")
	got := parameterMap(Extract("text/csv", payload))
	if len(got) != 3 || got["name"] != "STRING" || got["height"] != "STRING" || got["active"] != "STRING" {
		t.Fatalf("parameters = %v", got)
	}
}

func TestExtractSkipsUnsupportedFormats(t *testing.T) {
	for _, contentType := range []string{"image/tiff", "application/zip", "application/octet-stream", ""} {
		if parameters := Extract(contentType, []byte("binary")); parameters != nil {
			t.Fatalf("content type %q: parameters = %v, want nil", contentType, parameters)
		}
	}
}
