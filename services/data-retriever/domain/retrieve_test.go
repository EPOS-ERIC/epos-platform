package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func validRetrievalInput() RetrievalInput {
	return RetrievalInput{
		ResourceID:  uuid.MustParse("2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192"),
		VersionID:   uuid.MustParse("67f6f345-5a27-467b-967b-0aa9e7f85bb1"),
		Method:      "GET",
		URLTemplate: "https://example.org/resource",
	}
}

func TestNewPreparedRetrievalJSONBody(t *testing.T) {
	input := validRetrievalInput()
	input.Method = "post"
	input.URLTemplate = "https://example.org/resource/{id}"
	input.PathParams = map[string]string{"id": "abc"}
	input.Body = &RequestBody{
		ContentType: "application/json; charset=utf-8",
		JSON:        []byte(`{"k":"v"}`),
	}
	prepared, err := NewPreparedRetrieval(input)
	if err != nil {
		t.Fatalf("NewPreparedRetrieval() error = %v", err)
	}
	if prepared.Mode != RetrieveModeExecute || prepared.Method != "POST" {
		t.Fatalf("normalized mode/method = %q/%q", prepared.Mode, prepared.Method)
	}
	if prepared.GeneratedURL != "https://example.org/resource/abc" {
		t.Fatalf("generated URL = %q", prepared.GeneratedURL)
	}
	if string(prepared.BodyBytes) != `{"k":"v"}` {
		t.Fatalf("body bytes = %q", prepared.BodyBytes)
	}
}

func TestNewPreparedRetrievalRejectsDeleteBody(t *testing.T) {
	input := validRetrievalInput()
	input.Method = "DELETE"
	input.Body = &RequestBody{ContentType: "text/plain", Raw: "x"}
	if _, err := NewPreparedRetrieval(input); err == nil {
		t.Fatal("NewPreparedRetrieval() accepted a DELETE body")
	}
}

func TestNewPreparedRetrievalRejectsPathParameterMismatch(t *testing.T) {
	input := validRetrievalInput()
	input.URLTemplate += "/{id}"
	input.PathParams = map[string]string{"other": "abc"}
	if _, err := NewPreparedRetrieval(input); err == nil {
		t.Fatal("NewPreparedRetrieval() accepted mismatched path parameters")
	}
}

func TestNewPreparedRetrievalCanonicalizesJSONBody(t *testing.T) {
	input := validRetrievalInput()
	input.Method = "POST"
	input.Body = &RequestBody{ContentType: "application/json", JSON: []byte(`{"b":2,"a":1}`)}
	prepared, err := NewPreparedRetrieval(input)
	if err != nil {
		t.Fatalf("NewPreparedRetrieval() error = %v", err)
	}
	if got, want := string(prepared.BodyBytes), `{"a":1,"b":2}`; got != want {
		t.Fatalf("canonical JSON = %q, want %q", got, want)
	}
}

func TestPreparedRetrievalCurlRedactsCredentialsAndBody(t *testing.T) {
	input := validRetrievalInput()
	input.Mode = RetrieveModeDry
	input.Method = "POST"
	input.URLTemplate = "https://example.org/resource/{id}"
	input.PathParams = map[string]string{"id": "a b"}
	input.QueryParams = map[string]string{"z": "last", "a": "first"}
	input.Headers = map[string]string{
		"Authorization": "header-secret",
		"Cookie":        "cookie-secret",
		"Set-Cookie":    "set-cookie-secret",
		"X-Zeta":        "z",
		"X-Alpha":       "it's safe",
	}
	input.Body = &RequestBody{ContentType: "application/json", JSON: []byte(`{"password":"body-secret"}`)}
	prepared, err := NewPreparedRetrieval(input)
	if err != nil {
		t.Fatalf("NewPreparedRetrieval() error = %v", err)
	}
	command := prepared.CurlCommand()
	for _, secret := range []string{"header-secret", "cookie-secret", "set-cookie-secret", "body-secret"} {
		if strings.Contains(command, secret) {
			t.Errorf("CurlCommand() exposed %q", secret)
		}
	}
	for _, fragment := range []string{
		`"Authorization: ${AUTHORIZATION}"`,
		"Cookie: [redacted]",
		"Set-Cookie: [redacted]",
		`--data-binary "${REQUEST_BODY}"`,
		"?a=first&z=last",
	} {
		if !strings.Contains(command, fragment) {
			t.Errorf("CurlCommand() missing %q: %s", fragment, command)
		}
	}
}

func TestPreparedRetrievalCurlRedactsBearerToken(t *testing.T) {
	input := validRetrievalInput()
	input.BearerToken = "bearer-secret"
	prepared, err := NewPreparedRetrieval(input)
	if err != nil {
		t.Fatalf("NewPreparedRetrieval() error = %v", err)
	}
	command := prepared.CurlCommand()
	if strings.Contains(command, "bearer-secret") || !strings.Contains(command, `--oauth2-bearer "${BEARER_TOKEN}"`) {
		t.Fatalf("CurlCommand() did not safely represent bearer token: %s", command)
	}
}
