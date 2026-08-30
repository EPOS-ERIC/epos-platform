// Package transport contains shared HTTP transport helpers.
package transport

import (
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

const ActorHeader = "X-EPOS-User-ID"

// Actor returns a validated actor and whether the request contains one.
func Actor(w http.ResponseWriter, r *http.Request) (actor domain.Actor, valid bool) {
	value, valid := OptionalActor(w, r)
	if !valid {
		return "", false
	}

	if value == nil {
		http.Error(w, ActorHeader+" is required", http.StatusUnauthorized)
		return "", false
	}

	return *value, true
}

// OptionalActor returns a validated actor, or nil when the header is absent.
func OptionalActor(w http.ResponseWriter, r *http.Request) (actor *domain.Actor, ok bool) {
	if strings.TrimSpace(r.Header.Get(ActorHeader)) == "" {
		return nil, true
	}

	parsedActor, err := domain.NewActor(r.Header.Get(ActorHeader))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, false
	}

	return &parsedActor, true
}
