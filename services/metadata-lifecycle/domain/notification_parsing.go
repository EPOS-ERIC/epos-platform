package domain

import "strings"

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
		NotificationTopicGlobalResourcePublished:
		return key, nil
	default:
		return "", ErrNotificationInvalid
	}
}
