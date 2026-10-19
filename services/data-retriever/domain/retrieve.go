package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/textproto"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

var placeholderRegex = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

const (
	methodGet    = "GET"
	methodPost   = "POST"
	methodPut    = "PUT"
	methodDelete = "DELETE"
)

type RetrieveMode string

const (
	RetrieveModeDry     RetrieveMode = "DRY"
	RetrieveModeExecute RetrieveMode = "EXECUTE"
)

type RequestBody struct {
	ContentType string
	Raw         string
	JSON        json.RawMessage
}

type RetrievalInput struct {
	Mode        RetrieveMode
	ResourceID  uuid.UUID
	VersionID   uuid.UUID
	Method      string
	URLTemplate string
	PathParams  map[string]string
	QueryParams map[string]string
	Headers     map[string]string
	BearerToken string
	Body        *RequestBody
}

type OutputParameter struct {
	Name string
	Kind string
}

type ResponseHead struct {
	StatusCode    int
	ContentType   string
	ContentLength int64
	IsBinary      bool
}

type RetrievalRecord struct {
	ID               int64
	ResourceID       uuid.UUID
	VersionID        uuid.UUID
	Method           string
	URLTemplate      string
	GeneratedURL     string
	HTTPStatus       int
	ContentType      string
	IsBinary         bool
	PayloadHash      string
	PayloadSize      int64
	UsedAuth         bool
	FetchedAt        time.Time
	CreatedAt        time.Time
	OutputParameters []OutputParameter
}

type PreparedRetrieval struct {
	RetrievalInput
	GeneratedURL string
	BodyBytes    []byte
}

// NewPreparedRetrieval validates and normalizes input before upstream execution.
func NewPreparedRetrieval(input RetrievalInput) (PreparedRetrieval, error) {
	if input.Mode == "" {
		input.Mode = RetrieveModeExecute
	}
	if !slices.Contains([]RetrieveMode{RetrieveModeDry, RetrieveModeExecute}, input.Mode) {
		return PreparedRetrieval{}, apperror.Field("mode", "Value is invalid.", errors.New("mode must be DRY or EXECUTE"))
	}
	if input.ResourceID == uuid.Nil {
		return PreparedRetrieval{}, apperror.Field("resource_id", "Value is invalid.", errors.New("resource_id must be a non-zero UUID"))
	}
	if input.VersionID == uuid.Nil {
		return PreparedRetrieval{}, apperror.Field("version_id", "Value is invalid.", errors.New("version_id must be a non-zero UUID"))
	}

	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if !slices.Contains([]string{methodGet, methodPost, methodPut, methodDelete}, method) {
		return PreparedRetrieval{}, apperror.Field("method", "Value is invalid.", errors.New("method must be one of GET, POST, PUT, DELETE"))
	}
	input.Method = method

	if strings.TrimSpace(input.URLTemplate) == "" {
		return PreparedRetrieval{}, apperror.Field("url_template", "Value is invalid.", errors.New("url_template is required"))
	}
	parsedTemplate, err := url.Parse(input.URLTemplate)
	if err != nil {
		return PreparedRetrieval{}, apperror.Field("url_template", "Value is invalid.", err)
	}
	if !parsedTemplate.IsAbs() || parsedTemplate.Host == "" {
		return PreparedRetrieval{}, apperror.Field("url_template", "Value is invalid.", errors.New("url_template must be an absolute URL"))
	}
	if parsedTemplate.Scheme != "http" && parsedTemplate.Scheme != "https" {
		return PreparedRetrieval{}, apperror.Field("url_template", "Value is invalid.", errors.New("url_template scheme must be http or https"))
	}

	if hasHeader(input.Headers, "Authorization") && strings.TrimSpace(input.BearerToken) != "" {
		return PreparedRetrieval{}, apperror.Field("headers.Authorization", "Value is invalid.", errors.New("headers.Authorization cannot be set when bearer_token is provided"))
	}
	if hasHeader(input.Headers, "Host") {
		return PreparedRetrieval{}, apperror.Field("headers.Host", "Value is invalid.", errors.New("headers.Host is not allowed"))
	}
	if hasHeader(input.Headers, "Content-Length") {
		return PreparedRetrieval{}, apperror.Field("headers.Content-Length", "Value is invalid.", errors.New("headers.Content-Length is not allowed"))
	}

	generatedURL, err := buildURL(input.URLTemplate, input.PathParams, input.QueryParams)
	if err != nil {
		return PreparedRetrieval{}, apperror.Field("path_params", "Value is invalid.", err)
	}
	bodyBytes, err := validateBody(method, input.Body)
	if err != nil {
		return PreparedRetrieval{}, apperror.Field("body", "Value is invalid.", err)
	}
	return PreparedRetrieval{RetrievalInput: input, GeneratedURL: generatedURL, BodyBytes: bodyBytes}, nil
}

func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if textproto.CanonicalMIMEHeaderKey(key) == name {
			return true
		}
	}
	return false
}

func (r PreparedRetrieval) CurlCommand() string {
	arguments := []string{"curl", "--request", r.Method, "--url", shellQuote(r.GeneratedURL)}
	for _, name := range slices.Sorted(maps.Keys(r.Headers)) {
		canonical := textproto.CanonicalMIMEHeaderKey(name)
		if r.Body != nil && canonical == "Content-Type" {
			continue
		}
		switch canonical {
		case "Authorization":
			arguments = append(arguments, "--header", `"Authorization: ${AUTHORIZATION}"`)
		case "Cookie", "Set-Cookie":
			arguments = append(arguments, "--header", shellQuote(name+": [redacted]"))
		default:
			arguments = append(arguments, "--header", shellQuote(name+": "+r.Headers[name]))
		}
	}
	if r.Body != nil {
		arguments = append(arguments, "--header", shellQuote("Content-Type: "+r.Body.ContentType))
	}
	if strings.TrimSpace(r.BearerToken) != "" {
		arguments = append(arguments, "--oauth2-bearer", `"${BEARER_TOKEN}"`)
	}
	if len(r.BodyBytes) > 0 {
		arguments = append(arguments, "--data-binary", `"${REQUEST_BODY}"`)
	}
	return strings.Join(arguments, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func buildURL(template string, pathParams, queryParams map[string]string) (string, error) {
	matches := placeholderRegex.FindAllStringSubmatch(template, -1)
	placeholders := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		placeholders[match[1]] = struct{}{}
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
	parsedURL, err := url.Parse(generated)
	if err != nil {
		return "", fmt.Errorf("generated URL is invalid: %w", err)
	}
	query := parsedURL.Query()
	for key, value := range queryParams {
		query.Set(key, value)
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func validateBody(method string, body *RequestBody) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	if method == methodGet || method == methodDelete {
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
	base, _, _ := strings.Cut(contentType, ";")
	base = strings.TrimSpace(base)
	if base == "application/json" {
		return true
	}
	return strings.HasPrefix(base, "application/") && strings.HasSuffix(base, "+json")
}
