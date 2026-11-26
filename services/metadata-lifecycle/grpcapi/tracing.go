package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ActorTracingInterceptor adds the actor ID from trusted internal metadata to
// the current server span without changing RPC behavior.
func ActorTracingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if incomingMetadata, ok := metadata.FromIncomingContext(ctx); ok {
			values := incomingMetadata.Get(actorMetadataKey)
			if len(values) == 1 {
				if actor, err := domain.NormalizeActor(values[0]); err == nil {
					observability.SetActorID(ctx, actor)
				}
			}
		}

		return handler(ctx, req)
	}
}
