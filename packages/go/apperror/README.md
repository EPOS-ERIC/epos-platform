# Shared application errors

This package defines the application-error contract shared by services and transport boundaries.

## Overview

An application error carries a stable machine-readable outcome, a safe client-facing explanation, and an optional field path. It may also retain a private cause for server-side diagnostics. Causes are never part of a client response.

Use a standard explanation when no more specific safe detail is available. When assigning an application outcome to a lower-level failure, retain that failure as the internal cause. Avoid composing separate public errors and causes with generic error formatting.

## Classification rules

- Invalid caller input is a client validation outcome. Include an actionable explanation and field path when known.
- Invalid persisted state and unexpected failures are server errors, not client validation failures.
- Authentication, authorization, missing resources, conflicts, invalid state transitions, unsupported content, oversized requests, and upstream timeout or availability failures have distinct outcomes. Preserve those distinctions across service boundaries.
- Unknown outcomes and unclassified errors fail closed as server errors.

## Field paths and causes

Create a field-specific error at the first layer that can identify the field. When a child error already has a path, preserve it while adding parent context. Keep array indices and nested field names intact.

Retain causes for server-side logging and tracing. Do not use cause text as client-facing detail, serialize causes, or log and return raw diagnostics at the same boundary.

## Transport boundaries

This package owns the shared application classification and its HTTP status mapping through `HTTPStatus`. Service boundaries encode HTTP response shapes; adapter packages such as `grpc/` map application errors to protocol-specific statuses and details. Keep mappings aligned with the shared classification, and expose only the safe outcome, explanation, and field path.

See Gateway and Metadata Lifecycle `AGENTS.md` files for service-specific boundary responsibilities.
