// Package software provides PostgreSQL persistence for software lifecycle data.
package software

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/softwarelifecycle"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	ApplicationRepo struct{ db *pgxpool.Pool }
	SourceCodeRepo  struct{ db *pgxpool.Pool }
)

// NewApplicationRepo creates a software application repository.
func NewApplicationRepo(pool *pgxpool.Pool) *ApplicationRepo {
	return &ApplicationRepo{db: pool}
}

// NewSourceCodeRepo creates a software source code repository.
func NewSourceCodeRepo(pool *pgxpool.Pool) *SourceCodeRepo {
	return &SourceCodeRepo{db: pool}
}

func withTx(ctx context.Context, db *pgxpool.Pool, fn func(*sqlc.Queries) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(sqlc.New(tx)); err != nil {
		return fmt.Errorf("execute transaction: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func lifecycleError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return softwarelifecycle.ErrConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (strings.HasPrefix(pgErr.Code, "23") || pgErr.Code == "P0001") {
		return fmt.Errorf("database invariant %q: %w", pgErr.ConstraintName, softwarelifecycle.ErrConflict)
	}
	return err
}

func organization(value sqlc.Organization) metadata.Organization {
	return metadata.Organization{
		ID:            value.ID,
		SourceIRI:     value.SourceIri,
		Name:          value.Name,
		Street:        value.Street,
		Locality:      value.Locality,
		PostalCode:    value.PostalCode,
		Country:       value.Country,
		LogoURL:       value.LogoUrl,
		PageURL:       value.PageUrl,
		Email:         value.Email,
		LEICode:       value.LeiCode,
		ContactPoints: nil,
		Identifiers:   nil,
	}
}

func contactPoint(value sqlc.ContactPoint) metadata.ContactPoint {
	return metadata.ContactPoint{ID: value.ID, SourceIRI: value.SourceIri, Email: value.Email, Role: metadata.ContactPointRole(value.Role)}
}

func organizationWithContactPoints(ctx context.Context, q *sqlc.Queries, value sqlc.Organization) (metadata.Organization, error) {
	result := organization(value)
	rows, err := q.ListLifecycleOrganizationContactPoints(ctx, []uuid.UUID{value.ID})
	if err != nil {
		return metadata.Organization{}, fmt.Errorf("list organization contact points: %w", err)
	}
	for _, row := range rows {
		result.ContactPoints = append(result.ContactPoints, contactPoint(row.ContactPoint))
	}
	return result, nil
}

func identifier(value sqlc.Identifier) metadata.Identifier {
	return metadata.Identifier{ID: value.ID, SourceIRI: value.SourceIri, Type: metadata.IdentifierType(value.Type), Value: value.Value}
}

func category(id uuid.UUID, sourceIRI, name string, group sqlc.Group) metadata.Category {
	return metadata.Category{ID: id, SourceIRI: sourceIRI, Name: name, Scheme: categoryScheme(group), Children: nil}
}

func categoryScheme(value sqlc.Group) metadata.CategoryScheme {
	return metadata.CategoryScheme{
		ID: value.ID, SourceIRI: value.SourceIri, Name: value.Name, Label: value.Label,
		Logo: value.Logo, Page: value.Page, Color: value.Color, Position: value.Position,
		Concept: metadata.TopLevelConcept(value.Concept),
	}
}
