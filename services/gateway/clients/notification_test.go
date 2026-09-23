package clients

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"google.golang.org/grpc"
)

func TestFromProtoNotification(t *testing.T) {
	value := &metadatalifecyclev1.Notification{
		EventKind:   metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED,
		Channel:     metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL,
		TargetEmail: "user@example.org",
		Subject:     "Subject",
		Body:        "Body",
	}

	got, err := fromProtoNotification(value)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Notification{
		EventKind:   value.GetEventKind().String(),
		Channel:     domain.NotificationChannelEmail,
		TargetEmail: "user@example.org",
		Subject:     "Subject",
		Body:        "Body",
	}
	if got != want {
		t.Fatalf("fromProtoNotification() = %+v, want %+v", got, want)
	}
}

func TestFromProtoNotificationRejectsMalformedValues(t *testing.T) {
	tests := []struct {
		name  string
		value *metadatalifecyclev1.Notification
	}{
		{name: "nil", value: nil},
		{name: "unspecified event", value: &metadatalifecyclev1.Notification{Channel: metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL, TargetEmail: "user@example.org", Subject: "Subject", Body: "Body"}},
		{name: "unsupported channel", value: &metadatalifecyclev1.Notification{EventKind: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED, TargetEmail: "user@example.org", Subject: "Subject", Body: "Body"}},
		{name: "empty target", value: &metadatalifecyclev1.Notification{EventKind: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED, Channel: metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL, Subject: "Subject", Body: "Body"}},
		{name: "empty subject", value: &metadatalifecyclev1.Notification{EventKind: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED, Channel: metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL, TargetEmail: "user@example.org", Body: "Body"}},
		{name: "empty body", value: &metadatalifecyclev1.Notification{EventKind: metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED, Channel: metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL, TargetEmail: "user@example.org", Subject: "Subject"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := fromProtoNotification(tt.value); err == nil {
				t.Fatal("fromProtoNotification() error = nil, want error")
			}
		})
	}
}

func TestNotificationInterceptorDispatchesValidValues(t *testing.T) {
	dispatcher := &testNotificationDispatcher{}
	interceptor := notificationInterceptor(slog.New(slog.NewTextHandler(io.Discard, nil)), dispatcher)
	valid := &metadatalifecyclev1.Notification{
		EventKind:   metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED,
		Channel:     metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL,
		TargetEmail: "user@example.org",
		Subject:     "Subject",
		Body:        "Body",
	}

	response := &metadatalifecyclev1.RequestMembershipResponse{
		Notifications: []*metadatalifecyclev1.Notification{
			nil,
			{EventKind: valid.EventKind, Channel: valid.Channel, TargetEmail: valid.TargetEmail, Subject: valid.Subject},
			valid,
		},
	}
	invoked := false
	err := interceptor(context.Background(), "/metadata_lifecycle.v1.GroupAccessService/RequestMembership", nil, response, nil, func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		invoked = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !invoked {
		t.Fatal("RPC invoker was not called")
	}

	if len(dispatcher.notifications) != 1 || dispatcher.notifications[0].TargetEmail != valid.TargetEmail {
		t.Fatalf("dispatched notifications = %+v, want one valid notification", dispatcher.notifications)
	}
}

func TestNotificationInterceptorDoesNotDispatchOnRPCError(t *testing.T) {
	dispatcher := &testNotificationDispatcher{}
	interceptor := notificationInterceptor(slog.New(slog.NewTextHandler(io.Discard, nil)), dispatcher)
	wantErr := errors.New("RPC failed")
	response := &metadatalifecyclev1.RequestMembershipResponse{}
	err := interceptor(context.Background(), "/metadata_lifecycle.v1.GroupAccessService/RequestMembership", nil, response, nil, func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("interceptor() error = %v, want %v", err, wantErr)
	}
	if len(dispatcher.notifications) != 0 {
		t.Fatalf("dispatched notifications = %+v, want none", dispatcher.notifications)
	}
}

func TestNotificationInterceptorAllowsNilDispatcher(t *testing.T) {
	interceptor := notificationInterceptor(slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	invoked := false
	err := interceptor(context.Background(), "/metadata_lifecycle.v1.GroupService/ListGroups", nil, &metadatalifecyclev1.ListGroupsResponse{}, nil, func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		invoked = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !invoked {
		t.Fatal("RPC invoker was not called")
	}
}

type testNotificationDispatcher struct {
	notifications []domain.Notification
}

func (d *testNotificationDispatcher) Dispatch(_ context.Context, notifications []domain.Notification) {
	d.notifications = append(d.notifications, notifications...)
}
