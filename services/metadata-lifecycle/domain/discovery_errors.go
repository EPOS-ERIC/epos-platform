package domain

import "errors"

var (
	ErrInvalidDiscovery = errors.New("invalid discovery")
	ErrDiscoveryDenied  = errors.New("discovery preview requires authentication")
)
