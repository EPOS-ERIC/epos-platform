package authz

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func recordAuthzDecision(ctx context.Context, operation string, allowed bool, reason string, err error) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attrs := []attribute.KeyValue{attribute.String("authz.operation", operation)}
	if err != nil {
		attrs = append(attrs, attribute.String("authz.outcome", "error"))
	} else {
		attrs = append(attrs,
			attribute.Bool("authz.allowed", allowed),
			attribute.String("authz.reason", reason),
		)
	}

	span.AddEvent("authz.decision", trace.WithAttributes(attrs...))
}

func recordAuthzDiscovery(ctx context.Context, actorPresent bool, candidateCount, allowedCandidateCount int, err error) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.Bool("authz.actor_present", actorPresent),
		attribute.Int("authz.candidate_count", candidateCount),
		attribute.Int("authz.allowed_candidate_count", allowedCandidateCount),
	}
	if err != nil {
		attrs = append(attrs, attribute.String("authz.outcome", "error"))
	}

	span.AddEvent("authz.discovery", trace.WithAttributes(attrs...))
}
