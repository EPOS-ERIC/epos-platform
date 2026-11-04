package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

// NewNotificationTopicKey validates and normalizes a notification topic key.
func NewNotificationTopicKey(value string) (NotificationTopicKey, error) {
	key := NotificationTopicKey(strings.TrimSpace(value))
	switch key {
	case NotificationTopicDirectMembershipAccepted,
		NotificationTopicDirectMembershipRejected,
		NotificationTopicDirectRoleRequestAccepted,
		NotificationTopicDirectRoleRequestRejected,
		NotificationTopicDirectRoleAssigned,
		NotificationTopicDirectPermissionAssigned,
		NotificationTopicDirectResourcePublished,
		NotificationTopicDirectResourceRejected,
		NotificationTopicDirectResourceUnpublished,
		NotificationTopicGroupMembershipRequested,
		NotificationTopicGlobalMembershipRequested,
		NotificationTopicGroupRoleRequestRequested,
		NotificationTopicGlobalRoleRequestRequested,
		NotificationTopicGroupResourceSubmitted,
		NotificationTopicGroupResourcePublished,
		NotificationTopicGlobalResourceSubmitted,
		NotificationTopicGlobalResourcePublished,
		NotificationTopicGlobalConversionRequested:
		return key, nil
	default:
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
}
