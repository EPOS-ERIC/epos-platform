// Package softwarelifecycle manages software versions and their lifecycle.
package softwarelifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/google/uuid"
)

var (
	ErrDenied   = errors.New("permission denied")
	ErrConflict = errors.New("lifecycle conflict")
	ErrNotFound = errors.New("software not found")
	ErrInvalid  = errors.New("invalid software request")
	ErrInternal = errors.New("software operation failed")
)

const (
	permissionCreate    = "resource.create"
	permissionEdit      = "resource.edit"
	permissionSubmit    = "resource.submit"
	permissionWithdraw  = "resource.withdraw"
	permissionReject    = "resource.reject"
	permissionReopen    = "resource.reopen"
	permissionPublish   = "resource.publish"
	permissionDiscard   = "resource.discard"
	permissionUnpublish = "resource.unpublish"
)

type ApplicationAuthorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (domain.Result, error)
	AuthorizeResource(context.Context, string, domain.ResourceType, uuid.UUID, string) (domain.Result, error)
	CanReadSoftwareApplicationVersion(context.Context, *string, uuid.UUID, uuid.UUID) (domain.Result, error)
	CanReadContactPoint(context.Context, string) (domain.Result, error)
}

type SourceCodeAuthorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (domain.Result, error)
	AuthorizeResource(context.Context, string, domain.ResourceType, uuid.UUID, string) (domain.Result, error)
	CanReadSoftwareSourceCodeVersion(context.Context, *string, uuid.UUID, uuid.UUID) (domain.Result, error)
	CanReadContactPoint(context.Context, string) (domain.Result, error)
}

func authorizeGroup(ctx context.Context, authorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (domain.Result, error)
}, actorID string, groupID uuid.UUID, permission string,
) error {
	result, err := authorizer.AuthorizeGroup(ctx, actorID, groupID, permission)
	if err != nil {
		return fmt.Errorf("authorize group: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", ErrDenied, result.Reason)
	}
	return nil
}

func authorizeResource(ctx context.Context, authorizer interface {
	AuthorizeResource(context.Context, string, domain.ResourceType, uuid.UUID, string) (domain.Result, error)
}, actorID string, resourceType domain.ResourceType, resourceID uuid.UUID, permission string,
) error {
	result, err := authorizer.AuthorizeResource(ctx, actorID, resourceType, resourceID, permission)
	if err != nil {
		return fmt.Errorf("authorize software: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", ErrDenied, result.Reason)
	}
	return nil
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrDenied) || errors.Is(err, ErrConflict) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalid) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, ErrInternal, err)
}

func validateIdentifiers(values []metadata.Identifier) error {
	for _, value := range values {
		if strings.TrimSpace(string(value.Type)) == "" || strings.TrimSpace(value.Value) == "" {
			return fmt.Errorf("identifier type and value are required: %w", ErrInvalid)
		}
	}
	return nil
}

func validateReferences(name string, values []uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return fmt.Errorf("%s ID is required: %w", name, ErrInvalid)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate %s %q: %w", name, value, ErrInvalid)
		}
		seen[value] = struct{}{}
	}
	return nil
}
