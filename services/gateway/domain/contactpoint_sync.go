package domain

// SyncedContactPoint is a Drupal person with a primary email.
type SyncedContactPoint struct {
	NID   int64
	Email string
}
