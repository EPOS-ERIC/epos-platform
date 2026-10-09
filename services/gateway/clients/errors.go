package clients

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	apperrorgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/apperror/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func wrapRPCError(log *slog.Logger, ctx context.Context, service, operation string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return fmt.Errorf("%s: %w", operation, apperror.Wrap(apperror.CodeUpstreamTimeout, "", err))
		}
		return fmt.Errorf("%s: %w", operation, ctxErr)
	}

	if appErr, ok := apperrorgrpc.FromStatus(err); ok {
		return fmt.Errorf("%s: %w", operation, appErr)
	}

	switch status.Code(err) {
	case codes.Unavailable:
		log.ErrorContext(ctx, service+" request failed", "operation", operation, "code", status.Code(err), "error", err)
		return fmt.Errorf("%s: %w", operation, apperror.Wrap(apperror.CodeUpstreamUnavailable, "", err))
	case codes.DeadlineExceeded:
		return fmt.Errorf("%s: %w", operation, apperror.Wrap(apperror.CodeUpstreamTimeout, "", err))
	case codes.OK,
		codes.Canceled,
		codes.Unknown,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Internal,
		codes.DataLoss,
		codes.Unauthenticated:
		return fmt.Errorf("%s: %w", operation, apperror.Wrap(apperror.CodeInternal, "", err))
	default:
		return fmt.Errorf("%s: %w", operation, apperror.Wrap(apperror.CodeInternal, "", err))
	}
}
