// Package groups contains Gateway group metadata use-cases.
package groups

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	backoffice *clients.Backoffice
}

func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{backoffice: backoffice}
}

func (s *Service) List(ctx context.Context) ([]domain.Group, error) {
	if s == nil || s.backoffice == nil {
		return nil, fmt.Errorf("backoffice group client is not configured")
	}

	groups, err := s.backoffice.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups from backoffice: %w", err)
	}

	return groups, nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Group, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Group{}, fmt.Errorf("invalid group id: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.Group{}, fmt.Errorf("backoffice group client is not configured")
	}

	group, err := s.backoffice.GetGroup(ctx, id)
	if err != nil {
		return domain.Group{}, fmt.Errorf("get group from backoffice: %w", err)
	}

	return group, nil
}

func (s *Service) WithdrawMembershipRequest(ctx context.Context, groupID, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || actorID == "" {
		return fmt.Errorf("invalid membership request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.WithdrawMembershipRequest(ctx, groupID, actorID); err != nil {
		return fmt.Errorf("withdraw membership request from backoffice: %w", err)
	}

	return nil
}

func (s *Service) DecideMembership(ctx context.Context, groupID, userID, actorID, status string) (domain.GroupMembership, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	actorID = strings.TrimSpace(actorID)
	status = strings.TrimSpace(status)
	if groupID == "" || userID == "" || actorID == "" || (status != "ACTIVE" && status != "REJECTED") {
		return domain.GroupMembership{}, fmt.Errorf("invalid membership decision: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.GroupMembership{}, fmt.Errorf("backoffice group client is not configured")
	}

	membership, err := s.backoffice.DecideMembership(ctx, groupID, userID, status, actorID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide membership in backoffice: %w", err)
	}

	return membership, nil
}
