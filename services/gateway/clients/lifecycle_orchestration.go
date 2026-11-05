package clients

import (
	"fmt"

	"github.com/google/uuid"
)

func fromProtoDistributionPairs(values map[string]string) (map[uuid.UUID]uuid.UUID, error) {
	result := make(map[uuid.UUID]uuid.UUID, len(values))
	for sourceID, targetID := range values {
		source, err := uuid.Parse(sourceID)
		if err != nil {
			return nil, fmt.Errorf("invalid source distribution id: %w", err)
		}
		target, err := uuid.Parse(targetID)
		if err != nil {
			return nil, fmt.Errorf("invalid target distribution id: %w", err)
		}
		result[source] = target
	}
	return result, nil
}

func fromProtoDistributionIDs(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid removed distribution id: %w", err)
		}
		result = append(result, id)
	}
	return result, nil
}
