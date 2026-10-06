package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoListActivityRequest(value gatewayapi.ListActivityParams) *metadatalifecyclev1.ListActivityRequest {
	result := &metadatalifecyclev1.ListActivityRequest{}
	if value.GroupId.Set {
		result.GroupId = new(value.GroupId.Value.String())
	}
	if value.DateFrom.Set {
		result.DateFrom = timestamppb.New(value.DateFrom.Value)
	}
	if value.DateTo.Set {
		result.DateTo = timestamppb.New(value.DateTo.Value)
	}
	return result
}

func toAPIActivityEvent(value *metadatalifecyclev1.ActivityEvent) (gatewayapi.ActivityEvent, error) {
	if value == nil {
		return gatewayapi.ActivityEvent{}, fmt.Errorf("metadata-lifecycle returned a nil activity event")
	}
	if value.OccurredAt == nil {
		return gatewayapi.ActivityEvent{}, fmt.Errorf("metadata-lifecycle activity event omitted occurred_at")
	}
	if err := value.OccurredAt.CheckValid(); err != nil {
		return gatewayapi.ActivityEvent{}, fmt.Errorf("metadata-lifecycle activity event has invalid occurred_at: %w", err)
	}
	action, err := toAPIActivityAction(value.Action)
	if err != nil {
		return gatewayapi.ActivityEvent{}, err
	}
	result := gatewayapi.ActivityEvent{Action: action, OccurredAt: value.OccurredAt.AsTime()}
	if value.ActorId != nil {
		result.ActorID = gatewayapi.NewOptNilString(*value.ActorId)
	}
	if value.GroupId != nil {
		id, err := parseUUID("activity group id", *value.GroupId)
		if err != nil {
			return gatewayapi.ActivityEvent{}, err
		}
		result.GroupID = gatewayapi.NewOptNilUUID(id)
	}
	if value.ResourceType != nil {
		resourceType, err := toAPIDiscoveryResourceType(*value.ResourceType)
		if err != nil {
			return gatewayapi.ActivityEvent{}, err
		}
		result.ResourceType = gatewayapi.NewOptNilDiscoveryResourceType(resourceType)
	}
	if value.ResourceId != nil {
		id, err := parseUUID("activity resource id", *value.ResourceId)
		if err != nil {
			return gatewayapi.ActivityEvent{}, err
		}
		result.ResourceID = gatewayapi.NewOptNilUUID(id)
	}
	if value.VersionId != nil {
		id, err := parseUUID("activity version id", *value.VersionId)
		if err != nil {
			return gatewayapi.ActivityEvent{}, err
		}
		result.VersionID = gatewayapi.NewOptNilUUID(id)
	}
	if value.TargetUserId != nil {
		result.TargetUserID = gatewayapi.NewOptNilString(*value.TargetUserId)
	}
	if value.Role != nil {
		result.Role = gatewayapi.NewOptNilString(*value.Role)
	}
	if value.Permission != nil {
		result.Permission = gatewayapi.NewOptNilString(*value.Permission)
	}
	if value.Note != nil {
		result.Note = gatewayapi.NewOptNilString(*value.Note)
	}
	if value.Reason != nil {
		result.Reason = gatewayapi.NewOptNilString(*value.Reason)
	}
	return result, nil
}

func toAPIActivityAction(value metadatalifecyclev1.ActivityAction) (gatewayapi.ActivityAction, error) {
	switch value {
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_CREATED:
		return gatewayapi.ActivityActionRESOURCECREATED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_VERSION_CREATED:
		return gatewayapi.ActivityActionRESOURCEVERSIONCREATED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_SUBMITTED:
		return gatewayapi.ActivityActionRESOURCESUBMITTED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_REJECTED:
		return gatewayapi.ActivityActionRESOURCEREJECTED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_PUBLISHED:
		return gatewayapi.ActivityActionRESOURCEPUBLISHED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_RESOURCE_ARCHIVED:
		return gatewayapi.ActivityActionRESOURCEARCHIVED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_REQUESTED:
		return gatewayapi.ActivityActionROLEREQUESTED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_REQUEST_ACCEPTED:
		return gatewayapi.ActivityActionROLEREQUESTACCEPTED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_REQUEST_REJECTED:
		return gatewayapi.ActivityActionROLEREQUESTREJECTED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_ROLE_ASSIGNED:
		return gatewayapi.ActivityActionROLEASSIGNED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_PERMISSION_GRANTED:
		return gatewayapi.ActivityActionPERMISSIONGRANTED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_USER_REGISTERED:
		return gatewayapi.ActivityActionUSERREGISTERED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_USER_BLOCKED:
		return gatewayapi.ActivityActionUSERBLOCKED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_GROUP_CREATED:
		return gatewayapi.ActivityActionGROUPCREATED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_CATEGORY_CREATED:
		return gatewayapi.ActivityActionCATEGORYCREATED, nil
	case metadatalifecyclev1.ActivityAction_ACTIVITY_ACTION_MEMBERSHIP_CREATED:
		return gatewayapi.ActivityActionMEMBERSHIPCREATED, nil
	default:
		return "", fmt.Errorf("unknown metadata-lifecycle activity action %q", value)
	}
}
