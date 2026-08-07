package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/domain"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/service"
	"github.com/google/uuid"
)

type MetadataImport struct {
	importer *service.MetadataImporter
	users    *service.Users
	groups   *service.Groups
	log      *slog.Logger
}

func NewMetadataImport(importer *service.MetadataImporter, users *service.Users, groups *service.Groups) *MetadataImport {
	return &MetadataImport{importer: importer, users: users, groups: groups, log: logging.Get("metadata-import-handler")}
}

type metadataImportResponse struct {
	Datasets        int               `json:"datasets"`
	CategorySchemes int               `json:"categorySchemes"`
	Categories      int               `json:"categories"`
	Organizations   int               `json:"organizations"`
	ContactPoints   int               `json:"contactPoints"`
	Warnings        []importerWarning `json:"warnings,omitempty"`
}

type importerWarning struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (h *MetadataImport) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeMessage(w, http.StatusBadRequest, "request must be multipart form data")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, "file is required")
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			h.log.WarnContext(r.Context(), "close uploaded metadata file failed", "error", closeErr)
		}
	}()

	editorID, err := parseFormUUID(r, "editorId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	groupID, err := parseFormUUID(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.users.GetByID(r.Context(), editorID)
	if err != nil {
		writeServiceError(w, err, "editor")
		return
	}
	group, err := h.groups.GetByID(r.Context(), groupID)
	if err != nil {
		writeServiceError(w, err, "group")
		return
	}

	batch, err := h.importer.Import(r.Context(), file, domain.User{
		ID:             editorID,
		AuthIdentifier: user.AuthIdentifier,
		FamilyName:     stringPtr(user.FamilyName),
		GivenName:      stringPtr(user.GivenName),
		Email:          stringPtr(user.Email),
		IsSuperAdmin:   user.IsSuperAdmin,
		Status:         domain.UserStatus(user.Status),
		Created:        user.CreatedAt,
		Updated:        user.UpdatedAt,
	}, domain.Group{
		ID:          groupID,
		Slug:        group.Slug,
		Name:        group.Name,
		Description: stringPtr(group.Description),
		IsSystem:    group.IsSystem,
		Created:     group.CreatedAt,
		Updated:     group.UpdatedAt,
	})
	if err != nil {
		h.log.ErrorContext(r.Context(), "metadata import failed", "error", err)
		if errors.Is(err, service.ErrMetadataImportPersist) {
			writeMessage(w, http.StatusInternalServerError, "metadata import persistence failed")
		} else {
			writeMessage(w, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}

	warnings := make([]importerWarning, 0, len(batch.Warnings))
	for _, warning := range batch.Warnings {
		warnings = append(warnings, importerWarning{Subject: warning.Subject, Message: warning.Message})
	}
	writeJSON(w, http.StatusCreated, metadataImportResponse{
		Datasets: len(batch.Datasets), CategorySchemes: len(batch.CategorySchemes), Categories: len(batch.Categories),
		Organizations: len(batch.Organizations), ContactPoints: len(batch.ContactPoints), Warnings: warnings,
	})
}

func parseFormUUID(r *http.Request, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(r.FormValue(name))
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", name)
	}
	return id, nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
