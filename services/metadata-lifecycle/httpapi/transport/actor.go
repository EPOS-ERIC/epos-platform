// Package transport contains shared HTTP transport helpers.
package transport

import (
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

const ActorHeader = "X-EPOS-User-ID"

// Actor returns actor ID and whether request contains a valid actor header.
func Actor(w http.ResponseWriter, r *http.Request) (actorID string, valid bool) {
	id, present, ok := OptionalActor(w, r)
	if !ok {
		return "", false
	}
	if !present {
		http.Error(w, ActorHeader+" is required", http.StatusUnauthorized)
		return "", false
	}
	return id, true
}

// OptionalActor returns actor ID, whether header is present, and whether value is valid.
func OptionalActor(w http.ResponseWriter, r *http.Request) (actorID string, present, valid bool) {
	value := strings.TrimSpace(r.Header.Get(ActorHeader))
	if value == "" {
		return "", false, true
	}
	return value, true, true
}

// ActorValue returns a validated actor and whether the request contains one.
func ActorValue(w http.ResponseWriter, r *http.Request) (actor domain.Actor, valid bool) {
	value, valid := OptionalActorValue(w, r)
	if !valid {
		return "", false
	}

	if value == nil {
		http.Error(w, ActorHeader+" is required", http.StatusUnauthorized)
		return "", false
	}

	return *value, true
}

// OptionalActorValue returns a validated actor, or nil when the header is absent.
func OptionalActorValue(w http.ResponseWriter, r *http.Request) (actor *domain.Actor, ok bool) {
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
