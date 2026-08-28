package groups

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toCreateGroupParams(id uuid.UUID, input domain.GroupInput) sqlc.CreateGroupParams {
	return sqlc.CreateGroupParams{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Concept:     sqlc.TopLevelConcept(input.Concept),
		Label:       input.Label,
		Logo:        input.Logo,
		Page:        input.Page,
		Color:       input.Color,
		Position:    input.Position,
	}
}

func toReplaceGroupParams(id uuid.UUID, input domain.GroupInput) sqlc.ReplaceGroupParams {
	return sqlc.ReplaceGroupParams{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Label:       input.Label,
		Logo:        input.Logo,
		Page:        input.Page,
		Color:       input.Color,
		Position:    input.Position,
		Concept:     sqlc.TopLevelConcept(input.Concept),
	}
}

func toDomainGroup(row sqlc.Group) (domain.Group, error) {
	group, err := domain.NewGroup(row.ID, domain.GroupInput{
		Name:        row.Name,
		Description: row.Description,
		Label:       row.Label,
		Logo:        row.Logo,
		Page:        row.Page,
		Color:       row.Color,
		Position:    row.Position,
		Concept:     string(row.Concept),
	})
	if err != nil {
		return domain.Group{}, fmt.Errorf("bad group: %w", err)
	}

	return group, nil
}
