package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
)

var placeholderRegex = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

type RetrieveMode string

const (
	RetrieveModeDry     RetrieveMode = "DRY"
	RetrieveModeExecute RetrieveMode = "EXECUTE"
)

type RequestBody struct {
	ContentType string          `json:"content_type"`
	Raw         string          `json:"raw,omitempty"`
	JSON        json.RawMessage `json:"json,omitempty"`
}

type RetrieveRequest struct {
	Mode             RetrieveMode      `json:"mode"`
	ResourceID       string            `json:"resource_id"`
	VersionID        string            `json:"version_id"`
	Method           string            `json:"method"`
	URLTemplate      string            `json:"url_template"`
	PathParams       map[string]string `json:"path_params"`
	QueryParams      map[string]string `json:"query_params,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	BearerToken      string            `json:"bearer_token,omitempty"`
	Body             *RequestBody      `json:"body,omitempty"`
	OutputParameters []OutputParameter `json:"output_parameters"`
}

type OutputParameter struct {
	Name    string
	Kind    string
	Pattern *string
}

type ValidatedRequest struct {
	RetrieveRequest
	GeneratedURL string
	BodyBytes    []byte
}

func (r RetrieveRequest) Validate() (ValidatedRequest, error) {
	if r.Mode == "" {
		r.Mode = RetrieveModeExecute
	}
	if !slices.Contains([]RetrieveMode{RetrieveModeDry, RetrieveModeExecute}, r.Mode) {
		return ValidatedRequest{}, errors.New("mode must be DRY or EXECUTE")
	}

	if _, err := uuid.Parse(r.ResourceID); err != nil {
		return ValidatedRequest{}, errors.New("resource_id must be a UUID")
	}
	if _, err := uuid.Parse(r.VersionID); err != nil {
		return ValidatedRequest{}, errors.New("version_id must be a UUID")
	}

	parameters, err := validateOutputParameters(r.OutputParameters)
	if err != nil {
		return ValidatedRequest{}, err
	}
	r.OutputParameters = parameters

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

	if hasHeader(r.Headers, "Authorization") && strings.TrimSpace(r.BearerToken) != "" {
		return ValidatedRequest{}, errors.New("headers.Authorization cannot be set when bearer_token is provided")
	}
	if hasHeader(r.Headers, "Host") {
		return ValidatedRequest{}, errors.New("headers.Host is not allowed")
	}
	if hasHeader(r.Headers, "Content-Length") {
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

func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if textproto.CanonicalMIMEHeaderKey(key) == name {
			return true
		}
	}
	return false
}

func (r ValidatedRequest) CurlCommand() string {
	arguments := []string{"curl", "--request", r.Method, "--url", shellQuote(r.GeneratedURL)}
	headerNames := make([]string, 0, len(r.Headers))
	for name := range r.Headers {
		headerNames = append(headerNames, name)
	}
	sort.Strings(headerNames)
	for _, name := range headerNames {
		if r.Body != nil && textproto.CanonicalMIMEHeaderKey(name) == "Content-Type" {
			continue
		}
		if textproto.CanonicalMIMEHeaderKey(name) == "Authorization" {
			arguments = append(arguments, "--header", `"Authorization: ${AUTHORIZATION}"`)
			continue
		}
		arguments = append(arguments, "--header", shellQuote(name+": "+r.Headers[name]))
	}
	if r.Body != nil {
		arguments = append(arguments, "--header", shellQuote("Content-Type: "+r.Body.ContentType))
	}
	if strings.TrimSpace(r.BearerToken) != "" {
		arguments = append(arguments, "--oauth2-bearer", `"${BEARER_TOKEN}"`)
	}
	if len(r.BodyBytes) > 0 {
		arguments = append(arguments, "--data-binary", shellQuote(string(r.BodyBytes)))
	}
	return strings.Join(arguments, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func validateOutputParameters(parameters []OutputParameter) ([]OutputParameter, error) {
	validated := make([]OutputParameter, 0, len(parameters))
	names := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		parameter.Name = strings.TrimSpace(parameter.Name)
		parameter.Kind = strings.ToUpper(strings.TrimSpace(parameter.Kind))
		if parameter.Name == "" {
			return nil, errors.New("output parameter name is required")
		}
		if _, exists := names[parameter.Name]; exists {
			return nil, fmt.Errorf("output parameter %q is duplicated", parameter.Name)
		}
		if !slices.Contains([]string{"STRING", "BOOLEAN", "INTEGER", "FLOAT", "DATETIME", "JSON", "BINARY"}, parameter.Kind) {
			return nil, fmt.Errorf("output parameter %q has an invalid kind", parameter.Name)
		}
		if parameter.Pattern != nil {
			pattern := strings.TrimSpace(*parameter.Pattern)
			if pattern == "" {
				return nil, fmt.Errorf("output parameter %q pattern is empty", parameter.Name)
			}
			if _, err := regexp.Compile(pattern); err != nil {
				return nil, fmt.Errorf("output parameter %q pattern is invalid: %w", parameter.Name, err)
			}
			parameter.Pattern = &pattern
		}
		names[parameter.Name] = struct{}{}
		validated = append(validated, parameter)
	}
	return validated, nil
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
