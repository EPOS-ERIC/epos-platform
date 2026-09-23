package domain

import (
	"errors"
	"testing"
)

func TestNewNotificationTopicKey(t *testing.T) {
	key, err := NewNotificationTopicKey(" direct.resource.published ")
	if err != nil {
		t.Fatal(err)
	}
	if key != NotificationTopicDirectResourcePublished {
		t.Fatalf("key = %q, want %q", key, NotificationTopicDirectResourcePublished)
	}

	if _, err := NewNotificationTopicKey("resource.published"); !errors.Is(err, ErrNotificationInvalid) {
		t.Fatalf("error = %v, want ErrNotificationInvalid", err)
	}
}
