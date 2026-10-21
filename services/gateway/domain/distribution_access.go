package domain

import (
	"io"

	"github.com/google/uuid"
)

type AccessRequest struct {
	DistributionID uuid.UUID
	OutputFormat   *string
	ConversionID   *uuid.UUID
	Parameters     []AccessParameter
}

type AccessParameter struct {
	ID    uuid.UUID
	Value AccessValue
}

type AccessValue struct {
	Scalar *AccessScalar
	Array  []AccessScalar
}

type AccessScalarKind string

const (
	AccessScalarString  AccessScalarKind = "STRING"
	AccessScalarBoolean AccessScalarKind = "BOOLEAN"
	AccessScalarInteger AccessScalarKind = "INTEGER"
	AccessScalarFloat   AccessScalarKind = "FLOAT"
)

type AccessScalar struct {
	Kind    AccessScalarKind
	String  string
	Boolean bool
	Integer int64
	Float   float64
}

// DownloadTarget is a metadata-authorized file request for Data Retriever.
type DownloadTarget struct {
	// TODO: maybe we sholud return the distribution id too?
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	URL        string
}

type HTTPMethod string

const (
	HTTPMethodGET    HTTPMethod = "GET"
	HTTPMethodPOST   HTTPMethod = "POST"
	HTTPMethodPUT    HTTPMethod = "PUT"
	HTTPMethodDELETE HTTPMethod = "DELETE"
)

type PreparedRequest struct {
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	Method     HTTPMethod
	URL        string
	Headers    map[string]string
	Body       []byte
}

type RetrievalResponse struct {
	StatusCode int
	Body       io.ReadCloser
}
