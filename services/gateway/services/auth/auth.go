package auth

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
)

var log = logging.Get("auth service")

type Service struct {
	AAIServiceURL string
}

func NewService(aaiServiceURL string) *Service {
	log.Debug("creating new AuthService instance")
	return &Service{
		AAIServiceURL: aaiServiceURL,
	}
}
