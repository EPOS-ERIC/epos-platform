package server

import (
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/grpcapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO: simplify this huge error mapper. return the correct formatted error already in the fuction that handles the error

// statusError keeps the internal cause available to server-side logging while
// exposing only the sanitized gRPC status to clients.
type statusError struct {
	cause  error
	status *status.Status
}

// NormalizeError maps a handler error to a client-safe gRPC status while
// retaining the original cause for server-side logging.
func NormalizeError(method string, cause error) error {
	if cause == nil {
		return nil
	}

	code, message := statusForError(method, cause)
	return &statusError{
		cause:  fmt.Errorf("%s: %w", method, cause),
		status: status.New(code, message),
	}
}

func statusForError(method string, cause error) (codes.Code, string) {
	name := methodName(method)
	if grpcStatus, ok := status.FromError(cause); ok {
		return grpcStatus.Code(), grpcStatus.Message()
	}

	if errors.Is(cause, domain.ErrInvalidAccess) {
		return codes.InvalidArgument, "invalid access request"
	}
	if errors.Is(cause, domain.ErrAccessForbidden) {
		return codes.PermissionDenied, "access operation is not permitted"
	}
	if errors.Is(cause, domain.ErrAccessNotFound) {
		return codes.NotFound, "access resource not found"
	}
	if errors.Is(cause, domain.ErrAccessConflict) {
		return codes.AlreadyExists, "access operation conflicts"
	}

	if errors.Is(cause, domain.ErrInvalidGroup) {
		if name == "GetGroup" {
			return codes.InvalidArgument, "invalid group id"
		}
		return codes.InvalidArgument, "invalid group"
	}
	if errors.Is(cause, domain.ErrGroupNotFound) {
		return codes.NotFound, "group not found"
	}
	if name != "GetGroup" && errors.Is(cause, domain.ErrGroupForbidden) {
		return codes.PermissionDenied, "group operation is not permitted"
	}
	if name != "GetGroup" && errors.Is(cause, domain.ErrGroupConflict) {
		return codes.AlreadyExists, "group operation conflicts with existing data"
	}

	switch name {
	case "ProvisionUser":
		if errors.Is(cause, domain.ErrInvalidUser) {
			return codes.InvalidArgument, "invalid user claims"
		}
		if errors.Is(cause, domain.ErrUserConflict) {
			return codes.AlreadyExists, "user already exists"
		}
	case "GetUser", "ListUsers":
		if errors.Is(cause, domain.ErrUserNotFound) {
			return codes.NotFound, "user not found"
		}
		if errors.Is(cause, domain.ErrUserForbidden) {
			return codes.PermissionDenied, "user read is not permitted"
		}
	default:
		if isUserMutation(name) {
			if errors.Is(cause, domain.ErrInvalidUser) {
				return codes.InvalidArgument, "invalid user permission request"
			}
			if errors.Is(cause, domain.ErrUserNotFound) {
				return codes.NotFound, "user not found"
			}
			if errors.Is(cause, domain.ErrUserConflict) {
				return codes.Aborted, "global role change conflicts with current users"
			}
			if errors.Is(cause, domain.ErrUserForbidden) {
				return codes.PermissionDenied, "user permission change is not permitted"
			}
		}
	}

	if errors.Is(cause, domain.ErrInvalidCategory) {
		return codes.InvalidArgument, "invalid category"
	}
	if errors.Is(cause, domain.ErrCategoryForbidden) {
		return codes.PermissionDenied, "category operation is not permitted"
	}
	if errors.Is(cause, domain.ErrCategoryNotFound) {
		return codes.NotFound, "category not found"
	}
	if errors.Is(cause, domain.ErrCategoryConflict) {
		return codes.AlreadyExists, "category operation conflicts with existing data"
	}

	if errors.Is(cause, domain.ErrInvalidOrganization) {
		return codes.InvalidArgument, "invalid organization"
	}
	if errors.Is(cause, domain.ErrOrganizationForbidden) {
		return codes.PermissionDenied, "organization operation is not permitted"
	}
	if errors.Is(cause, domain.ErrOrganizationNotFound) {
		return codes.NotFound, "organization not found"
	}
	if errors.Is(cause, domain.ErrOrganizationConflict) {
		return codes.AlreadyExists, "organization already exists"
	}

	if errors.Is(cause, domain.ErrInvalidContactPoint) {
		return codes.InvalidArgument, "invalid contact point"
	}
	if errors.Is(cause, domain.ErrContactPointForbidden) {
		return codes.PermissionDenied, "contact point operation is not permitted"
	}
	if errors.Is(cause, domain.ErrContactPointNotFound) {
		return codes.NotFound, "contact point not found"
	}
	if errors.Is(cause, domain.ErrContactPointConflict) {
		return codes.AlreadyExists, "contact point already exists"
	}

	if errors.Is(cause, domain.ErrInvalidDataset) ||
		errors.Is(cause, domain.ErrSoftwareApplicationInvalid) ||
		errors.Is(cause, domain.ErrSoftwareSourceCodeInvalid) {
		if name == "GetResourceVersion" {
			return codes.Internal, "read resource"
		}
		return codes.InvalidArgument, "invalid resource draft"
	}
	if errors.Is(cause, domain.ErrDatasetForbidden) ||
		errors.Is(cause, domain.ErrSoftwareApplicationForbidden) ||
		errors.Is(cause, domain.ErrSoftwareSourceCodeForbidden) {
		return codes.PermissionDenied, "resource operation is not permitted"
	}
	if errors.Is(cause, domain.ErrDatasetConflict) ||
		errors.Is(cause, domain.ErrSoftwareApplicationConflict) ||
		errors.Is(cause, domain.ErrSoftwareSourceCodeConflict) {
		if name == "GetResourceVersion" {
			return codes.Internal, "read resource"
		}
		return codes.AlreadyExists, "resource operation conflicts with existing data"
	}
	if errors.Is(cause, domain.ErrDatasetNotFound) ||
		errors.Is(cause, domain.ErrSoftwareApplicationNotFound) ||
		errors.Is(cause, domain.ErrSoftwareSourceCodeNotFound) {
		return codes.NotFound, "resource or version not found"
	}

	if errors.Is(cause, domain.ErrInvalidImport) {
		return codes.InvalidArgument, "invalid metadata import"
	}
	if errors.Is(cause, domain.ErrImportForbidden) {
		return codes.PermissionDenied, "global admin required"
	}
	if errors.Is(cause, domain.ErrImportConflict) {
		return codes.Aborted, "metadata import conflicts with existing data"
	}

	if errors.Is(cause, grpcapi.ErrInvalidDiscoveryFilter) {
		return codes.InvalidArgument, "invalid discovery filter"
	}
	if errors.Is(cause, domain.ErrInvalidDiscovery) {
		return codes.InvalidArgument, "invalid discovery request"
	}
	if errors.Is(cause, domain.ErrDiscoveryDenied) {
		return codes.Unauthenticated, "authentication is required for non-published discovery results"
	}

	return codes.Internal, operationForMethod(method)
}

func methodName(method string) string {
	if index := strings.LastIndexByte(method, '/'); index >= 0 {
		return method[index+1:]
	}
	return method
}

func operationForMethod(method string) string {
	switch methodName(method) {
	case "ProvisionUser":
		return "provision user"
	case "GetUser", "ListUsers":
		return "get user"
	case "ListGroups":
		return "list groups"
	case "GetGroup":
		return "get group"
	case "CreateGroup", "ReplaceGroup", "DeleteGroup":
		return "group operation"
	case "ListCategories", "GetCategory", "CreateCategory", "ReplaceCategory", "DeleteCategory":
		return "category operation"
	case "ListContactPoints", "GetContactPoint", "CreateContactPoint", "ReplaceContactPoint", "DeleteContactPoint":
		return "contact point operation"
	case "ListOrganizations", "GetOrganization", "CreateOrganization", "ReplaceOrganization", "DeleteOrganization":
		return "organization operation"
	case "GetResourceVersion", "ListResourceVersions":
		return "read resource"
	case "CreateResource", "ReplaceResourceDraft", "DiscardResourceDraft", "SubmitResource", "WithdrawResource", "RejectResource", "ReopenResource", "PublishResource", "UnpublishResource", "DraftResourceFromPublication", "DraftResourceFromLatestArchive", "CopyPublishedResource":
		return "resource operation"
	case "ImportTurtle":
		return "metadata import"
	case "SearchResources":
		return "discovery search"
	case "GrantGlobalPermission", "RevokeGlobalPermission", "AssignGlobalRole", "RemoveGlobalRole", "BlockUser", "UnblockUser":
		return "user operation"
	case "ListGroupMemberships":
		return "list memberships"
	case "RequestMembership":
		return "request membership"
	case "WithdrawMembership":
		return "withdraw membership"
	case "DecideMembership":
		return "decide membership"
	case "RevokeMembership":
		return "revoke membership"
	case "RequestRole":
		return "request role"
	case "ApplyRoleRequestDecision":
		return "decide role request"
	case "WithdrawRoleRequest":
		return "withdraw role request"
	case "AssignRole":
		return "assign role"
	case "RemoveRole":
		return "remove role"
	case "GrantPermission":
		return "grant permission"
	case "RevokePermission":
		return "revoke permission"
	default:
		return "request failed"
	}
}

func isUserMutation(method string) bool {
	switch method {
	case "GrantGlobalPermission", "RevokeGlobalPermission", "AssignGlobalRole", "RemoveGlobalRole", "BlockUser", "UnblockUser":
		return true
	default:
		return false
	}
}

func (e *statusError) Error() string {
	if e.cause == nil {
		return e.status.Message()
	}
	return e.cause.Error()
}

func (e *statusError) Unwrap() error {
	return e.cause
}

func (e *statusError) GRPCStatus() *status.Status {
	return e.status
}
