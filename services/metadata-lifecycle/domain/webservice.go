package domain

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

type HTTPMethod string

const (
	HTTPMethodGet    HTTPMethod = "GET"
	HTTPMethodPost   HTTPMethod = "POST"
	HTTPMethodPut    HTTPMethod = "PUT"
	HTTPMethodDelete HTTPMethod = "DELETE"
)

func (m HTTPMethod) Valid() bool {
	switch m {
	case HTTPMethodGet, HTTPMethodPost, HTTPMethodPut, HTTPMethodDelete:
		return true
	default:
		return false
	}
}

type WebServiceType string

const (
	WebServiceTypeWMS     WebServiceType = "WMS"
	WebServiceTypeHTTP    WebServiceType = "HTTP"
	WebServiceTypeWMTS    WebServiceType = "WMTS"
	WebServiceTypeWFS     WebServiceType = "WFS"
	WebServiceTypeDynamic WebServiceType = "DYNAMIC"
)

func (t WebServiceType) Valid() bool {
	switch t {
	case WebServiceTypeWMS, WebServiceTypeHTTP, WebServiceTypeWMTS, WebServiceTypeWFS, WebServiceTypeDynamic:
		return true
	default:
		return false
	}
}

type WebServiceVisibleOn string

const (
	WebServiceVisibleOnGraph WebServiceVisibleOn = "GRAPH"
	WebServiceVisibleOnTable WebServiceVisibleOn = "TABLE"
	WebServiceVisibleOnMap   WebServiceVisibleOn = "MAP"
)

func (v WebServiceVisibleOn) Valid() bool {
	switch v {
	case WebServiceVisibleOnGraph, WebServiceVisibleOnTable, WebServiceVisibleOnMap:
		return true
	default:
		return false
	}
}

func isTypedNilWebService(value any) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	kind := v.Kind()
	if kind == reflect.Chan || kind == reflect.Func || kind == reflect.Interface ||
		kind == reflect.Map || kind == reflect.Pointer || kind == reflect.Slice {
		return v.IsNil()
	}
	return false
}

type WebService interface {
	ServiceType() WebServiceType
	Common() *WebServiceCommon
	Normalize() error
}

type ExecutableWebService interface {
	WebService
	NativeFormats() []NativeFormat
	PrepareAccess(DistributionAccessInput) (PreparedHTTPRequest, error)
}

type WebServiceCommon struct {
	ID               uuid.UUID
	ResourceID       uuid.UUID
	VisibleOn        []WebServiceVisibleOn
	Provider         *OrganizationMetadata
	DocumentationURI *string
	ContactPoints    []ContactPoint
}

func (w *WebServiceCommon) Common() *WebServiceCommon {
	return w
}

func (w *WebServiceCommon) Normalize() error {
	if w.ID == uuid.Nil {
		return fmt.Errorf("web service ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}

	visibleOn, err := normalizeWebServiceVisibleOn(w.VisibleOn)
	if err != nil {
		return fmt.Errorf("visible on: %w", err)
	}
	w.VisibleOn = visibleOn

	if w.DocumentationURI != nil {
		uri := strings.TrimSpace(*w.DocumentationURI)
		if uri == "" {
			return fmt.Errorf("documentation URI must not be blank: %w", apperror.Default(apperror.CodeValidationFailed))
		}

		parsed, err := url.Parse(uri)
		if err != nil || !parsed.IsAbs() || strings.ContainsAny(uri, " \t\r\n") {
			return fmt.Errorf("documentation URI must be an absolute URI: %w", apperror.Default(apperror.CodeValidationFailed))
		}

		w.DocumentationURI = &uri
	}

	if w.Provider != nil {
		if err := w.Provider.Normalize(); err != nil {
			return fmt.Errorf("provider: %w", err)
		}
	}

	if err := validateContactPoints(w.ContactPoints, "web service contact point"); err != nil {
		return fmt.Errorf("contact points: %w", err)
	}

	return nil
}

// ########### INPUT ###########

type WebServiceInput interface {
	ServiceType() WebServiceType
	CommonInput() *WebServiceCommonInput
	Normalize() error
}

// WebServiceCommonInput contains caller-controlled web-service fields.
type WebServiceCommonInput struct {
	ResourceID       *uuid.UUID
	VisibleOn        []WebServiceVisibleOn
	ProviderID       *uuid.UUID
	DocumentationURI *string
	ContactPointIDs  []uuid.UUID
}

func (w *WebServiceCommonInput) CommonInput() *WebServiceCommonInput {
	return w
}

func (w *WebServiceCommonInput) Normalize() error {
	visibleOn, err := normalizeWebServiceVisibleOn(w.VisibleOn)
	if err != nil {
		return invalidField("visible_on", "Audience selection must contain supported values.", err)
	}
	w.VisibleOn = visibleOn

	if w.ProviderID != nil && *w.ProviderID == uuid.Nil {
		return invalidField("provider", "Provider must be a valid UUID.", nil)
	}
	if err := validateUUIDListField(w.ContactPointIDs, "contact_points", "Contact point"); err != nil {
		return err
	}
	if w.DocumentationURI != nil {
		uri := strings.TrimSpace(*w.DocumentationURI)
		if uri == "" {
			return invalidField("documentation_uri", "Documentation URI must not be blank.", nil)
		}

		parsed, err := url.Parse(uri)
		if err != nil || !parsed.IsAbs() || strings.ContainsAny(uri, " \t\r\n") {
			return invalidField("documentation_uri", "Documentation URI must be an absolute URI.", err)
		}

		w.DocumentationURI = &uri
	}
	return nil
}
