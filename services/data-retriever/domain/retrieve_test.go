package domain

import (
	"strings"
	"testing"
)

func TestValidateSuccessJSONBody(t *testing.T) {
	req := RetrieveRequest{
		ResourceID:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionID:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
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
		ResourceID:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionID:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
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
		ResourceID:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionID:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
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
		ResourceID:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionID:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
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

func TestValidateRejectsInvalidOutputParameterPattern(t *testing.T) {
	pattern := "["
	req := RetrieveRequest{
		ResourceID:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionID:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
		Method:      "GET",
		URLTemplate: "https://example.org/resource",
		OutputParameters: []OutputParameter{{
			Name:    "value",
			Kind:    "STRING",
			Pattern: &pattern,
		}},
	}

	if _, err := req.Validate(); err == nil {
		t.Fatal("expected invalid output parameter pattern")
	}
}

func TestValidatedRequestCurlCommandIsDeterministicAndRedactsCredentials(t *testing.T) {
	req := RetrieveRequest{
		Mode:        RetrieveModeDry,
		ResourceID:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionID:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
		Method:      "POST",
		URLTemplate: "https://example.org/resource/{id}",
		PathParams:  map[string]string{"id": "a b"},
		QueryParams: map[string]string{"z": "last", "a": "first"},
		Headers: map[string]string{
			"X-Zeta":  "z",
			"X-Alpha": "it's safe",
		},
		BearerToken: "secret",
		Body: &RequestBody{
			ContentType: "application/json",
			JSON:        []byte(`{"b":2,"a":1}`),
		},
	}

	validated, err := req.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	command := validated.CurlCommand()
	want := "curl --request POST --url 'https://example.org/resource/a%20b?a=first&z=last' --header 'X-Alpha: it'\\''s safe' --header 'X-Zeta: z' --header 'Content-Type: application/json' --oauth2-bearer \"${BEARER_TOKEN}\" --data-binary '{\"a\":1,\"b\":2}'"
	if command != want {
		t.Fatalf("CurlCommand() = %q, want %q", command, want)
	}
	if strings.Contains(command, "secret") {
		t.Fatal("curl command exposes bearer token")
	}
}
