package contactpoints

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toCreateContactPointParams(id uuid.UUID, input domain.ContactPointInput) sqlc.CreateContactPointParams {
	return sqlc.CreateContactPointParams{
		ID:    id,
		Email: input.Email,
	}
}

func toReplaceContactPointParams(id uuid.UUID, input domain.ContactPointInput) sqlc.ReplaceContactPointParams {
	return sqlc.ReplaceContactPointParams{
		ID:    id,
		Email: input.Email,
	}
}

func toDomainContactPoint(row sqlc.ContactPoint) (domain.ContactPoint, error) {
	contactPoint := domain.ContactPoint{
		ID:    row.ID,
		Email: row.Email,
	}
	if err := contactPoint.Normalize(); err != nil {
		return domain.ContactPoint{}, fmt.Errorf("bad contact point: %w", err)
	}

	return contactPoint, nil
}
