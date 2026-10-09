package configurations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/configuration"
	configurationsqlc "github.com/EPOS-ERIC/epos-platform/services/sharing/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	db      *pgxpool.Pool
	queries *configurationsqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{db: db, queries: configurationsqlc.New(db)}
}

func (r *Repo) Create(ctx context.Context, ownerID domain.Actor, input domain.ConfigurationInput) (domain.Configuration, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Configuration{}, fmt.Errorf("begin create configuration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	if err := queries.LockOwnerConfigurations(ctx, string(ownerID)); err != nil {
		return domain.Configuration{}, persistenceError("lock owner configurations", err)
	}

	count, err := queries.CountNonOfficialConfigurationsForOwner(ctx, string(ownerID))
	if err != nil {
		return domain.Configuration{}, persistenceError("count owner configurations", err)
	}

	if count >= domain.MaxNonOfficialConfigurationsPerOwner {
		return domain.Configuration{}, apperror.Default(apperror.CodeConflict)
	}

	services := input.ListOfServices
	if services == nil {
		services = []string{}
	}
	configuration, err := queries.CreateConfiguration(ctx, configurationsqlc.CreateConfigurationParams{
		Configuration:  input.Configuration,
		Title:          input.Title,
		Description:    input.Description,
		ListOfServices: services,
		IsPublic:       input.IsPublic != nil && *input.IsPublic,
		OwnerID:        string(ownerID),
	})
	if err != nil {
		return domain.Configuration{}, persistenceError("create configuration", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Configuration{}, fmt.Errorf("commit create configuration: %w", err)
	}

	return toDomainConfiguration(configuration.ID, configuration.Configuration, configuration.Title, configuration.Description, configuration.ListOfServices, configuration.IsOfficial, configuration.IsPublic, configuration.OwnerID, configuration.CreatedAt, configuration.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (domain.Configuration, error) {
	configuration, err := r.queries.GetConfigurationByID(ctx, id)
	if err != nil {
		return domain.Configuration{}, persistenceError("get configuration", err)
	}

	return toDomainConfiguration(configuration.ID, configuration.Configuration, configuration.Title, configuration.Description, configuration.ListOfServices, configuration.IsOfficial, configuration.IsPublic, configuration.OwnerID, configuration.CreatedAt, configuration.UpdatedAt)
}

func (r *Repo) List(ctx context.Context, filter configuration.ListFilter) ([]domain.ConfigurationSummary, error) {
	rows, err := r.queries.ListConfigurations(ctx, configurationsqlc.ListConfigurationsParams{
		Scope:    string(filter.Scope),
		CallerID: string(filter.Caller.Actor),
		OwnerID:  string(filter.OwnerID),
	})
	if err != nil {
		return nil, persistenceError("list configurations", err)
	}

	records := make([]domain.ConfigurationSummary, 0, len(rows))
	for _, row := range rows {
		record, err := domain.NewConfigurationSummary(row.ID, row.Title, &row.Description, row.ListOfServices, row.IsOfficial, row.IsPublic, row.OwnerID, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("list configurations: %w", err)
		}

		records = append(records, record)
	}

	return records, nil
}

func (r *Repo) Update(ctx context.Context, id uuid.UUID, caller domain.Caller, input domain.ConfigurationInput) (domain.Configuration, error) {
	services := input.ListOfServices
	if services == nil {
		services = []string{}
	}
	configuration, err := r.queries.UpdateConfiguration(ctx, configurationsqlc.UpdateConfigurationParams{
		ID:             id,
		Configuration:  input.Configuration,
		Title:          input.Title,
		Description:    input.Description,
		ListOfServices: services,
		IsPublic:       input.IsPublic,
		CallerID:       string(caller.Actor),
		CanManage:      caller.CanManage,
	})
	if err != nil {
		return domain.Configuration{}, persistenceError("update configuration", err)
	}
	return toDomainConfiguration(configuration.ID, configuration.Configuration, configuration.Title, configuration.Description, configuration.ListOfServices, configuration.IsOfficial, configuration.IsPublic, configuration.OwnerID, configuration.CreatedAt, configuration.UpdatedAt)
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID, caller domain.Caller) error {
	affected, err := r.queries.DeleteConfiguration(ctx, configurationsqlc.DeleteConfigurationParams{
		ID:        id,
		CallerID:  string(caller.Actor),
		CanManage: caller.CanManage,
	})
	if err != nil {
		return persistenceError("delete configuration", err)
	}
	if affected == 0 {
		return apperror.Default(apperror.CodeNotFound)
	}
	return nil
}

func (r *Repo) SetOfficialStatus(ctx context.Context, id uuid.UUID, caller domain.Caller, isOfficial bool) (domain.Configuration, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Configuration{}, fmt.Errorf("begin configuration status update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)

	ownerID, err := queries.GetConfigurationOwnerByID(ctx, id)
	if err != nil {
		return domain.Configuration{}, persistenceError("get configuration owner", err)
	}

	if err := queries.LockOwnerConfigurations(ctx, ownerID); err != nil {
		return domain.Configuration{}, persistenceError("lock owner configurations", err)
	}

	configuration, err := queries.SetConfigurationOfficialStatus(ctx, configurationsqlc.SetConfigurationOfficialStatusParams{
		IsOfficial: isOfficial,
		ID:         id,
		CanManage:  caller.CanManage,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := queries.GetConfigurationByID(ctx, id)
			if getErr != nil {
				return domain.Configuration{}, persistenceError("get configuration", getErr)
			}

			if existing.IsOfficial == isOfficial {
				configuration = configurationsqlc.SetConfigurationOfficialStatusRow(existing)
			} else {
				return domain.Configuration{}, persistenceError("set configuration official status", err)
			}
		} else {
			return domain.Configuration{}, persistenceError("set configuration official status", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Configuration{}, fmt.Errorf("commit configuration status update: %w", err)
	}

	return toDomainConfiguration(configuration.ID, configuration.Configuration, configuration.Title, configuration.Description, configuration.ListOfServices, configuration.IsOfficial, configuration.IsPublic, configuration.OwnerID, configuration.CreatedAt, configuration.UpdatedAt)
}

func toDomainConfiguration(id uuid.UUID, rawConfiguration []byte, title, description string, services []string, isOfficial, isPublic bool, ownerID string, createdAt, updatedAt time.Time) (domain.Configuration, error) {
	public := isPublic
	return domain.NewConfiguration(id, domain.ConfigurationInput{
		Configuration:  rawConfiguration,
		Title:          title,
		Description:    &description,
		ListOfServices: services,
		IsPublic:       &public,
	}, isOfficial, ownerID, createdAt, updatedAt)
}

func persistenceError(operation string, err error) error {
	code := apperror.CodeInternal
	if errors.Is(err, pgx.ErrNoRows) {
		code = apperror.CodeNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		code = apperror.CodeConflict
	}
	return fmt.Errorf("%s: %w", operation, apperror.Wrap(code, "", err))
}
