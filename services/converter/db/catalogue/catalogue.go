// Package catalogue stores converter plugins and relations in PostgreSQL.
package catalogue

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/converter/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/converter/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *sqlc.Queries }

func New(pool *pgxpool.Pool) *Repository { return &Repository{queries: sqlc.New(pool)} }

func (r *Repository) CreatePlugin(ctx context.Context, input domain.PluginInput, actor string) (domain.Plugin, error) {
	p, err := r.queries.CreatePlugin(ctx, sqlc.CreatePluginParams{
		Name: input.Name, Description: input.Description, Repository: input.Repository.String(),
		Branch: input.Branch, Runtime: string(input.Runtime), Arguments: input.Arguments, CreatedBy: actor,
	})
	if err != nil {
		return domain.Plugin{}, persistenceError("create plugin", err)
	}
	return fromPlugin(p)
}

func (r *Repository) Plugin(ctx context.Context, id uuid.UUID) (domain.Plugin, error) {
	p, err := r.queries.GetPlugin(ctx, id)
	if err != nil {
		return domain.Plugin{}, persistenceError("get plugin", err)
	}
	return fromPlugin(p)
}

func (r *Repository) Plugins(ctx context.Context) ([]domain.Plugin, error) {
	rows, err := r.queries.ListPlugins(ctx)
	if err != nil {
		return nil, persistenceError("list plugins", err)
	}
	result := make([]domain.Plugin, 0, len(rows))
	for _, row := range rows {
		p, err := fromPlugin(row)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *Repository) UpdatePlugin(ctx context.Context, id uuid.UUID, input domain.PluginInput, enabled bool, actor string) (domain.Plugin, error) {
	p, err := r.queries.UpdatePlugin(ctx, sqlc.UpdatePluginParams{
		ID: id, Name: input.Name, Description: input.Description, Repository: input.Repository.String(),
		Branch: input.Branch, Runtime: string(input.Runtime), Arguments: input.Arguments, Enabled: enabled, UpdatedBy: actor,
	})
	if err != nil {
		return domain.Plugin{}, persistenceError("update plugin", err)
	}
	return fromPlugin(p)
}

func (r *Repository) DeletePlugin(ctx context.Context, id uuid.UUID) error {
	count, err := r.queries.DeletePlugin(ctx, id)
	return rowsAffectedError("delete plugin", count, err)
}

func (r *Repository) Installation(ctx context.Context, id uuid.UUID) (*domain.Installation, error) {
	row, err := r.queries.GetPluginInstallation(ctx, id)
	if err != nil {
		return nil, persistenceError("get plugin installation", err)
	}
	if row.InstalledCommit == nil {
		return nil, nil
	}
	return &domain.Installation{
		Commit: *row.InstalledCommit, Runtime: domain.Runtime(row.Runtime),
	}, nil
}

func (r *Repository) SetInstallation(ctx context.Context, id uuid.UUID, commit *string) error {
	count, err := r.queries.SetPluginInstallation(ctx, sqlc.SetPluginInstallationParams{
		ID: id, InstalledCommit: commit,
	})
	return rowsAffectedError("set plugin installation", count, err)
}

func (r *Repository) CreateRelation(ctx context.Context, input domain.RelationInput, actor string) (domain.Relation, error) {
	relation, err := r.queries.CreateRelation(ctx, sqlc.CreateRelationParams{
		DistributionID: input.DistributionID,
		PluginID:       input.PluginID,
		InputFormat:    input.InputFormat,
		OutputFormat:   input.OutputFormat,
		CreatedBy:      actor,
	})
	return fromRelation(relation), persistenceError("create relation", err)
}

func (r *Repository) Relation(ctx context.Context, id uuid.UUID) (domain.Relation, error) {
	relation, err := r.queries.GetRelation(ctx, id)
	return fromRelation(relation), persistenceError("get relation", err)
}

func (r *Repository) Relations(ctx context.Context, distributionID *uuid.UUID) ([]domain.Relation, error) {
	var rows []sqlc.ConversionRelation
	var err error
	if distributionID == nil {
		rows, err = r.queries.ListRelations(ctx)
	} else {
		rows, err = r.queries.ListRelationsByDistribution(ctx, *distributionID)
	}
	if err != nil {
		return nil, persistenceError("list relations", err)
	}
	result := make([]domain.Relation, 0, len(rows))
	for _, row := range rows {
		result = append(result, fromRelation(row))
	}
	return result, nil
}

func (r *Repository) UpdateRelation(ctx context.Context, id uuid.UUID, input domain.RelationInput, actor string) (domain.Relation, error) {
	relation, err := r.queries.UpdateRelation(ctx, sqlc.UpdateRelationParams{
		ID:             id,
		DistributionID: input.DistributionID,
		PluginID:       input.PluginID,
		InputFormat:    input.InputFormat,
		OutputFormat:   input.OutputFormat,
		UpdatedBy:      actor,
	})
	return fromRelation(relation), persistenceError("update relation", err)
}

func (r *Repository) DeleteRelation(ctx context.Context, id uuid.UUID) error {
	count, err := r.queries.DeleteRelation(ctx, id)
	return rowsAffectedError("delete relation", count, err)
}

func fromPlugin(p sqlc.Plugin) (domain.Plugin, error) {
	repository, err := url.Parse(p.Repository)
	if err != nil {
		return domain.Plugin{}, fmt.Errorf("invalid stored plugin repository URL")
	}
	return domain.Plugin{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Repository:  *repository,
		Branch:      p.Branch,
		Runtime:     domain.Runtime(p.Runtime),
		Arguments:   p.Arguments,
		Enabled:     p.Enabled,
		CreatedBy:   p.CreatedBy,
		UpdatedBy:   p.UpdatedBy,
	}, nil
}

func fromRelation(r sqlc.ConversionRelation) domain.Relation {
	return domain.Relation{
		ID:             r.ID,
		DistributionID: r.DistributionID,
		PluginID:       r.PluginID,
		InputFormat:    r.InputFormat,
		OutputFormat:   r.OutputFormat,
		CreatedBy:      r.CreatedBy,
		UpdatedBy:      r.UpdatedBy,
	}
}

func rowsAffectedError(operation string, count int64, err error) error {
	if err != nil {
		return persistenceError(operation, err)
	}
	if count == 0 {
		return apperror.Default(apperror.CodeNotFound)
	}
	return nil
}

func persistenceError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Wrap(apperror.CodeNotFound, "", err)
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23505":
			return apperror.Wrap(apperror.CodeConflict, "Conversion relation already exists.", err)
		case "23503":
			return apperror.Wrap(apperror.CodeNotFound, "Plugin not found.", err)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}
