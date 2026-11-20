package db

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/organizations"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/jackc/pgx/v5"
)

// SyncMetadata applies both snapshots or rolls back both.
func (d *PostgresDB) SyncMetadata(ctx context.Context, orgs []domain.SyncedOrganization, contacts []domain.SyncedContactPoint) error {
	return dbtx.RunWithTx(ctx, d.db, d.Organizations.Queries, func(q *sqlc.Queries, tx pgx.Tx) error {
		// Serialize snapshot writes across Gateway replicas.
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('organization-sync'))"); err != nil {
			return fmt.Errorf("lock metadata sync: %w", err)
		}

		if err := organizations.Sync(ctx, q, orgs); err != nil {
			return fmt.Errorf("sync organizations: %w", err)
		}

		return contactpoints.Sync(ctx, q, contacts)
	})
}
