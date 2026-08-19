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
	ID         string
	Email      string
	GivenName  string
	FamilyName string
}

type Store interface {
	GetUserByID(context.Context, string) (User, error)
	CreateUser(context.Context, Claims) (User, error)
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

func (s *Service) GetByID(ctx context.Context, userID string) (User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return User{}, ErrInvalid
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
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
