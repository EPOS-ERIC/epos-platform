package grpcapi

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toDomainActivityFilter(value *metadatalifecyclev1.ListActivityRequest) (domain.ActivityFilter, error) {
	var groupID *uuid.UUID
	if value.GroupId != nil {
		id, err := uuid.Parse(*value.GroupId)
		if err != nil {
			return domain.ActivityFilter{}, apperror.Field("groupId", "Must be a valid UUID.", err)
		}
		groupID = &id
	}

	dateFrom, err := timestampInput(value.DateFrom)
	if err != nil {
		return domain.ActivityFilter{}, err
	}

	dateTo, err := timestampInput(value.DateTo)
	if err != nil {
		return domain.ActivityFilter{}, err
	}

	return domain.ActivityFilter{
		GroupID:  groupID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
	}, nil
}

func toProtoActivityEvent(value domain.ActivityEvent) (*metadatalifecyclev1.ActivityEvent, error) {
	action, err := toProtoActivityAction(value.Action)
	if err != nil {
		return nil, err
	}

	result := &metadatalifecyclev1.ActivityEvent{
		Action:       action,
		OccurredAt:   timestamppb.New(value.OccurredAt),
		ActorId:      value.ActorID,
		TargetUserId: value.TargetUserID,
		Role:         value.Role,
		Permission:   value.Permission,
		Note:         value.Note,
		Reason:       value.Reason,
	}

	if value.GroupID != nil {
		result.GroupId = new(value.GroupID.String())
	}

	if value.ResourceType != nil {
		resourceType := toGRPCResourceType(*value.ResourceType)
		if resourceType == metadatalifecyclev1.ResourceType_RESOURCE_TYPE_UNSPECIFIED {
			return nil, apperror.Default(apperror.CodeInternal)
		}
		result.ResourceType = &resourceType
	}

	if value.ResourceID != nil {
		result.ResourceId = new(value.ResourceID.String())
	}

	if value.VersionID != nil {
		result.VersionId = new(value.VersionID.String())
	}

	return result, nil
}

func toProtoActivityAction(value domain.ActivityAction) (metadatalifecyclev1.ActivityAction, error) {
	switch value {
	case domain.ActivityActionResourceCreated:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_CREATED, nil
	case domain.ActivityActionResourceVersionCreated:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_VERSION_CREATED, nil
	case domain.ActivityActionResourceSubmitted:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_SUBMITTED, nil
	case domain.ActivityActionResourceRejected:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_REJECTED, nil
	case domain.ActivityActionResourcePublished:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_PUBLISHED, nil
	case domain.ActivityActionResourceArchived:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_ARCHIVED, nil
	case domain.ActivityActionRoleRequested:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_REQUESTED, nil
	case domain.ActivityActionRoleRequestAccepted:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_REQUEST_ACCEPTED, nil
	case domain.ActivityActionRoleRequestRejected:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_REQUEST_REJECTED, nil
	case domain.ActivityActionRoleAssigned:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_ASSIGNED, nil
	case domain.ActivityActionPermissionGranted:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_PERMISSION_GRANTED, nil
	case domain.ActivityActionUserRegistered:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_USER_REGISTERED, nil
	case domain.ActivityActionUserBlocked:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_USER_BLOCKED, nil
	case domain.ActivityActionGroupCreated:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_GROUP_CREATED, nil
	case domain.ActivityActionCategoryCreated:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_CATEGORY_CREATED, nil
	case domain.ActivityActionMembershipCreated:
		return metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_MEMBERSHIP_CREATED, nil
	default:
		return 0, apperror.Default(apperror.CodeInternal)
	}
}
