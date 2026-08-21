// Package identity resolves authenticated external identities to Backoffice users.
package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/google/uuid"
)

var (
	ErrInvalid   = errors.New("invalid identity")
	ErrNotFound  = errors.New("user not found")
	ErrForbidden = errors.New("identity operation forbidden")
	ErrConflict  = errors.New("identity operation conflict")
)

type Claims struct {
	ID         string
	Email      string
	GivenName  string
	FamilyName string
}

// TODO: see if we should move this interface to the consuming package instead of the implementing one (also for the other services)
type Store interface {
	GetUserByID(context.Context, string) (User, error)
	ListUsers(context.Context) ([]UserSummary, error)
	CreateUser(context.Context, Claims) (User, error)
	GrantGlobalPermission(context.Context, string, string, string) error
	RevokeGlobalPermission(context.Context, string, string) error
	AssignGlobalRole(context.Context, string, string, string) error
	RemoveGlobalRole(context.Context, string, string) error
	BlockUser(context.Context, string, string, string) error
	UnblockUser(context.Context, string) error
}

type Authorizer interface {
	CanReadUser(context.Context, string, string) (authz.Result, error)
	CanListUsers(context.Context, string) (authz.Result, error)
	AuthorizeAdmin(context.Context, string) (authz.Result, error)
	AuthorizeGlobal(context.Context, string, string) (authz.Result, error)
}

type UserSummary struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            string
	GlobalRoles       []string
	GlobalPermissions []string
}

type User struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	GlobalRoles       []string
	GlobalPermissions []string
	Memberships       []Membership
}

type Membership struct {
	GroupID      uuid.UUID
	GroupName    string
	Roles        []string
	Status       string
	Permissions  []string
	RoleRequests []RoleRequest
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RoleRequest struct {
	ID          uuid.UUID
	GroupID     uuid.UUID
	Role        string
	Status      string
	RequestedAt time.Time
}

type Service struct {
	store      Store
	authorizer Authorizer
}

func New(store Store, authorizer Authorizer) *Service {
	return &Service{store: store, authorizer: authorizer}
}

func (s *Service) GetByID(ctx context.Context, actorID, userID string) (User, error) {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	if actorID == "" || userID == "" {
		return User{}, ErrInvalid
	}

	allowed, err := s.authorizer.CanReadUser(ctx, actorID, userID)
	if err != nil {
		return User{}, fmt.Errorf("authorize user read: %w", err)
	}

	if !allowed.Allowed {
		return User{}, ErrForbidden
	}

	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (s *Service) List(ctx context.Context, actorID string) ([]UserSummary, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, ErrInvalid
	}

	allowed, err := s.authorizer.CanListUsers(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("authorize user list: %w", err)
	}

	if !allowed.Allowed {
		return nil, ErrForbidden
	}

	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

func (s *Service) GrantGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	if actorID == "" || userID == "" || permission == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "global.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize global permission grant: %w", err)
	}

	if !allowed.Allowed {
		return ErrForbidden
	}

	err = s.store.GrantGlobalPermission(ctx, userID, permission, actorID)
	if err != nil {
		return fmt.Errorf("grant global permission: %w", err)
	}

	return nil
}

func (s *Service) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	if actorID == "" || userID == "" || permission == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "global.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize global permission revoke: %w", err)
	}

	if !allowed.Allowed {
		return ErrForbidden
	}

	err = s.store.RevokeGlobalPermission(ctx, userID, permission)
	if err != nil {
		return fmt.Errorf("revoke global permission: %w", err)
	}

	return nil
}

func (s *Service) AssignGlobalRole(ctx context.Context, actorID, userID, role string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	if actorID == "" || userID == "" || role != "ADMIN" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeAdmin(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize global role assignment: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.AssignGlobalRole(ctx, userID, actorID, role); err != nil {
		return fmt.Errorf("assign global role: %w", err)
	}
	return nil
}

func (s *Service) RemoveGlobalRole(ctx context.Context, actorID, userID, role string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	if actorID == "" || userID == "" || role != "ADMIN" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeAdmin(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize global role removal: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.RemoveGlobalRole(ctx, userID, role); err != nil {
		return fmt.Errorf("remove global role: %w", err)
	}
	return nil
}

func (s *Service) BlockUser(ctx context.Context, actorID, userID, reason string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	reason = strings.TrimSpace(reason)
	if actorID == "" || userID == "" || reason == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeAdmin(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize user blocking: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	err = s.store.BlockUser(ctx, actorID, userID, reason)
	if err != nil {
		return fmt.Errorf("block user: %w", err)
	}

	return nil
}

func (s *Service) UnblockUser(ctx context.Context, actorID, userID string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	if actorID == "" || userID == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeAdmin(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize user unblocking: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	err = s.store.UnblockUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	return nil
}

func (s *Service) Create(ctx context.Context, claims Claims) (User, error) {
	claims.ID = strings.TrimSpace(claims.ID)
	claims.Email = strings.TrimSpace(claims.Email)
	claims.GivenName = strings.TrimSpace(claims.GivenName)
	claims.FamilyName = strings.TrimSpace(claims.FamilyName)
	if claims.ID == "" || claims.Email == "" || claims.GivenName == "" || claims.FamilyName == "" {
		return User{}, ErrInvalid
	}

	user, err := s.store.CreateUser(ctx, claims)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
