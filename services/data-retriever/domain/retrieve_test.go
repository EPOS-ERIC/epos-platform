package domain

import "testing"

func TestValidateSuccessJSONBody(t *testing.T) {
	req := RetrieveRequest{
		SourceID:    "svc-a",
		Method:      "POST",
		URLTemplate: "https://example.org/resource/{id}",
		PathParams:  map[string]string{"id": "abc"},
		Body: &RequestBody{
			ContentType: "application/json; charset=utf-8",
			JSON:        []byte(`{"k":"v"}`),
		},
	}

	validated, err := req.Validate()
	if err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	if validated.GeneratedURL != "https://example.org/resource/abc" {
		t.Fatalf("unexpected generated URL: %s", validated.GeneratedURL)
	}
}

func TestValidateRejectDeleteBody(t *testing.T) {
	req := RetrieveRequest{
		SourceID:    "svc-a",
		Method:      "DELETE",
		URLTemplate: "https://example.org/resource/{id}",
		PathParams:  map[string]string{"id": "abc"},
		Body: &RequestBody{
			ContentType: "text/plain",
			Raw:         "x",
		},
	}

	if _, err := req.Validate(); err == nil {
		t.Fatal("expected error for DELETE body")
	}
}

func TestValidateRejectPathParamMismatch(t *testing.T) {
	req := RetrieveRequest{
		SourceID:    "svc-a",
		Method:      "GET",
		URLTemplate: "https://example.org/resource/{id}",
		PathParams:  map[string]string{"other": "abc"},
	}

	if _, err := req.Validate(); err == nil {
		t.Fatal("expected error for mismatched path params")
	}
}

func TestValidateCanonicalizesJSONBody(t *testing.T) {
	req := RetrieveRequest{
		SourceID:    "svc-a",
		Method:      "POST",
		URLTemplate: "https://example.org/resource/{id}",
		PathParams:  map[string]string{"id": "abc"},
		Body: &RequestBody{
			ContentType: "application/json",
			JSON:        []byte(`{"b":2, "a":1}`),
		},
	}

	validated, err := req.Validate()
	if err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	if string(validated.BodyBytes) != `{"a":1,"b":2}` {
		t.Fatalf("unexpected canonical JSON: %s", string(validated.BodyBytes))
	}
}
