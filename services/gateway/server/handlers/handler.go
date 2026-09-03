//nolint:dupl,exhaustruct_v5,wrapcheck // Direct transport mapping intentionally constructs partial domain/response values and returns service errors unchanged for Ogen classification.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/canonical"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	groupservice "github.com/EPOS-ERIC/epos-platform/services/gateway/groups"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	metadataimports "github.com/EPOS-ERIC/epos-platform/services/gateway/imports"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/ogen-go/ogen/ogenerrors"
)

var (
	errAuthUnavailable         = errors.New("authentication unavailable")
	errProvisioningUnavailable = errors.New("user provisioning unavailable")
)

// ogenHandler is the HTTP boundary. It calls domain services directly; no
// request is re-encoded or dispatched through a second HTTP transport.
type ogenHandler struct {
	canonical *canonical.Service
	discovery *discovery.Service
	groups    *groupservice.Service
	identity  *identity.Service
	imports   *metadataimports.Service
	lifecycle *lifecycle.Service
}

func NewOgenHandler(c *canonical.Service, d *discovery.Service, g *groupservice.Service, i *identity.Service, m *metadataimports.Service, l *lifecycle.Service) *ogenHandler {
	return &ogenHandler{canonical: c, discovery: d, groups: g, identity: i, imports: m, lifecycle: l}
}

var _ gatewayapi.Handler = (*ogenHandler)(nil)

func (h *ogenHandler) NewError(ctx context.Context, err error) *gatewayapi.ProblemStatusCode {
	status, title, detail := problemDetails(err)
	response := gatewayapi.Problem{Title: title, Status: status}
	if detail != "" {
		response.Detail = gatewayapi.NewOptString(detail)
	}
	return &gatewayapi.ProblemStatusCode{
		StatusCode: status,
		Response:   response,
	}
}

var errMissingActor = errors.New("bearer token required")

type authenticatedActorKey struct{}

func withActor(ctx context.Context, actor domain.User) context.Context {
	return context.WithValue(ctx, authenticatedActorKey{}, actor)
}

func actorID(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(authenticatedActorKey{}).(domain.User)
	return user.ID, ok
}

func fromOptNilString(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return &v.Value
}

func problemStatus(err error) int {
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
	default:
		switch status {
		case http.StatusGatewayTimeout:
			return status, "upstream_timeout", "request timed out"
		}
		return status, "request_failed", "request failed"
	}
}
