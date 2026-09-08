package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const actorIDAttribute = "enduser.id"

// SetActorID adds the authenticated actor to the current trace span.
func SetActorID(ctx context.Context, actorID string) {
	if actorID == "" {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.SetAttributes(attribute.String(actorIDAttribute, actorID))
}
