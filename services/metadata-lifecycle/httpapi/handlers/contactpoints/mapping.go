package contactpoints

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toDomainContactPointInput(value contactPointRequest) (domain.ContactPointInput, error) {
	input, err := domain.NewContactPointInput(domain.ContactPointInput{
		Email: value.Email,
		Role:  value.Role,
	})
	if err != nil {
		return domain.ContactPointInput{}, fmt.Errorf("map contact point request: %w", err)
	}

	return input, nil
}

func toResponse(value domain.ContactPoint) contactPointResponse {
	return contactPointResponse{
		ID:    value.ID,
		Email: value.Email,
		Role:  value.Role,
	}
}
