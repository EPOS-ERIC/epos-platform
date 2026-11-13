package clients

import (
	"context"
	"log/slog"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"google.golang.org/grpc"
)

type notificationResponse interface {
	GetNotifications() []*metadatalifecyclev1.Notification
}

func notificationInterceptor(log *slog.Logger, dispatcher NotificationDispatcher) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			return err
		}
		if dispatcher == nil {
			return nil
		}

		// if the response does not have notifications, skip
		response, ok := reply.(notificationResponse)
		if !ok {
			return nil
		}

		// map valid notifications
		notifications := make([]domain.Notification, 0, len(response.GetNotifications()))
		for _, value := range response.GetNotifications() {
			notification, err := fromProtoNotification(value)
			if err != nil {
				log.WarnContext(ctx, "skip invalid notification", "method", method, "error", err)
				continue
			}
			notifications = append(notifications, notification)
		}

		// dispatch
		if len(notifications) > 0 {
			dispatcher.Dispatch(ctx, notifications)
		}

		return nil
	}
}
