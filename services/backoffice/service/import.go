package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/domain"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/importer"
)

var (
	ErrMetadataImportParse   = errors.New("metadata import parse failed")
	ErrMetadataImportPersist = errors.New("metadata import persistence failed")
)

type MetadataImporter struct {
	db *db.PostgresDB
}

func NewMetadataImporter(database *db.PostgresDB) *MetadataImporter {
	return &MetadataImporter{db: database}
}

func (s *MetadataImporter) Import(ctx context.Context, input io.Reader, editor domain.User, group domain.Group) (importer.Batch, error) {
	batch, err := importer.Parse(input, importer.Options{Editor: editor, Group: group})
	if err != nil {
		return importer.Batch{}, fmt.Errorf("%w: %v", ErrMetadataImportParse, err)
	}
	if err := s.db.ImportBatch(ctx, batch); err != nil {
		return importer.Batch{}, fmt.Errorf("%w: %v", ErrMetadataImportPersist, err)
	}
	return batch, nil
}
