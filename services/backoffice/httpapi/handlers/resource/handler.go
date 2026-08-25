// Package resource dispatches shared resource routes to typed lifecycle handlers.
package resource

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	"github.com/go-chi/chi/v5"
)

const (
	Dataset             = "dataset"
	SoftwareApplication = "software-application"
	SoftwareSourceCode  = "software-source-code"
)

type typedHandler interface {
	Create(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	ReplaceDraft(http.ResponseWriter, *http.Request)
	Submit(http.ResponseWriter, *http.Request)
	Withdraw(http.ResponseWriter, *http.Request)
	Reject(http.ResponseWriter, *http.Request)
	Reopen(http.ResponseWriter, *http.Request)
	Publish(http.ResponseWriter, *http.Request)
	Unpublish(http.ResponseWriter, *http.Request)
	DraftFromPublication(http.ResponseWriter, *http.Request)
	DraftFromArchive(http.ResponseWriter, *http.Request)
	Discard(http.ResponseWriter, *http.Request)
	Copy(http.ResponseWriter, *http.Request)
}

type Handler struct {
	dataset             typedHandler
	softwareApplication typedHandler
	softwareSourceCode  typedHandler
}

// New creates a resource-type dispatch handler.
func New(dataset, softwareApplication, softwareSourceCode typedHandler) *Handler {
	return &Handler{
		dataset:             dataset,
		softwareApplication: softwareApplication,
		softwareSourceCode:  softwareSourceCode,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Create(w, r) })
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Get(w, r) })
}

func (h *Handler) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.ReplaceDraft(w, r) })
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Submit(w, r) })
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Withdraw(w, r) })
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Reject(w, r) })
}

func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Reopen(w, r) })
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Publish(w, r) })
}

func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Unpublish(w, r) })
}

func (h *Handler) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.DraftFromPublication(w, r) })
}

func (h *Handler) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.DraftFromArchive(w, r) })
}

func (h *Handler) Discard(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Discard(w, r) })
}

func (h *Handler) Copy(w http.ResponseWriter, r *http.Request) {
	h.dispatch(w, r, func(handler typedHandler) { handler.Copy(w, r) })
}

func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request, call func(typedHandler)) {
	resourceType := chi.URLParam(r, "resource_type")
	if resourceType == "" {
		// TODO: use a helper for stuff like this
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
		if err != nil {
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid JSON request")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		var envelope struct {
			ResourceType string `json:"resource_type"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid JSON request")
			return
		}
		resourceType = envelope.ResourceType
	}

	var handler typedHandler
	switch resourceType {
	case Dataset:
		handler = h.dataset
	case SoftwareApplication:
		handler = h.softwareApplication
	case SoftwareSourceCode:
		handler = h.softwareSourceCode
	default:
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid resource_type")
		return
	}
	call(handler)
}
