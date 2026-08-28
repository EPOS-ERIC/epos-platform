package groups

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toDomainGroupInput(value groupRequest) (domain.GroupInput, error) {
	input, err := domain.NewGroupInput(domain.GroupInput{
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		Concept:     value.Concept,
	})
	if err != nil {
		return domain.GroupInput{}, fmt.Errorf("map group request: %w", err)
	}

	return input, nil
}

func toResponse(value domain.Group) groupResponse {
	return groupResponse{
		ID:          value.ID,
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		Concept:     value.Concept,
	}
}
