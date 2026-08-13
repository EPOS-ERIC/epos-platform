// Package transport contains shared HTTP transport helpers.
package transport

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const ActorHeader = "X-EPOS-User-ID"

func Actor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
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

func OptionalActor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool, bool) {
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
