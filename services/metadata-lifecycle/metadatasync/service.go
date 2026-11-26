// Package metadatasync applies complete snapshots of remote canonical metadata.
package metadatasync

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

type Store interface {
	SyncMetadata(context.Context, []domain.SyncedOrganization, []domain.SyncedContactPoint) error
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	if store == nil {
		panic("metadatasync: nil store")
	}
	return &Service{
		store: store,
	}
}

// Sync applies both complete snapshots. The internal RPC is network restricted.
func (s *Service) Sync(ctx context.Context, orgs []domain.SyncedOrganization, contacts []domain.SyncedContactPoint) error {
	orgNIDs := make(map[int64]bool, len(orgs))
	for _, org := range orgs {
		if orgNIDs[org.NID] {
			return apperror.Field("organizations", "Organization NIDs must be unique.", nil)
		}
		orgNIDs[org.NID] = true
	}

	for _, org := range orgs {
		for _, parent := range org.ParentNIDs {
			if !orgNIDs[parent] || parent == org.NID {
				return apperror.Field("organizations", "Parents must be other organizations in the snapshot.", nil)
			}
		}
	}

	contactNIDs := make(map[int64]bool, len(contacts))
	for _, contact := range contacts {
		if contactNIDs[contact.NID] {
			return apperror.Field("contact_points", "Contact point NIDs must be unique.", nil)
		}

		contactNIDs[contact.NID] = true
	}

	return s.store.SyncMetadata(ctx, orgs, contacts)
}
