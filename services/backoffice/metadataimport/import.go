// Package metadataimport coordinates metadata parsing and persistence.
package metadataimport

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
	domain "github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	importer "github.com/EPOS-ERIC/epos-platform/services/backoffice/turtle"
	"github.com/google/uuid"
)

var (
	ErrMetadataImportParse   = errors.New("metadata import parse failed")
	ErrMetadataImportPersist = errors.New("metadata import persistence failed")
)

type MetadataImporter struct {
	db     metadataRepository
	users  *identity.Users
	groups *identity.Groups
}

type metadataRepository interface {
	ImportBatch(context.Context, importer.Batch) error
}

func NewMetadataImporter(database metadataRepository, users *identity.Users, groups *identity.Groups) *MetadataImporter {
	return &MetadataImporter{db: database, users: users, groups: groups}
}

func (s *MetadataImporter) Import(ctx context.Context, input io.Reader, editorID, groupID uuid.UUID) (importer.Batch, error) {
	editor, err := s.users.GetByID(ctx, editorID)
	if err != nil {
		return importer.Batch{}, fmt.Errorf("get editor: %w", err)
	}
	group, err := s.groups.GetByID(ctx, groupID)
	if err != nil {
		return importer.Batch{}, fmt.Errorf("get group: %w", err)
	}

	batch, err := importer.Parse(input, importer.Options{
		Editor:  toDomainUser(editor),
		Group:   toDomainGroup(group),
		BaseURI: "",
	})
	if err != nil {
		return importer.Batch{}, fmt.Errorf("%w: %w", ErrMetadataImportParse, err)
	}
	if err := s.db.ImportBatch(ctx, batch); err != nil {
		return importer.Batch{}, fmt.Errorf("%w: %w", ErrMetadataImportPersist, err)
	}
	return batch, nil
}

func toDomainUser(user identity.User) domain.User {
	return domain.User{
		ID:             uuid.MustParse(user.ID),
		AuthIdentifier: user.AuthIdentifier,
		FamilyName:     stringValue(user.FamilyName),
		GivenName:      stringValue(user.GivenName),
		Email:          stringValue(user.Email),
		IsSuperAdmin:   user.IsSuperAdmin,
		Status:         domain.UserStatus(user.Status),
		Created:        user.CreatedAt,
		Updated:        user.UpdatedAt,
	}
}

func toDomainGroup(group identity.Group) domain.Group {
	return domain.Group{
		ID:          uuid.MustParse(group.ID),
		Slug:        group.Slug,
		Name:        group.Name,
		Description: stringValue(group.Description),
		IsSystem:    group.IsSystem,
		Created:     group.CreatedAt,
		Updated:     group.UpdatedAt,
	}
}

func stringValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
