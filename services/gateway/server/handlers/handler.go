package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/membership"
	"github.com/ogen-go/ogen/ogenerrors"
)

var (
	errAuthUnavailable         = errors.New("authentication unavailable")
	errProvisioningUnavailable = errors.New("user provisioning unavailable")
)

// ogenHandler is the HTTP boundary. Pure proxy operations call the concrete
// Metadata Lifecycle client; identity orchestration remains in identity.Service.
type ogenHandler struct {
	metadataLifecycle *clients.MetadataLifecycle
	dataRetriever     *clients.DataRetriever
	sharing           *clients.Sharing
	identity          *identity.Service
	membership        *membership.Service
}

func NewOgenHandler(i *identity.Service, m *membership.Service, metadataLifecycle *clients.MetadataLifecycle, dataRetriever *clients.DataRetriever, sharing *clients.Sharing) *ogenHandler {
	return &ogenHandler{
		metadataLifecycle: metadataLifecycle,
		dataRetriever:     dataRetriever,
		sharing:           sharing,
		identity:          i,
		membership:        m,
	}
}

var _ gatewayapi.Handler = (*ogenHandler)(nil)

func (h *ogenHandler) NewError(ctx context.Context, err error) *gatewayapi.ProblemStatusCode {
	status, title, detail := problemDetails(err)
	response := gatewayapi.Problem{
		Title:  title,
		Status: status,
	}
	if detail != "" {
		response.Detail = gatewayapi.NewOptString(detail)
	}
	return &gatewayapi.ProblemStatusCode{
		StatusCode: status,
		Response:   response,
	}
}

var errMissingActor = errors.New("bearer token required")

func actorID(ctx context.Context) (string, bool) {
	return authcontext.ActorID(ctx)
}

func isAdmin(ctx context.Context) bool {
	return authcontext.IsAdmin(ctx)
}

func toClientStringPtr(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return &v.Value
}

func problemStatus(err error) int {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return http.StatusRequestEntityTooLarge
	}

	switch {
	case errors.Is(err, errMissingActor),
		errors.Is(err, auth.ErrUnauthorized),
		errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrUpstreamTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, domain.ErrUpstreamUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, errAuthUnavailable), errors.Is(err, errProvisioningUnavailable):
		return http.StatusServiceUnavailable
	default:
		if upstream, ok := errors.AsType[*domain.UpstreamError](err); ok {
			return upstream.StatusCode
		}
		return http.StatusInternalServerError
	}
}

func problemDetails(err error) (int, string, string) {
	status := problemStatus(err)
	if metadata, ok := domain.ErrorMetadataFrom(err); ok {
		return status, metadata.Title, metadata.Detail
	}
	if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return status, "request_too_large", fmt.Sprintf("request body exceeds %d bytes", maxBytesErr.Limit)
	}
	switch {
	case errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied):
		return http.StatusUnauthorized, "missing_token", "bearer token required"
	case errors.Is(err, auth.ErrUnauthorized), errors.Is(err, errMissingActor):
		return http.StatusUnauthorized, "invalid_token", "token is invalid"
	case errors.Is(err, errAuthUnavailable):
		return status, "authentication_failed", "authentication failed"
	case errors.Is(err, errProvisioningUnavailable):
		return status, "user_provisioning_failed", "user provisioning failed"
	case errors.Is(err, domain.ErrBadRequest):
		return status, "invalid_request", "invalid request"
	case errors.Is(err, domain.ErrForbidden):
		return status, "forbidden", "operation is not permitted"
	case errors.Is(err, domain.ErrNotFound):
		return status, "not_found", "resource not found"
	case errors.Is(err, domain.ErrConflict):
		return status, "conflict", "operation conflicts with existing data"
	case errors.Is(err, domain.ErrUpstreamTimeout):
		return status, "upstream_timeout", "request timed out"
	case errors.Is(err, domain.ErrUpstreamUnavailable):
		return status, "upstream_unavailable", "upstream service is unavailable"
	default:
		if status == http.StatusGatewayTimeout {
			return status, "upstream_timeout", "request timed out"
		}
		return status, "request_failed", "request failed"
	}
}
