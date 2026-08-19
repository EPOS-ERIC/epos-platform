// Package identity resolves authenticated external identities to Backoffice users.
package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalid  = errors.New("invalid identity")
	ErrNotFound = errors.New("user not found")
)

type Claims struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

type Store interface {
	GetUserByAuthIdentifier(context.Context, string) (User, error)
	GetUserByID(context.Context, uuid.UUID) (User, error)
	CreateUser(context.Context, Claims) (User, error)
}

type User struct {
	ID                uuid.UUID
	AuthIdentifier    string
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
	GroupID     uuid.UUID
	GroupName   string
	Roles       []string
	Status      string
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

// GetByAuthIdentifier returns user associated with external identity.
func (s *Service) GetByAuthIdentifier(ctx context.Context, authIdentifier string) (User, error) {
	authIdentifier = strings.TrimSpace(authIdentifier)
	if authIdentifier == "" {
		return User{}, ErrInvalid
	}
	user, err := s.store.GetUserByAuthIdentifier(ctx, authIdentifier)
	if err != nil {
		return User{}, fmt.Errorf("get user by auth identifier: %w", err)
	}
	return user, nil
}

func (s *Service) GetByID(ctx context.Context, userID uuid.UUID) (User, error) {
	if userID == uuid.Nil {
		return User{}, ErrInvalid
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (s *Service) Create(ctx context.Context, claims Claims) (User, error) {
	claims.AuthIdentifier = strings.TrimSpace(claims.AuthIdentifier)
	claims.Email = strings.TrimSpace(claims.Email)
	claims.GivenName = strings.TrimSpace(claims.GivenName)
	claims.FamilyName = strings.TrimSpace(claims.FamilyName)
	if claims.AuthIdentifier == "" || claims.Email == "" || claims.GivenName == "" || claims.FamilyName == "" {
		return User{}, ErrInvalid
	}

	user, err := s.store.CreateUser(ctx, claims)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}
