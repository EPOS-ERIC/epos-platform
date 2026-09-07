package metadataimport

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRecoverableImportError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "invalid import", err: domain.ErrInvalidImport, want: true},
		{name: "import conflict", err: domain.ErrImportConflict, want: true},
		{name: "unique violation", err: &pgconn.PgError{Code: "23505"}, want: true},
		{name: "raise exception", err: &pgconn.PgError{Code: "P0001"}, want: true},
		{name: "serialization failure", err: &pgconn.PgError{Code: "40001"}, want: false},
		{name: "other error", err: errors.New("database unavailable"), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := recoverableImportError(fmt.Errorf("wrapped: %w", test.err)); got != test.want {
				t.Fatalf("recoverableImportError() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSavepointNameIsDeterministicAndSafe(t *testing.T) {
	first := savepointName("dataset", "dataset:one")
	if first != savepointName("dataset", "dataset:one") {
		t.Fatal("savepointName() is not deterministic")
	}
	if !strings.HasPrefix(first, "import_") || strings.Contains(first, "-") {
		t.Fatalf("savepointName() = %q, want SQL-safe import name", first)
	}
}

func TestRemapDatasetReferencesDeduplicatesCanonicalProviders(t *testing.T) {
	firstSource := uuid.New()
	secondSource := uuid.New()
	canonical := uuid.New()
	value := domain.DatasetInput{ProviderIDs: []uuid.UUID{firstSource, secondSource}}

	remapDatasetReferences(&value, map[uuid.UUID]uuid.UUID{
		firstSource:  canonical,
		secondSource: canonical,
	}, nil, nil)

	if len(value.ProviderIDs) != 1 || value.ProviderIDs[0] != canonical {
		t.Fatalf("provider IDs = %v, want [%s]", value.ProviderIDs, canonical)
	}
}
