package resource

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// IdentifierIdentities validates caller-supplied identities against the current version.
func IdentifierIdentities(inputs []domain.IdentifierInput, existing []domain.Identifier, target string) (map[uuid.UUID]domain.Identifier, error) {
	byResourceID := make(map[uuid.UUID]domain.Identifier, len(existing))
	for _, identifier := range existing {
		byResourceID[identifier.ResourceID] = identifier
	}

	result := make(map[uuid.UUID]domain.Identifier)
	seen := make(map[uuid.UUID]struct{})
	for i, input := range inputs {
		if input.ResourceID == nil {
			continue
		}
		id := *input.ResourceID
		identifier, ok := byResourceID[id]
		_, duplicate := seen[id]
		if !ok || duplicate || id == uuid.Nil {
			return nil, apperror.Field(fmt.Sprintf("%s[%d].resource_id", target, i), "Identifier must belong to this version and appear only once.", nil)
		}
		seen[id] = struct{}{}
		result[id] = identifier
	}
	return result, nil
}

// IdentifierIDs returns validated existing IDs or generates IDs for a new identifier.
func IdentifierIDs(input domain.IdentifierInput, identities map[uuid.UUID]domain.Identifier) (uuid.UUID, uuid.UUID) {
	if input.ResourceID != nil {
		if identifier, ok := identities[*input.ResourceID]; ok {
			return identifier.ID, identifier.ResourceID
		}
	}
	return uuid.New(), uuid.New()
}
