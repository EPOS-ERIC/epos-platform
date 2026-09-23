package grpcapi

import (
	"testing"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToProtoNotificationEventKind(t *testing.T) {
	tests := []struct {
		name  string
		value domain.NotificationEventKind
		want  metadatalifecyclev1.NotificationEventKind
	}{
		{name: "membership requested", value: domain.NotificationEventMembershipRequested, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_REQUESTED},
		{name: "membership accepted", value: domain.NotificationEventMembershipAccepted, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED},
		{name: "membership rejected", value: domain.NotificationEventMembershipRejected, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_REJECTED},
		{name: "role requested", value: domain.NotificationEventRoleRequested, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_REQUESTED},
		{name: "role request accepted", value: domain.NotificationEventRoleRequestAccepted, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_ACCEPTED},
		{name: "role request rejected", value: domain.NotificationEventRoleRequestRejected, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_REJECTED},
		{name: "role assigned", value: domain.NotificationEventRoleAssigned, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_ASSIGNED},
		{name: "permission assigned", value: domain.NotificationEventPermissionAssigned, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_PERMISSION_ASSIGNED},
		{name: "resource submitted", value: domain.NotificationEventResourceSubmitted, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_SUBMITTED},
		{name: "resource published", value: domain.NotificationEventResourcePublished, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_PUBLISHED},
		{name: "resource rejected", value: domain.NotificationEventResourceRejected, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_REJECTED},
		{name: "resource unpublished", value: domain.NotificationEventResourceUnpublished, want: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_UNPUBLISHED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toProtoNotificationEventKind(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("toProtoNotificationEventKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToProtoNotificationRejectsUnknownEventKind(t *testing.T) {
	_, err := toProtoNotification(domain.Notification{
		EventKind: domain.NotificationEventKind("unknown"),
		Channel:   domain.NotificationChannelEmail,
	})
	if err == nil {
		t.Fatal("toProtoNotification() error = nil, want error")
	}
}

func TestToProtoNotificationPreservesOptionalIDs(t *testing.T) {
	groupID := uuid.New()
	resourceID := uuid.New()
	versionID := uuid.New()
	accessID := uuid.New()

	result, err := toProtoNotification(domain.Notification{
		EventKind:    domain.NotificationEventMembershipRequested,
		GroupID:      &groupID,
		ResourceID:   &resourceID,
		VersionID:    &versionID,
		AccessID:     &accessID,
		TargetUserID: "user",
		TargetEmail:  "user@example.org",
		Subject:      "Subject",
		Body:         "Body",
		Channel:      domain.NotificationChannelEmail,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.GroupId == nil || result.GetGroupId() != groupID.String() {
		t.Fatalf("group_id = %v, want %q", result.GroupId, groupID)
	}
	if result.ResourceId == nil || result.GetResourceId() != resourceID.String() {
		t.Fatalf("resource_id = %v, want %q", result.ResourceId, resourceID)
	}
	if result.VersionId == nil || result.GetVersionId() != versionID.String() {
		t.Fatalf("version_id = %v, want %q", result.VersionId, versionID)
	}
	if result.AccessId == nil || result.GetAccessId() != accessID.String() {
		t.Fatalf("access_id = %v, want %q", result.AccessId, accessID)
	}
}

func TestNotificationTopicKeyMappings(t *testing.T) {
	tests := []struct {
		name   string
		domain domain.NotificationTopicKey
		proto  metadatalifecyclev1.NotificationTopicKey
	}{
		{
			name:   "group membership",
			domain: domain.NotificationTopicGroupMembershipRequested,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_MEMBERSHIP_REQUESTED,
		},
		{
			name:   "global membership",
			domain: domain.NotificationTopicGlobalMembershipRequested,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_MEMBERSHIP_REQUESTED,
		},
		{
			name:   "group role request",
			domain: domain.NotificationTopicGroupRoleRequestRequested,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_ROLE_REQUEST_REQUESTED,
		},
		{
			name:   "global role request",
			domain: domain.NotificationTopicGlobalRoleRequestRequested,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_ROLE_REQUEST_REQUESTED,
		},
		{
			name:   "group resource submitted",
			domain: domain.NotificationTopicGroupResourceSubmitted,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_SUBMITTED,
		},
		{
			name:   "group resource published",
			domain: domain.NotificationTopicGroupResourcePublished,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_PUBLISHED,
		},
		{
			name:   "global resource submitted",
			domain: domain.NotificationTopicGlobalResourceSubmitted,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_SUBMITTED,
		},
		{
			name:   "global resource published",
			domain: domain.NotificationTopicGlobalResourcePublished,
			proto:  metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_PUBLISHED,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			protoValue, err := toProtoNotificationTopicKey(test.domain)
			if err != nil {
				t.Fatal(err)
			}
			if protoValue != test.proto {
				t.Fatalf("toProtoNotificationTopicKey() = %v, want %v", protoValue, test.proto)
			}

			domainValue, err := toDomainNotificationTopicKey(test.proto)
			if err != nil {
				t.Fatal(err)
			}
			if domainValue != test.domain {
				t.Fatalf("toDomainNotificationTopicKey() = %q, want %q", domainValue, test.domain)
			}
		})
	}
}

func TestNotificationTopicKeyMappingsRejectInvalidValues(t *testing.T) {
	_, err := toDomainNotificationTopicKey(metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_UNSPECIFIED)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid proto topic key error code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}

	_, err = toProtoNotificationTopicKey(domain.NotificationTopicDirectResourcePublished)
	if status.Code(err) != codes.Internal {
		t.Fatalf("direct topic key error code = %s, want %s", status.Code(err), codes.Internal)
	}
}
