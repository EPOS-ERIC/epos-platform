package domain

import "errors"

var (
	ErrNotificationInvalid   = errors.New("invalid notification request")
	ErrNotificationNotFound  = errors.New("notification resource not found")
	ErrNotificationForbidden = errors.New("notification operation forbidden")
)
