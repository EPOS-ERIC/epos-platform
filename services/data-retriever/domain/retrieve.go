package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

type RequestBody struct {
	ContentType string          `json:"content_type"`
	Raw         string          `json:"raw,omitempty"`
	JSON        json.RawMessage `json:"json,omitempty"`
}

type RetrieveRequest struct {
	SourceID    string            `json:"source_id"`
	Method      string            `json:"method"`
	URLTemplate string            `json:"url_template"`
	PathParams  map[string]string `json:"path_params"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	BearerToken string            `json:"bearer_token,omitempty"`
	Body        *RequestBody      `json:"body,omitempty"`
}

type ValidatedRequest struct {
	RetrieveRequest
	GeneratedURL string
	BodyBytes    []byte
}

func (r RetrieveRequest) Validate() (ValidatedRequest, error) {
	if strings.TrimSpace(r.SourceID) == "" {
		return ValidatedRequest{}, errors.New("source_id is required")
	}

	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if !slices.Contains([]string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}, method) {
		return ValidatedRequest{}, errors.New("method must be one of GET, POST, PUT, DELETE")
	}
	r.Method = method

	if strings.TrimSpace(r.URLTemplate) == "" {
		return ValidatedRequest{}, errors.New("url_template is required")
	}
	parsedTemplate, err := url.Parse(r.URLTemplate)
	if err != nil {
		return ValidatedRequest{}, fmt.Errorf("invalid url_template: %w", err)
	}
	if !parsedTemplate.IsAbs() || parsedTemplate.Host == "" {
		return ValidatedRequest{}, errors.New("url_template must be an absolute URL")
	}
	if parsedTemplate.Scheme != "http" && parsedTemplate.Scheme != "https" {
		return ValidatedRequest{}, errors.New("url_template scheme must be http or https")
	}

	if _, exists := r.Headers["Authorization"]; exists && strings.TrimSpace(r.BearerToken) != "" {
		return ValidatedRequest{}, errors.New("headers.Authorization cannot be set when bearer_token is provided")
	}
	if _, exists := r.Headers["Host"]; exists {
		return ValidatedRequest{}, errors.New("headers.Host is not allowed")
	}
	if _, exists := r.Headers["Content-Length"]; exists {
		return ValidatedRequest{}, errors.New("headers.Content-Length is not allowed")
	}

	generatedURL, err := buildURL(r.URLTemplate, r.PathParams, r.QueryParams)
	if err != nil {
		return ValidatedRequest{}, err
	}

	bodyBytes, err := validateBody(method, r.Body)
	if err != nil {
		return ValidatedRequest{}, err
	}

	return ValidatedRequest{RetrieveRequest: r, GeneratedURL: generatedURL, BodyBytes: bodyBytes}, nil
}

func buildURL(template string, pathParams map[string]string, queryParams map[string]string) (string, error) {
	matches := placeholderRegex.FindAllStringSubmatch(template, -1)
	placeholders := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		placeholders[m[1]] = struct{}{}
	}

	for key := range placeholders {
		if _, ok := pathParams[key]; !ok {
			return "", fmt.Errorf("missing path_params key %q", key)
		}
	}
	for key := range pathParams {
		if _, ok := placeholders[key]; !ok {
			return "", fmt.Errorf("unexpected path_params key %q", key)
		}
	}

	generated := template
	for key, value := range pathParams {
		generated = strings.ReplaceAll(generated, "{"+key+"}", url.PathEscape(value))
	}

	u, err := url.Parse(generated)
	if err != nil {
		return "", fmt.Errorf("generated URL is invalid: %w", err)
	}

	query := u.Query()
	for key, value := range queryParams {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func validateBody(method string, body *RequestBody) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	if method == http.MethodGet || method == http.MethodDelete {
		return nil, fmt.Errorf("body is not allowed for %s", method)
	}

	contentType := strings.ToLower(strings.TrimSpace(body.ContentType))
	if contentType == "" {
		return nil, errors.New("body.content_type is required when body is provided")
	}

	hasRaw := strings.TrimSpace(body.Raw) != ""
	hasJSON := len(body.JSON) > 0
	if hasRaw == hasJSON {
		return nil, errors.New("exactly one of body.raw or body.json must be provided")
	}

	if hasJSON {
		if !isJSONContentType(contentType) {
			return nil, errors.New("body.content_type must be application/json or application/*+json when body.json is provided")
		}
		if !json.Valid(body.JSON) {
			return nil, errors.New("body.json must be valid JSON")
		}

		var decoded any
		if err := json.Unmarshal(body.JSON, &decoded); err != nil {
			return nil, errors.New("body.json must be valid JSON")
		}

		canonicalJSON, err := json.Marshal(decoded)
		if err != nil {
			return nil, errors.New("body.json cannot be serialized")
		}

		return canonicalJSON, nil
	}

	return []byte(body.Raw), nil
}

func isJSONContentType(contentType string) bool {
	base := strings.Split(contentType, ";")[0]
	base = strings.TrimSpace(base)
	if base == "application/json" {
		return true
	}
	return strings.HasPrefix(base, "application/") && strings.HasSuffix(base, "+json")
}
