package domain

import (
	"testing"

	"github.com/google/uuid"
)

func validRetrievalInput() RetrievalInput {
	return RetrievalInput{
		ResourceID: uuid.MustParse("2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192"),
		VersionID:  uuid.MustParse("67f6f345-5a27-467b-967b-0aa9e7f85bb1"),
		Method:     "GET",
		URL:        "https://example.org/resource",
	}
}

func TestNewPreparedRetrievalJSONBody(t *testing.T) {
	input := validRetrievalInput()
	input.Method = "post"
	input.URL = "https://example.org/resource/abc"
	input.Body = []byte(`{"k":"v"}`)
	prepared, err := NewPreparedRetrieval(input)
	if err != nil {
		t.Fatalf("NewPreparedRetrieval() error = %v", err)
	}
	if prepared.Method != "POST" {
		t.Fatalf("normalized method = %q", prepared.Method)
	}
	if prepared.URL != input.URL {
		t.Fatalf("prepared URL = %q", prepared.URL)
	}
	if string(prepared.Body) != `{"k":"v"}` {
		t.Fatalf("body bytes = %q", prepared.Body)
	}
}

func TestNewPreparedRetrievalRejectsDeleteBody(t *testing.T) {
	input := validRetrievalInput()
	input.Method = "DELETE"
	input.Body = []byte("x")
	if _, err := NewPreparedRetrieval(input); err == nil {
		t.Fatal("NewPreparedRetrieval() accepted a DELETE body")
	}
}

func TestNewPreparedRetrievalPreservesJSONBody(t *testing.T) {
	input := validRetrievalInput()
	input.Method = "POST"
	input.Body = []byte(`{"b":2,"a":1}`)
	prepared, err := NewPreparedRetrieval(input)
	if err != nil {
		t.Fatalf("NewPreparedRetrieval() error = %v", err)
	}
	if got, want := string(prepared.Body), `{"b":2,"a":1}`; got != want {
		t.Fatalf("prepared JSON = %q, want %q", got, want)
	}
}
