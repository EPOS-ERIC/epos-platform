package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/google/uuid"
)

var usersLog = logging.Get("users-service")

type User struct {
	ID             string     `json:"id"`
	AuthIdentifier string     `json:"authIdentifier"`
	Email          string     `json:"email,omitempty"`
	GivenName      string     `json:"givenName,omitempty"`
	FamilyName     string     `json:"familyName,omitempty"`
	IsSuperAdmin   bool       `json:"isSuperAdmin"`
	Status         UserStatus `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
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
	IsSuperAdmin   bool
	Status         UserStatus
}

type UpdateUserParams struct {
	ID             uuid.UUID
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
	IsSuperAdmin   bool
	Status         UserStatus
}

type PatchUserParams struct {
	AuthIdentifier *string
	Email          *string
	GivenName      *string
	FamilyName     *string
	IsSuperAdmin   *bool
	Status         *UserStatus
}

func (s *Users) List(ctx context.Context) ([]User, error) {
	rows, err := s.db.ListUsers(ctx)
	if err != nil {
		usersLog.ErrorContext(ctx, "list users failed", "error", err)
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = User{
			ID:             row.ID.String(),
			AuthIdentifier: row.AuthIdentifier,
			Email:          row.Email,
			GivenName:      row.GivenName,
			FamilyName:     row.FamilyName,
			IsSuperAdmin:   row.IsSuperAdmin,
			Status:         UserStatus(row.Status),
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}

	return users, nil
}

func (s *Users) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := s.db.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		usersLog.ErrorContext(ctx, "get user by id failed", "user_id", id, "error", err)
		return User{}, fmt.Errorf("get user by id: %w", err)
	}

	return User{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		IsSuperAdmin:   row.IsSuperAdmin,
		Status:         UserStatus(row.Status),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (s *Users) GetByAuthIdentifier(ctx context.Context, authIdentifier string) (User, error) {
	row, err := s.db.GetUserByAuthIdentifier(ctx, authIdentifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		usersLog.ErrorContext(ctx, "get user by auth identifier failed", "auth_identifier", authIdentifier, "error", err)
		return User{}, fmt.Errorf("get user by auth identifier: %w", err)
	}

	return User{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		IsSuperAdmin:   row.IsSuperAdmin,
		Status:         UserStatus(row.Status),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (s *Users) Create(ctx context.Context, params CreateUserParams) (User, error) {
	params.AuthIdentifier = strings.TrimSpace(params.AuthIdentifier)
	if params.AuthIdentifier == "" {
		return User{}, ErrInvalid
	}

	status := NormalizeUserStatus(string(params.Status))
	if status == "" {
		status = UserStatusActive
	}
	if !status.Valid() {
		return User{}, ErrInvalid
	}

	row, err := s.db.CreateUser(ctx, sqlc.CreateUserParams{
		AuthIdentifier: params.AuthIdentifier,
		Email:          nullString(params.Email),
		GivenName:      nullString(params.GivenName),
		FamilyName:     nullString(params.FamilyName),
		IsSuperAdmin:   params.IsSuperAdmin,
		Status:         sqlc.UserStatus(status),
	})
	if err != nil {
		if isUniqueViolation(err, "users_auth_identifier_key") || isUniqueViolation(err, "users_email_key") {
			return User{}, ErrConflict
		}
		usersLog.ErrorContext(ctx, "create user failed", "auth_identifier", params.AuthIdentifier, "error", err)
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return User{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		IsSuperAdmin:   row.IsSuperAdmin,
		Status:         UserStatus(row.Status),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (s *Users) Replace(ctx context.Context, params UpdateUserParams) (User, error) {
	params.AuthIdentifier = strings.TrimSpace(params.AuthIdentifier)
	status := NormalizeUserStatus(string(params.Status))
	if params.AuthIdentifier == "" || status == "" || !status.Valid() {
		return User{}, ErrInvalid
	}

	row, err := s.db.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:             params.ID,
		AuthIdentifier: params.AuthIdentifier,
		Email:          nullString(params.Email),
		GivenName:      nullString(params.GivenName),
		FamilyName:     nullString(params.FamilyName),
		IsSuperAdmin:   params.IsSuperAdmin,
		Status:         sqlc.UserStatus(status),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		if isUniqueViolation(err, "users_auth_identifier_key") || isUniqueViolation(err, "users_email_key") {
			return User{}, ErrConflict
		}
		usersLog.ErrorContext(ctx, "update user failed", "user_id", params.ID, "error", err)
		return User{}, fmt.Errorf("update user: %w", err)
	}

	return User{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		IsSuperAdmin:   row.IsSuperAdmin,
		Status:         UserStatus(row.Status),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
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
	if params.IsSuperAdmin != nil {
		current.IsSuperAdmin = *params.IsSuperAdmin
	}
	if params.Status != nil {
		current.Status = *params.Status
	}

	return s.Replace(ctx, UpdateUserParams{
		ID:             id,
		AuthIdentifier: current.AuthIdentifier,
		Email:          current.Email,
		GivenName:      current.GivenName,
		FamilyName:     current.FamilyName,
		IsSuperAdmin:   current.IsSuperAdmin,
		Status:         current.Status,
	})
}

func (s *Users) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.db.DeleteUser(ctx, id); err != nil {
		usersLog.ErrorContext(ctx, "delete user failed", "user_id", id, "error", err)
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
