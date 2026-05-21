package repository

import (
	"database/sql"
	"errors"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

type UserPostgresRepository struct {
	db *sql.DB
}

func NewUserPostgresRepository(db *sql.DB) *UserPostgresRepository {
	return &UserPostgresRepository{db: db}
}

func (r *UserPostgresRepository) List() []domain.User {
	rows, err := r.db.Query(`SELECT id::text, auth_identifier, COALESCE(email,''), COALESCE(given_name,''), COALESCE(family_name,''), status, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return []domain.User{}
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.AuthIdentifier, &u.Email, &u.GivenName, &u.FamilyName, &u.Status, &u.CreatedAt, &u.UpdatedAt); err == nil {
			users = append(users, u)
		}
	}
	return users
}

func (r *UserPostgresRepository) GetByID(id string) (domain.User, error) {
	return r.getOne(`SELECT id::text, auth_identifier, COALESCE(email,''), COALESCE(given_name,''), COALESCE(family_name,''), status, created_at, updated_at FROM users WHERE id = $1`, id)
}

func (r *UserPostgresRepository) GetByAuthIdentifier(authIdentifier string) (domain.User, error) {
	return r.getOne(`SELECT id::text, auth_identifier, COALESCE(email,''), COALESCE(given_name,''), COALESCE(family_name,''), status, created_at, updated_at FROM users WHERE auth_identifier = $1`, authIdentifier)
}

func (r *UserPostgresRepository) Create(authIdentifier, email, givenName, familyName, status string) (domain.User, error) {
	if !isAllowedUserStatus(status) {
		return domain.User{}, ErrUserInvalidStatus
	}
	return r.getOne(`INSERT INTO users (auth_identifier, email, given_name, family_name, status) VALUES ($1,$2,$3,$4,$5) RETURNING id::text, auth_identifier, COALESCE(email,''), COALESCE(given_name,''), COALESCE(family_name,''), status, created_at, updated_at`, authIdentifier, nullIfEmpty(email), nullIfEmpty(givenName), nullIfEmpty(familyName), status)
}

func (r *UserPostgresRepository) Replace(id, authIdentifier, email, givenName, familyName, status string) (domain.User, error) {
	if !isAllowedUserStatus(status) {
		return domain.User{}, ErrUserInvalidStatus
	}
	res, err := r.db.Exec(`UPDATE users SET auth_identifier=$2, email=$3, given_name=$4, family_name=$5, status=$6, updated_at=now() WHERE id=$1`, id, authIdentifier, nullIfEmpty(email), nullIfEmpty(givenName), nullIfEmpty(familyName), status)
	if err != nil {
		return domain.User{}, mapDBError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.User{}, ErrUserNotFound
	}
	return r.GetByID(id)
}

func (r *UserPostgresRepository) Patch(id string, authIdentifier, email, givenName, familyName, status *string) (domain.User, error) {
	u, err := r.GetByID(id)
	if err != nil {
		return domain.User{}, err
	}
	if authIdentifier != nil { u.AuthIdentifier = *authIdentifier }
	if email != nil { u.Email = *email }
	if givenName != nil { u.GivenName = *givenName }
	if familyName != nil { u.FamilyName = *familyName }
	if status != nil { u.Status = *status }
	return r.Replace(id, u.AuthIdentifier, u.Email, u.GivenName, u.FamilyName, u.Status)
}

func (r *UserPostgresRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserPostgresRepository) getOne(q string, args ...any) (domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(q, args...).Scan(&u.ID, &u.AuthIdentifier, &u.Email, &u.GivenName, &u.FamilyName, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, ErrUserNotFound
		}
		return domain.User{}, mapDBError(err)
	}
	return u, nil
}
