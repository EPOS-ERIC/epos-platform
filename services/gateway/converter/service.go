// Package converter manages conversion relations and their lifecycle propagation.
package converter

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type client interface {
	CreatePlugin(context.Context, string, domain.PluginInput) (domain.Plugin, error)
	GetPlugin(context.Context, string, uuid.UUID) (domain.Plugin, error)
	ListPlugins(context.Context, string) ([]domain.Plugin, error)
	UpdatePlugin(context.Context, string, uuid.UUID, domain.PluginInput, bool) (domain.Plugin, error)
	DeletePlugin(context.Context, string, uuid.UUID) error
	SyncPlugin(context.Context, string, uuid.UUID) (domain.Plugin, error)
	ListRelations(context.Context, string) ([]domain.Relation, error)
	CreateRelation(context.Context, string, domain.Relation) (domain.Relation, error)
	GetRelation(context.Context, string, uuid.UUID) (domain.Relation, error)
	UpdateRelation(context.Context, string, uuid.UUID, domain.Relation) (domain.Relation, error)
	DeleteRelation(context.Context, string, uuid.UUID) error
}

type metadataLifecycle interface {
	DistributionFormats(context.Context, string, uuid.UUID) ([]string, error)
}

type Service struct {
	client   client
	metadata metadataLifecycle
}

func New(client client, metadata metadataLifecycle) *Service {
	return &Service{client: client, metadata: metadata}
}

func manager(user domain.User, authenticated bool) (string, error) {
	if !authenticated || user.ID == "" {
		return "", apperror.Default(apperror.CodeUnauthenticated)
	}

	if !slices.Contains(user.GlobalGrantedPermissions, domain.GlobalPermissionConverterManage) &&
		!slices.Contains(user.GlobalRolePermissions, domain.GlobalPermissionConverterManage) {
		return "", apperror.Default(apperror.CodePermissionDenied)
	}

	return user.ID, nil
}

func (s *Service) CreatePlugin(ctx context.Context, user domain.User, authenticated bool, input domain.PluginInput) (domain.Plugin, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return domain.Plugin{}, err
	}

	return s.client.CreatePlugin(ctx, actor, input)
}

func (s *Service) GetPlugin(ctx context.Context, user domain.User, authenticated bool, id uuid.UUID) (domain.Plugin, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return domain.Plugin{}, err
	}

	return s.client.GetPlugin(ctx, actor, id)
}

func (s *Service) ListPlugins(ctx context.Context, user domain.User, authenticated bool) ([]domain.Plugin, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return nil, err
	}

	return s.client.ListPlugins(ctx, actor)
}

func (s *Service) UpdatePlugin(ctx context.Context, user domain.User, authenticated bool, id uuid.UUID, input domain.PluginInput, enabled bool) (domain.Plugin, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return domain.Plugin{}, err
	}

	return s.client.UpdatePlugin(ctx, actor, id, input, enabled)
}

func (s *Service) DeletePlugin(ctx context.Context, user domain.User, authenticated bool, id uuid.UUID) error {
	actor, err := manager(user, authenticated)
	if err != nil {
		return err
	}

	return s.client.DeletePlugin(ctx, actor, id)
}

func (s *Service) SyncPlugin(ctx context.Context, user domain.User, authenticated bool, id uuid.UUID) (domain.Plugin, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return domain.Plugin{}, err
	}

	return s.client.SyncPlugin(ctx, actor, id)
}

func (s *Service) AvailableConversions(ctx context.Context) ([]domain.AvailableConversion, error) {
	plugins, err := s.client.ListPlugins(ctx, "")
	if err != nil {
		return nil, err
	}

	relations, err := s.client.ListRelations(ctx, "")
	if err != nil {
		return nil, err
	}

	available := make(map[uuid.UUID]domain.Plugin, len(plugins))
	for _, plugin := range plugins {
		if plugin.Enabled && plugin.Installed {
			available[plugin.ID] = plugin
		}
	}

	result := make([]domain.AvailableConversion, 0, len(relations))
	for _, relation := range relations {
		plugin, ok := available[relation.PluginID]
		if !ok {
			continue
		}

		result = append(result, domain.AvailableConversion{
			ID:                relation.ID,
			DistributionID:    relation.DistributionID,
			InputFormat:       relation.InputFormat,
			OutputFormat:      relation.OutputFormat,
			PluginID:          plugin.ID,
			PluginName:        plugin.Name,
			PluginDescription: plugin.Description,
		})
	}

	return result, nil
}

func (s *Service) ListRelations(ctx context.Context, user domain.User, authenticated bool) ([]domain.Relation, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return nil, err
	}

	return s.client.ListRelations(ctx, actor)
}

func (s *Service) CreateRelation(ctx context.Context, user domain.User, authenticated bool, relation domain.Relation) (domain.Relation, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return domain.Relation{}, err
	}

	if err := s.validateRelation(ctx, actor, relation); err != nil {
		return domain.Relation{}, err
	}

	return s.client.CreateRelation(ctx, actor, relation)
}

func (s *Service) GetRelation(ctx context.Context, user domain.User, authenticated bool, id uuid.UUID) (domain.Relation, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return domain.Relation{}, err
	}

	return s.client.GetRelation(ctx, actor, id)
}

func (s *Service) UpdateRelation(ctx context.Context, user domain.User, authenticated bool, id uuid.UUID, relation domain.Relation) (domain.Relation, error) {
	actor, err := manager(user, authenticated)
	if err != nil {
		return domain.Relation{}, err
	}

	if err := s.validateRelation(ctx, actor, relation); err != nil {
		return domain.Relation{}, err
	}

	return s.client.UpdateRelation(ctx, actor, id, relation)
}

func (s *Service) DeleteRelation(ctx context.Context, user domain.User, authenticated bool, id uuid.UUID) error {
	actor, err := manager(user, authenticated)
	if err != nil {
		return err
	}

	return s.client.DeleteRelation(ctx, actor, id)
}

func (s *Service) validateRelation(ctx context.Context, actor string, relation domain.Relation) error {
	formats, err := s.metadata.DistributionFormats(ctx, actor, relation.DistributionID)
	if err != nil {
		return err
	}

	if !slices.Contains(formats, relation.InputFormat) {
		return apperror.Field("input_format", "Input format must match a native format of the distribution.", nil)
	}

	return nil
}

func (s *Service) CopyRelations(ctx context.Context, actor string, pairs map[uuid.UUID]uuid.UUID) error {
	if len(pairs) == 0 {
		return nil
	}

	relations, err := s.client.ListRelations(ctx, actor)
	if err != nil {
		return fmt.Errorf("list relations: %w", err)
	}

	for sourceID, targetID := range pairs {
		for _, relation := range relations {
			if relation.DistributionID != sourceID {
				continue
			}

			relation.DistributionID = targetID
			if slices.ContainsFunc(relations, func(existing domain.Relation) bool { return existing.Equivalent(relation) }) {
				continue
			}

			if _, err := s.client.CreateRelation(ctx, actor, relation); err != nil {
				if apperror.IsCode(err, apperror.CodeConflict) {
					current, listErr := s.client.ListRelations(ctx, actor)
					if listErr != nil {
						return fmt.Errorf("verify conflicting relation for distribution %s: %w", targetID, errors.Join(err, listErr))
					}

					if slices.ContainsFunc(current, func(existing domain.Relation) bool { return existing.Equivalent(relation) }) {
						continue
					}
				}

				return fmt.Errorf("copy relation %s from distribution %s to %s: %w", relation.ID, sourceID, targetID, err)
			}
		}
	}

	return nil
}

func (s *Service) RemoveRelations(ctx context.Context, actor string, removed []uuid.UUID) error {
	if len(removed) == 0 {
		return nil
	}

	relations, err := s.client.ListRelations(ctx, actor)
	if err != nil {
		return fmt.Errorf("list relations: %w", err)
	}

	for _, id := range removed {
		for _, relation := range relations {
			if relation.DistributionID != id {
				continue
			}

			if err := s.client.DeleteRelation(ctx, actor, relation.ID); err != nil {
				return fmt.Errorf("delete relation %s for distribution %s: %w", relation.ID, id, err)
			}
		}
	}

	return nil
}
