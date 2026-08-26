// Package transport contains shared HTTP transport helpers.
package transport

import (
	"net/http"
	"strings"
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
func OptionalActor(w http.ResponseWriter, r *http.Request) (actorID string, present bool, valid bool) {
	value := strings.TrimSpace(r.Header.Get(ActorHeader))
	if value == "" {
		return "", false, true
	}
	return value, true, true
}
