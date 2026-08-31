// Package users provides Metadata Lifecycle user operations.
package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

// TODO: see if we should move this interface to the consuming package instead of the implementing one (also for the other services)

type Store interface {
	GetUserByID(context.Context, string) (domain.User, error)
	ListUsers(context.Context) ([]domain.UserSummary, error)
	CreateUser(context.Context, domain.UserClaims) (domain.User, error)
	GrantGlobalPermission(context.Context, string, string, string) error
	RevokeGlobalPermission(context.Context, string, string) error
	AssignGlobalRole(context.Context, string, string, string) error
	RemoveGlobalRole(context.Context, string, string) error
	BlockUser(context.Context, string, string, string) error
	UnblockUser(context.Context, string) error
}

type Authorizer interface {
	CanReadUser(context.Context, string, string) (domain.Result, error)
	CanListUsers(context.Context, string) (domain.Result, error)
	AuthorizeAdmin(context.Context, string) (domain.Result, error)
	AuthorizeGlobal(context.Context, string, string) (domain.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	log        *slog.Logger
}

func New(log *slog.Logger, store Store, authorizer Authorizer) *Service {
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "users-service"),
	}
}

func (s *Service) GetByID(ctx context.Context, actorID domain.Actor, userID string) (domain.User, error) {
	// Resolve self before authorization so callers can distinguish a missing
	// authenticated user and provision it. Keep this limited to self-reads to
	// avoid exposing whether other user IDs exist to unauthorized callers.
	var self domain.User
	var err error
	if string(actorID) == userID {
		self, err = s.store.GetUserByID(ctx, userID)
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}
		if err != nil {
			return domain.User{}, fmt.Errorf("get user by id: %w", err)
		}
	}

	allowed, err := s.authorizer.CanReadUser(ctx, string(actorID), userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("authorize user read: %w", err)
	}

	if !allowed.Allowed {
		return domain.User{}, domain.ErrUserForbidden
	}

	if string(actorID) == userID {
		return self, nil
	}

	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (s *Service) List(ctx context.Context, actorID domain.Actor) ([]domain.UserSummary, error) {
	allowed, err := s.authorizer.CanListUsers(ctx, string(actorID))
	if err != nil {
		return nil, fmt.Errorf("authorize user list: %w", err)
	}

	if !allowed.Allowed {
		return nil, domain.ErrUserForbidden
	}

	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

func (s *Service) GrantGlobalPermission(ctx context.Context, actorID domain.Actor, userID, permission string) error {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), "global.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize global permission grant: %w", err)
	}

	if !allowed.Allowed {
		return domain.ErrUserForbidden
	}

	err = s.store.GrantGlobalPermission(ctx, userID, permission, string(actorID))
	if err != nil {
		return fmt.Errorf("grant global permission: %w", err)
	}

	return nil
}

func (s *Service) RevokeGlobalPermission(ctx context.Context, actorID domain.Actor, userID, permission string) error {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), "global.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize global permission revoke: %w", err)
	}

	if !allowed.Allowed {
		return domain.ErrUserForbidden
	}

	err = s.store.RevokeGlobalPermission(ctx, userID, permission)
	if err != nil {
		return fmt.Errorf("revoke global permission: %w", err)
	}

	return nil
}

func (s *Service) AssignGlobalRole(ctx context.Context, actorID domain.Actor, userID, role string) error {
	allowed, err := s.authorizer.AuthorizeAdmin(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize global role assignment: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrUserForbidden
	}

	if err := s.store.AssignGlobalRole(ctx, userID, string(actorID), role); err != nil {
		return fmt.Errorf("assign global role: %w", err)
	}
	return nil
}

func (s *Service) RemoveGlobalRole(ctx context.Context, actorID domain.Actor, userID, role string) error {
	allowed, err := s.authorizer.AuthorizeAdmin(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize global role removal: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrUserForbidden
	}

	if err := s.store.RemoveGlobalRole(ctx, userID, role); err != nil {
		return fmt.Errorf("remove global role: %w", err)
	}
	return nil
}

func (s *Service) BlockUser(ctx context.Context, actorID domain.Actor, userID, reason string) error {
	allowed, err := s.authorizer.AuthorizeAdmin(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize user blocking: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrUserForbidden
	}

	err = s.store.BlockUser(ctx, string(actorID), userID, reason)
	if err != nil {
		return fmt.Errorf("block user: %w", err)
	}

	return nil
}

func (s *Service) UnblockUser(ctx context.Context, actorID domain.Actor, userID string) error {
	allowed, err := s.authorizer.AuthorizeAdmin(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize user unblocking: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrUserForbidden
	}

	err = s.store.UnblockUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	return nil
}

func (s *Service) Create(ctx context.Context, claims domain.UserClaims) (domain.User, error) {
	user, err := s.store.CreateUser(ctx, claims)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
