package clients

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func wrapRPCError(log *slog.Logger, ctx context.Context, service, operation string, err error) error {
	if status.Code(err) == codes.Unavailable {
		log.ErrorContext(ctx, "upstream request failed", "service", service, "operation", operation, "code", status.Code(err), "error", err)
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}

	switch status.Code(err) {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return fmt.Errorf("%s: %w", operation, domain.ErrBadRequest)
	case codes.NotFound:
		return fmt.Errorf("%s: %w", operation, domain.ErrNotFound)
	case codes.AlreadyExists, codes.Aborted:
		return fmt.Errorf("%s: %w", operation, domain.ErrConflict)
	case codes.PermissionDenied:
		return fmt.Errorf("%s: %w", operation, domain.ErrForbidden)
	case codes.Unauthenticated:
		return fmt.Errorf("%s: %w", operation, domain.ErrUnauthorized)
	case codes.DeadlineExceeded, codes.Canceled:
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	case codes.Unavailable:
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamUnavailable)
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}
