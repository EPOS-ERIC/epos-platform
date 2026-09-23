package domain

// NotificationChannel identifies how a notification is delivered.
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"
)

// Notification contains the fields required to deliver a notification.
type Notification struct {
	EventKind   string
	Channel     NotificationChannel
	TargetEmail string
	Subject     string
	Body        string
}
