package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

const tokenPrefix = "epos_pat_"

func (s *Service) IssueAPIToken(ctx context.Context, actor string, rotate bool) (string, error) {
	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, actor)
	if err != nil {
		return "", fmt.Errorf("authorize API token management: %w", err)
	}

	if !allowed.Allowed {
		return "", apperror.Default(apperror.CodePermissionDenied)
	}

	var random [32]byte
	_, _ = rand.Read(random[:]) // never returns an error

	token := tokenPrefix + base64.RawURLEncoding.EncodeToString(random[:])
	digest := sha256.Sum256([]byte(token))
	if rotate {
		err = s.store.RotateAPIToken(ctx, actor, digest[:])
	} else {
		err = s.store.IssueAPIToken(ctx, actor, digest[:])
	}
	if err != nil {
		return "", fmt.Errorf("save user API token: %w", err)
	}

	event := "user API token issued"
	if rotate {
		event = "user API token rotated"
	}

	s.log.InfoContext(ctx, event, "actor_id", actor, "target_user_id", actor)

	return token, nil
}

func (s *Service) RevokeAPIToken(ctx context.Context, actor string, userID string) error {
	var allowed domain.Result
	var err error

	if actor == userID {
		allowed, err = s.authorizer.AuthorizeActiveUser(ctx, actor)
	} else {
		allowed, err = s.authorizer.AuthorizeGlobal(ctx, actor, domain.GlobalPermissionUserAPITokenRevoke)
	}
	if err != nil {
		return fmt.Errorf("authorize API token revocation: %w", err)
	}

	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	if err := s.store.RevokeAPIToken(ctx, userID); err != nil {
		return fmt.Errorf("revoke API token: %w", err)
	}

	s.log.InfoContext(ctx, "user API token revoked", "actor_id", actor, "target_user_id", userID)

	return nil
}

func (s *Service) FindUserByTokenDigest(ctx context.Context, digest []byte) (string, error) {
	if len(digest) != sha256.Size {
		return "", apperror.Default(apperror.CodeUnauthenticated)
	}

	return s.store.FindUserByTokenDigest(ctx, digest)
}
