package retrievals

import (
	"encoding/json"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/google/uuid"
)

func TestUpsertRetrievalParamsPersistsBodyAndRedactsAuth(t *testing.T) {
	bodyText := `{"password":"secret"}`
	request, err := domain.NewPreparedRetrieval(domain.RetrievalInput{
		ResourceID:  uuid.New(),
		VersionID:   uuid.New(),
		URLTemplate: "https://example.test",
		Method:      "POST",
		Headers: map[string]string{
			"aUtHoRiZaTiOn": "Bearer header-secret",
			"Cookie":        "cookie-secret",
		},
		Body: &domain.RequestBody{ContentType: "application/json", JSON: []byte(bodyText)},
	})
	if err != nil {
		t.Fatalf("NewPreparedRetrieval() error = %v", err)
	}
	params, err := upsertRetrievalParams(request, domain.ResponseHead{}, int64(len(bodyText)))
	if err != nil {
		t.Fatalf("upsertRetrievalParams() error = %v", err)
	}
	if params.RequestBodyText == nil || *params.RequestBodyText != bodyText {
		t.Fatalf("request body text = %v, want verbatim body", params.RequestBodyText)
	}
	if !params.UsedAuth {
		t.Fatal("UsedAuth = false for Authorization header")
	}
	var headers map[string]string
	if err := json.Unmarshal(params.HeadersJson, &headers); err != nil {
		t.Fatalf("decode headers: %v", err)
	}
	if headers["aUtHoRiZaTiOn"] != "[redacted]" || headers["Cookie"] != "[redacted]" {
		t.Fatalf("persisted headers = %v", headers)
	}
}
