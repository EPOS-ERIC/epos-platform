// Package catalogue coordinates converter records and local plugin installations.
package catalogue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/converter/domain"
	"github.com/EPOS-ERIC/epos-platform/services/converter/plugin"
	"github.com/google/uuid"
)

type Repository interface {
	CreatePlugin(context.Context, domain.PluginInput, string) (domain.Plugin, error)
	Plugin(context.Context, uuid.UUID) (domain.Plugin, error)
	Plugins(context.Context) ([]domain.Plugin, error)
	UpdatePlugin(context.Context, uuid.UUID, domain.PluginInput, bool, string) (domain.Plugin, error)
	DeletePlugin(context.Context, uuid.UUID) error
	CreateRelation(context.Context, domain.RelationInput, string) (domain.Relation, error)
	Relation(context.Context, uuid.UUID) (domain.Relation, error)
	Relations(context.Context) ([]domain.Relation, error)
	DistributionRelations(context.Context, uuid.UUID) ([]domain.PluginRelation, error)
	UpdateRelation(context.Context, uuid.UUID, domain.RelationInput, string) (domain.Relation, error)
	DeleteRelation(context.Context, uuid.UUID) error
}

func (s *Service) DistributionRelations(ctx context.Context, id uuid.UUID) ([]domain.PluginRelation, error) {
	return s.repo.DistributionRelations(ctx, id)
}

type Service struct {
	repo    Repository
	manager *plugin.Manager
	log     *slog.Logger
}

func New(repo Repository, manager *plugin.Manager, log *slog.Logger) *Service {
	return &Service{repo: repo, manager: manager, log: log}
}

func (s *Service) CreatePlugin(ctx context.Context, input domain.PluginInput, actor string) (domain.Plugin, error) {
	input, err := domain.NewPluginInput(input)
	if err != nil {
		return domain.Plugin{}, err
	}
	record, err := s.repo.CreatePlugin(ctx, input, actor)
	if err != nil {
		return domain.Plugin{}, err
	}
	if err := s.manager.Sync(ctx, definition(record), false); err != nil {
		s.log.ErrorContext(ctx, "initial plugin sync failed", "plugin_id", record.ID, "error", err)
	}
	return record, nil
}

func (s *Service) UpdatePlugin(ctx context.Context, id uuid.UUID, input domain.PluginInput, enabled bool, actor string) (domain.Plugin, error) {
	input, err := domain.NewPluginInput(input)
	if err != nil {
		return domain.Plugin{}, err
	}
	previous, err := s.repo.Plugin(ctx, id)
	if err != nil {
		return domain.Plugin{}, err
	}
	record, err := s.repo.UpdatePlugin(ctx, id, input, enabled, actor)
	if err != nil {
		return domain.Plugin{}, err
	}
	if previous.Repository.String() != input.Repository.String() || previous.Branch != input.Branch || previous.Runtime != input.Runtime {
		if err := s.manager.Sync(ctx, definition(record), true); err != nil {
			s.log.ErrorContext(ctx, "updated plugin sync failed", "plugin_id", id, "error", err)
		}
	}
	return record, nil
}

func (s *Service) DeletePlugin(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeletePlugin(ctx, id); err != nil {
		return err
	}
	return s.manager.Remove(id)
}

func (s *Service) SyncPlugin(ctx context.Context, id uuid.UUID) (domain.Plugin, error) {
	record, err := s.repo.Plugin(ctx, id)
	if err != nil {
		return domain.Plugin{}, err
	}
	if err := s.manager.Sync(ctx, definition(record), false); err != nil {
		return domain.Plugin{}, fmt.Errorf("sync plugin: %w", err)
	}
	return record, nil
}

func (s *Service) Plugin(ctx context.Context, id uuid.UUID) (domain.Plugin, error) {
	return s.repo.Plugin(ctx, id)
}

func (s *Service) Plugins(ctx context.Context) ([]domain.Plugin, error) {
	return s.repo.Plugins(ctx)
}

func (s *Service) Status(ctx context.Context, id uuid.UUID) (plugin.Status, error) {
	return s.manager.Status(ctx, id)
}

func (s *Service) CreateRelation(ctx context.Context, input domain.RelationInput, actor string) (domain.Relation, error) {
	input, err := domain.NewRelationInput(input)
	if err != nil {
		return domain.Relation{}, err
	}
	return s.repo.CreateRelation(ctx, input, actor)
}

func (s *Service) UpdateRelation(ctx context.Context, id uuid.UUID, input domain.RelationInput, actor string) (domain.Relation, error) {
	input, err := domain.NewRelationInput(input)
	if err != nil {
		return domain.Relation{}, err
	}
	return s.repo.UpdateRelation(ctx, id, input, actor)
}

func (s *Service) Relation(ctx context.Context, id uuid.UUID) (domain.Relation, error) {
	return s.repo.Relation(ctx, id)
}

func (s *Service) Relations(ctx context.Context) ([]domain.Relation, error) {
	return s.repo.Relations(ctx)
}

func (s *Service) DeleteRelation(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRelation(ctx, id)
}

func (s *Service) Executable(ctx context.Context, conversionID, distributionID uuid.UUID) (domain.Plugin, error) {
	relation, err := s.repo.Relation(ctx, conversionID)
	if err != nil {
		return domain.Plugin{}, err
	}
	if relation.DistributionID != distributionID {
		return domain.Plugin{}, apperror.Default(apperror.CodeNotFound)
	}
	record, err := s.repo.Plugin(ctx, relation.PluginID)
	if err != nil {
		return domain.Plugin{}, err
	}
	if !record.Enabled {
		return domain.Plugin{}, apperror.New(apperror.CodeConflict, "Plugin is disabled.")
	}
	return record, nil
}

func (s *Service) RunSync(ctx context.Context, interval time.Duration) {
	s.syncAll(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncAll(ctx)
		}
	}
}

func (s *Service) syncAll(ctx context.Context) {
	records, err := s.repo.Plugins(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "list plugins for synchronization", "error", err)
		return
	}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := s.manager.Sync(ctx, definition(record), false); err != nil {
			s.log.ErrorContext(ctx, "synchronize plugin", "plugin_id", record.ID, "error", err)
		}
	}
}

func definition(record domain.Plugin) plugin.Definition {
	return plugin.Definition{ID: record.ID, Repository: record.Repository, Branch: record.Branch, Runtime: record.Runtime, Arguments: record.Arguments}
}
