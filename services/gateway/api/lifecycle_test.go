package api

import (
	"encoding/json"
	"testing"
)

func TestDecodeParameterDraftAllowsMetadata(t *testing.T) {
	data := json.RawMessage(`{
		"kind": "FLOAT",
		"metadata": {
			"kind": "FLOAT",
			"variable": "latitude",
			"property": "latitude",
			"label": "Latitude",
			"required": false,
			"multiple_values": false,
			"read_only": false
		},
		"minimum": 33,
		"maximum": 74,
		"default": null,
		"allowed_values": []
	}`)

	parameter, err := decodeParameterDraft(data)
	if err != nil {
		t.Fatalf("decodeParameterDraft() error = %v", err)
	}
	if parameter == nil {
		t.Fatal("decodeParameterDraft() returned nil parameter")
	}
}
