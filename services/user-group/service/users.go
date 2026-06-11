package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/db/sqlc"
	"github.com/google/uuid"
)

type User struct {
	ID             string    `json:"id"`
	AuthIdentifier string    `json:"authIdentifier"`
	Email          string    `json:"email,omitempty"`
	GivenName      string    `json:"givenName,omitempty"`
	FamilyName     string    `json:"familyName,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Users struct {
	db *db.PostgresDB
}

func NewUsers(database *db.PostgresDB) *Users {
	return &Users{
		db: database,
	}
}

type CreateUserParams struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
	Status         string
}

type UpdateUserParams struct {
	ID             uuid.UUID
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
	Status         string
}

type PatchUserParams struct {
	AuthIdentifier *string
	Email          *string
	GivenName      *string
	FamilyName     *string
	Status         *string
}

func (s *Users) List(ctx context.Context) ([]User, error) {
	rows, err := s.db.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = toUser(row.ID, row.AuthIdentifier, row.Email, row.GivenName, row.FamilyName, row.Status, row.CreatedAt, row.UpdatedAt)
	}

	return users, nil
}

func (s *Users) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := s.db.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user by id: %w", err)
	}

	return toUser(row.ID, row.AuthIdentifier, row.Email, row.GivenName, row.FamilyName, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Users) GetByAuthIdentifier(ctx context.Context, authIdentifier string) (User, error) {
	row, err := s.db.GetUserByAuthIdentifier(ctx, authIdentifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user by auth identifier: %w", err)
	}

	return toUser(row.ID, row.AuthIdentifier, row.Email, row.GivenName, row.FamilyName, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Users) Create(ctx context.Context, params CreateUserParams) (User, error) {
	params.AuthIdentifier = strings.TrimSpace(params.AuthIdentifier)
	if params.AuthIdentifier == "" {
		return User{}, ErrInvalid
	}
	status := strings.ToUpper(strings.TrimSpace(params.Status))
	if status == "" {
		status = "ACTIVE"
	}
	if !isAllowedUserStatus(status) {
		return User{}, ErrInvalid
	}

	row, err := s.db.CreateUser(ctx, sqlc.CreateUserParams{
		AuthIdentifier: params.AuthIdentifier,
		Email:          nullString(params.Email),
		GivenName:      nullString(params.GivenName),
		FamilyName:     nullString(params.FamilyName),
		Status:         status,
	})
	if err != nil {
		if isUniqueViolation(err, "users_auth_identifier_key") || isUniqueViolation(err, "users_email_key") {
			return User{}, ErrConflict
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return toUser(row.ID, row.AuthIdentifier, row.Email, row.GivenName, row.FamilyName, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Users) Replace(ctx context.Context, params UpdateUserParams) (User, error) {
	params.AuthIdentifier = strings.TrimSpace(params.AuthIdentifier)
	status := strings.ToUpper(strings.TrimSpace(params.Status))
	if params.AuthIdentifier == "" || status == "" || !isAllowedUserStatus(status) {
		return User{}, ErrInvalid
	}

	row, err := s.db.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:             params.ID,
		AuthIdentifier: params.AuthIdentifier,
		Email:          nullString(params.Email),
		GivenName:      nullString(params.GivenName),
		FamilyName:     nullString(params.FamilyName),
		Status:         status,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		if isUniqueViolation(err, "users_auth_identifier_key") || isUniqueViolation(err, "users_email_key") {
			return User{}, ErrConflict
		}
		return User{}, fmt.Errorf("update user: %w", err)
	}

	return toUser(row.ID, row.AuthIdentifier, row.Email, row.GivenName, row.FamilyName, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Users) Patch(ctx context.Context, id uuid.UUID, params PatchUserParams) (User, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	if params.AuthIdentifier != nil {
		current.AuthIdentifier = strings.TrimSpace(*params.AuthIdentifier)
	}
	if params.Email != nil {
		current.Email = strings.TrimSpace(*params.Email)
	}
	if params.GivenName != nil {
		current.GivenName = strings.TrimSpace(*params.GivenName)
	}
	if params.FamilyName != nil {
		current.FamilyName = strings.TrimSpace(*params.FamilyName)
	}
	if params.Status != nil {
		current.Status = strings.ToUpper(strings.TrimSpace(*params.Status))
	}

	return s.Replace(ctx, UpdateUserParams{
		ID:             id,
		AuthIdentifier: current.AuthIdentifier,
		Email:          current.Email,
		GivenName:      current.GivenName,
		FamilyName:     current.FamilyName,
		Status:         current.Status,
	})
}

func (s *Users) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.db.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func toUser(id uuid.UUID, authIdentifier, email, givenName, familyName, status string, createdAt, updatedAt time.Time) User {
	return User{
		ID:             id.String(),
		AuthIdentifier: authIdentifier,
		Email:          email,
		GivenName:      givenName,
		FamilyName:     familyName,
		Status:         status,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}
