package domain

import (
	"strconv"

	"github.com/google/uuid"
)

// DistributionAccessInput contains caller-supplied values, not upstream metadata.
type DistributionAccessInput struct {
	Input        WebServiceAccessInput
	OutputFormat *string
}

type WebServiceAccessInput interface {
	webServiceAccessInput()
}

type HTTPAccessInput struct {
	Parameters []DistributionAccessParameter
}

func (HTTPAccessInput) webServiceAccessInput() {}

type WFSAccessInput struct {
	FeatureType *string
	ResultLimit *int64
	BBox        *WFSBoundingBox
	Parameters  []DistributionAccessParameter
}

func (WFSAccessInput) webServiceAccessInput() {}

type WFSBoundingBox struct {
	West  float64
	South float64
	East  float64
	North float64
}

type DistributionAccessParameter struct {
	ID    uuid.UUID
	Value AccessValue
}

// AccessValue preserves scalar types and scalar/array shape at the transport boundary.
type AccessValue interface {
	// TypeName returns a readable type name without exposing the value.
	TypeName() string
	accessValue()
}

// AccessScalar restricts array elements and template children to scalar values.
type AccessScalar interface {
	AccessValue
	// Text returns unescaped outbound text; validation must happen before serialization.
	Text() string
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

func (AccessString) TypeName() string  { return "string" }
func (AccessBoolean) TypeName() string { return "boolean" }
func (AccessInteger) TypeName() string { return "integer" }
func (AccessFloat) TypeName() string   { return "float" }
func (AccessArray) TypeName() string   { return "array" }

func (AccessString) accessScalar()  {}
func (AccessBoolean) accessScalar() {}
func (AccessInteger) accessScalar() {}
func (AccessFloat) accessScalar()   {}

// Text returns the string unchanged, without URL or JSON escaping.
func (value AccessString) Text() string { return string(value) }

// Text returns true or false.
func (value AccessBoolean) Text() string { return strconv.FormatBool(bool(value)) }

// Text returns the integer in signed decimal notation.
func (value AccessInteger) Text() string { return strconv.FormatInt(int64(value), 10) }

// Text returns the float's shortest round-trippable representation.
func (value AccessFloat) Text() string { return strconv.FormatFloat(float64(value), 'g', -1, 64) }

// PreparedHTTPRequest is a fully rendered upstream request; nil Body means absent.
type PreparedHTTPRequest struct {
	Method  HTTPMethod
	URL     string
	Headers map[string]string
	Body    []byte
}

type DistributionAccess struct {
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	PreparedHTTPRequest
}
