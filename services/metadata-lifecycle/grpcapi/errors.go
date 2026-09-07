package grpcapi

import (
	"fmt"

	"google.golang.org/grpc/status"
)

func wrapHandlerError(cause error) error {
	if cause == nil {
		return nil
	}
	if _, ok := status.FromError(cause); ok {
		return cause
	}
	return fmt.Errorf("grpc handler: %w", cause)
}
