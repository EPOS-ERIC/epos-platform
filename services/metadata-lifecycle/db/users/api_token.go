package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/jackc/pgx/v5"
)

func (r *Repo) IssueAPIToken(ctx context.Context, userID string, digest []byte) error {
	changed, err := r.IssueUserAPIToken(ctx, sqlc.IssueUserAPITokenParams{UserID: userID, Digest: digest})
	if err != nil {
		// PostgreSQL constraint errors can contain the digest.
		return fmt.Errorf("issue user API token")
	}

	if changed == 0 {
		return apperror.Default(apperror.CodeConflict)
	}

	return nil
}

func (r *Repo) RotateAPIToken(ctx context.Context, userID string, digest []byte) error {
	changed, err := r.RotateUserAPIToken(ctx, sqlc.RotateUserAPITokenParams{UserID: userID, Digest: digest})
	if err != nil {
		return fmt.Errorf("rotate user API token")
	}

	if changed == 0 {
		return apperror.Default(apperror.CodeNotFound)
	}

	return nil
}

func (r *Repo) RevokeAPIToken(ctx context.Context, userID string) error {
	changed, err := r.RevokeUserAPIToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("revoke user API token: %w", err)
	}

	if changed != 0 {
		return nil
	}

	_, err = r.GetAuthorizationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Default(apperror.CodeNotFound)
	}

	if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}

	return nil
}

func (r *Repo) FindUserByTokenDigest(ctx context.Context, digest []byte) (string, error) {
	userID, err := r.FindUserByAPITokenDigest(ctx, digest)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperror.Default(apperror.CodeUnauthenticated)
	}

	if err != nil {
		return "", fmt.Errorf("find user by API token digest")
	}

	return userID, nil
}
