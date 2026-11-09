package domain

import (
	"io"

	"github.com/google/uuid"
)

type ConversionRuntime string

const (
	ConversionRuntimePython ConversionRuntime = "python"
	ConversionRuntimeJava   ConversionRuntime = "java"
	ConversionRuntimeBinary ConversionRuntime = "binary"
)

type PluginInput struct {
	Name        string
	Description string
	Repository  string
	Branch      string
	Runtime     ConversionRuntime
	Arguments   []string
}

type Plugin struct {
	PluginInput
	ID              uuid.UUID
	Enabled         bool
	Installed       bool
	InstalledCommit *string
	CreatedBy       string
	UpdatedBy       string
}

type DistributionPluginRelation struct {
	PluginID       uuid.UUID
	Name           string
	Description    string
	Repository     string
	Branch         string
	Runtime        ConversionRuntime
	Arguments      []string
	Enabled        bool
	Installed      bool
	RelationID     uuid.UUID
	DistributionID uuid.UUID
	InputFormat    string
	OutputFormat   string
}

type AccessRequest struct {
	DistributionID uuid.UUID
	OutputFormat   *string
	ConversionID   *uuid.UUID
	Input          WebServiceAccessInput
}

type WebServiceAccessInput interface {
	webServiceAccessInput()
}

type HTTPAccessInput struct {
	Parameters []AccessParameter
}

func (HTTPAccessInput) webServiceAccessInput() {}

type DynamicAccessInput struct {
	Longitude  float64
	Latitude   float64
	Parameters []AccessParameter
}

func (DynamicAccessInput) webServiceAccessInput() {}

type WFSAccessInput struct {
	FeatureType *string
	ResultLimit *int64
	BBox        *WFSBoundingBox
	Parameters  []AccessParameter
}

func (WFSAccessInput) webServiceAccessInput() {}

type WFSBoundingBox struct {
	West  float64
	South float64
	East  float64
	North float64
}

type AccessParameter struct {
	ID    uuid.UUID
	Value AccessValue
}

// AccessValue preserves scalar types and scalar/array shape at the transport boundary.
type AccessValue interface {
	accessValue()
}

// AccessScalar restricts array elements and template children to scalar values.
type AccessScalar interface {
	AccessValue
	accessScalar()
}

type (
	AccessString  string
	AccessBoolean bool
	AccessInteger int64
	AccessFloat   float64
	AccessArray   []AccessScalar
)

func (AccessString) accessValue()  {}
func (AccessBoolean) accessValue() {}
func (AccessInteger) accessValue() {}
func (AccessFloat) accessValue()   {}
func (AccessArray) accessValue()   {}

func (AccessString) accessScalar()  {}
func (AccessBoolean) accessScalar() {}
func (AccessInteger) accessScalar() {}
func (AccessFloat) accessScalar()   {}

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

type ConversionRelation struct {
	ID             uuid.UUID
	DistributionID uuid.UUID
	InputFormat    string
	OutputFormat   string
}

type AvailableConversion struct {
	ConversionRelation
	PluginID          uuid.UUID
	PluginName        string
	PluginDescription string
}
