// Package transport contains shared HTTP transport helpers.
package transport

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const ActorHeader = "X-EPOS-User-ID"

// Actor returns actor ID and whether request contains a valid actor header.
func Actor(w http.ResponseWriter, r *http.Request) (actorID uuid.UUID, valid bool) {
	id, present, ok := OptionalActor(w, r)
	if !ok {
		return uuid.Nil, false
	}
	if !present {
		http.Error(w, ActorHeader+" is required", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

// OptionalActor returns actor ID, whether header is present, and whether value is valid.
func OptionalActor(w http.ResponseWriter, r *http.Request) (actorID uuid.UUID, present bool, valid bool) {
	value := strings.TrimSpace(r.Header.Get(ActorHeader))
	if value == "" {
		return uuid.Nil, false, true
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		http.Error(w, "invalid "+ActorHeader, http.StatusUnauthorized)
		return uuid.Nil, true, false
	}
	return id, true, true
}
