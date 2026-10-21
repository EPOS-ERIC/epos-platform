package domain

import (
	"errors"
	"net/textproto"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
	"golang.org/x/net/http/httpguts"
)

const (
	methodGet    = "GET"
	methodPost   = "POST"
	methodPut    = "PUT"
	methodDelete = "DELETE"
)

type RetrievalInput struct {
	ResourceID      uuid.UUID
	VersionID       uuid.UUID
	Method          string
	URL             string
	Headers         map[string]string
	Body            []byte
	SkipPersistence bool
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
}

// NewPreparedRetrieval validates and normalizes input before upstream execution.
func NewPreparedRetrieval(input RetrievalInput) (PreparedRetrieval, error) {
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

	if strings.TrimSpace(input.URL) == "" {
		return PreparedRetrieval{}, apperror.Field("url", "Value is invalid.", errors.New("url is required"))
	}
	parsedURL, err := url.ParseRequestURI(input.URL)
	if err != nil {
		return PreparedRetrieval{}, apperror.Field("url", "Value is invalid.", nil)
	}
	if !parsedURL.IsAbs() || parsedURL.Host == "" {
		return PreparedRetrieval{}, apperror.Field("url", "Value is invalid.", errors.New("url must be an absolute URL"))
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return PreparedRetrieval{}, apperror.Field("url", "Value is invalid.", errors.New("url scheme must be http or https"))
	}

	if hasHeader(input.Headers, "Host") {
		return PreparedRetrieval{}, apperror.Field("headers.Host", "Value is invalid.", errors.New("headers.Host is not allowed"))
	}
	if hasHeader(input.Headers, "Content-Length") {
		return PreparedRetrieval{}, apperror.Field("headers.Content-Length", "Value is invalid.", errors.New("headers.Content-Length is not allowed"))
	}
	for name, value := range input.Headers {
		if !httpguts.ValidHeaderFieldValue(value) {
			return PreparedRetrieval{}, apperror.Field("headers", "Header value is invalid.", nil)
		}
		if !httpguts.ValidHeaderFieldName(name) {
			return PreparedRetrieval{}, apperror.Field("headers", "Header name is invalid.", nil)
		}
	}

	if input.Body != nil && (method == methodGet || method == methodDelete) {
		return PreparedRetrieval{}, apperror.Field("body", "Request bodies are not supported for GET or DELETE.", nil)
	}
	return PreparedRetrieval{RetrievalInput: input}, nil
}

func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if textproto.CanonicalMIMEHeaderKey(key) == name {
			return true
		}
	}
	return false
}
