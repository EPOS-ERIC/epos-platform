package domain

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

func TestNewNotificationTopicKey(t *testing.T) {
	key, err := ParseNotificationTopicKey(" direct.resource.published ")
	if err != nil {
		t.Fatal(err)
	}
	if key != NotificationTopicDirectResourcePublished {
		t.Fatalf("key = %q, want %q", key, NotificationTopicDirectResourcePublished)
	}

	if _, err := ParseNotificationTopicKey("resource.published"); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("error = %v, want validation_failed", err)
	}
}
