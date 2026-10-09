package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/membership"
	sharingservice "github.com/EPOS-ERIC/epos-platform/services/gateway/sharing"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/ogen-go/ogen/validate"
	"go.opentelemetry.io/otel/trace"
)

// ogenHandler is the HTTP boundary. Pure proxy operations call the concrete
// Metadata Lifecycle client; identity orchestration remains in identity.Service.
type ogenHandler struct {
	metadataLifecycle *clients.MetadataLifecycle
	sharing           *sharingservice.Service
	identity          *identity.Service
	membership        *membership.Service
	log               *slog.Logger
}

func NewOgenHandler(i *identity.Service, m *membership.Service, metadataLifecycle *clients.MetadataLifecycle, sharing *sharingservice.Service, log *slog.Logger) *ogenHandler {
	return &ogenHandler{
		metadataLifecycle: metadataLifecycle,
		sharing:           sharing,
		identity:          i,
		membership:        m,
		log:               log,
	}
}

var _ gatewayapi.Handler = (*ogenHandler)(nil)

func (h *ogenHandler) NewError(ctx context.Context, err error) *gatewayapi.ProblemStatusCode {
	status, response := problemResponse(ctx, err)
	reportHTTPError(h.log, ctx, err, status)
	return &gatewayapi.ProblemStatusCode{
		StatusCode: status,
		Response:   response,
	}
}

// problemResponse maps an error to the Gateway HTTP status and Problem body.
func problemResponse(ctx context.Context, err error) (int, gatewayapi.Problem) {
	appErr := applicationError(err)
	status := apperror.HTTPStatus(appErr.Code)
	problem := gatewayapi.Problem{
		Type:   string(appErr.Code),
		Title:  http.StatusText(status),
		Status: status,
		Detail: gatewayapi.NewOptString(appErr.Detail),
	}
	if appErr.Target != "" {
		problem.Target = gatewayapi.NewOptString(appErr.Target)
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.TraceID().IsValid() {
		problem.Instance = gatewayapi.NewOptString(spanContext.TraceID().String())
	}
	return status, problem
}

// applicationError normalizes transport and application errors to a safe AppError.
func applicationError(err error) *apperror.AppError {
	if appErr, ok := errors.AsType[*apperror.AppError](err); ok {
		if !apperror.Known(appErr.Code) {
			return apperror.Default(apperror.CodeInternal)
		}
		safe := apperror.New(appErr.Code, appErr.Detail)
		safe.Target = appErr.Target
		return safe
	}

	if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return apperror.New(apperror.CodeRequestTooLarge, fmt.Sprintf("Request body exceeds %d bytes.", maxBytesErr.Limit))
	}

	if _, ok := errors.AsType[*validate.InvalidContentTypeError](err); ok {
		return apperror.Default(apperror.CodeUnsupportedMedia)
	}

	validationErr, ok := errors.AsType[*validate.Error](err)
	if ok && len(validationErr.Fields) > 0 {
		target, fieldErr := validationLeaf(err, "")
		return apperror.Field(target, validationDetail(fieldErr), err)
	}

	_, decodeRequest := errors.AsType[*ogenerrors.DecodeRequestError](err)
	_, decodeParams := errors.AsType[*ogenerrors.DecodeParamsError](err)
	if decodeRequest || decodeParams {
		return apperror.New(apperror.CodeValidationFailed, "The request is invalid.")
	}

	switch {
	case errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied):
		return apperror.Default(apperror.CodeUnauthenticated)
	case errors.Is(err, context.DeadlineExceeded):
		return apperror.Default(apperror.CodeUpstreamTimeout)
	default:
		return apperror.Default(apperror.CodeInternal)
	}
}

func validationLeaf(err error, target string) (string, error) {
	validationErr, ok := errors.AsType[*validate.Error](err)
	if !ok || len(validationErr.Fields) == 0 {
		return target, err
	}
	field := validationErr.Fields[0]
	switch {
	case strings.HasPrefix(field.Name, "["):
		target += field.Name
	case target == "":
		target = field.Name
	default:
		target += "." + field.Name
	}
	return validationLeaf(field.Error, target)
}

func validationDetail(err error) string {
	if errors.Is(err, validate.ErrFieldRequired) {
		return "Field is required."
	}
	if minLengthErr, ok := errors.AsType[*validate.MinLengthError](err); ok {
		if minLengthErr.Len == 0 {
			return "Field must not be blank."
		}
		return fmt.Sprintf("Must contain at least %d characters.", minLengthErr.MinLength)
	}
	if maxLengthErr, ok := errors.AsType[*validate.MaxLengthError](err); ok {
		return fmt.Sprintf("Must contain no more than %d characters.", maxLengthErr.MaxLength)
	}
	if _, ok := errors.AsType[*validate.NoRegexMatchError](err); ok {
		return "Value has an invalid format."
	}
	if _, ok := errors.AsType[*validate.DuplicateItemsError](err); ok {
		return "Items must be unique."
	}
	return "Value is invalid."
}
