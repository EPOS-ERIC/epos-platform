package handlers

import (
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataread"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MetadataRead struct {
	reader *metadataread.Reader
}

func NewMetadataRead(reader *metadataread.Reader) *MetadataRead {
	return &MetadataRead{reader: reader}
}

func (h *MetadataRead) Get(w http.ResponseWriter, r *http.Request) {
	resourceType, err := metadataread.ParseResourceType(chi.URLParam(r, "resourceType"))
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeMessage(w, http.StatusBadRequest, "id must be a UUID")
		return
	}

	resource, err := h.reader.Get(r.Context(), resourceType, id)
	if errors.Is(err, metadataread.ErrNotFound) {
		writeMessage(w, http.StatusNotFound, string(resourceType)+" not found")
		return
	}

	if err != nil {
		log.ErrorContext(r.Context(), "read metadata resource failed", "error", err)
		writeMessage(w, http.StatusInternalServerError, "metadata resource read failed")
		return
	}

	switch resourceType {
	case metadataread.ResourceTypeDataset:
		writeJSON(w, http.StatusOK, resource.Dataset)
	case metadataread.ResourceTypeDistribution:
		writeJSON(w, http.StatusOK, resource.Distribution)
	case metadataread.ResourceTypeWebService:
		writeJSON(w, http.StatusOK, resource.WebService)
	case metadataread.ResourceTypeOrganization:
		writeJSON(w, http.StatusOK, resource.Organization)
	case metadataread.ResourceTypeConcept:
		writeJSON(w, http.StatusOK, resource.Concept)
	case metadataread.ResourceTypeConceptScheme:
		writeJSON(w, http.StatusOK, resource.ConceptScheme)
	case metadataread.ResourceTypeSoftwareApplication:
		writeJSON(w, http.StatusOK, resource.SoftwareApplication)
	case metadataread.ResourceTypeSoftwareSourceCode:
		writeJSON(w, http.StatusOK, resource.SoftwareSourceCode)
	}
}
