package dataset

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
)

func TestParameterConfigRoundTrip(t *testing.T) {
	defaultValue := "json"
	parameter := metadata.StringParameter{
		Metadata: metadata.ParameterMetadata{
			ID:       uuid.New(),
			Variable: "format",
			Property: "schema:format",
			Label:    "Format",
		},
		Default:       &defaultValue,
		AllowedValues: []string{"json", "csv"},
	}

	record, err := encodeParameter(parameter)
	if err != nil {
		t.Fatalf("encodeParameter() error = %v", err)
	}
	decoded, err := decodeParameter(record)
	if err != nil {
		t.Fatalf("decodeParameter() error = %v", err)
	}
	value, ok := decoded.(metadata.StringParameter)
	if !ok || value.Default == nil || *value.Default != "json" || value.Kind() != metadata.ParameterKindString {
		t.Fatalf("decodeParameter() = %#v", decoded)
	}
}
