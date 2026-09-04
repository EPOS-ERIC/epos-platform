package clients

import (
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func mapSlice[T, K any](values []T, f func(T) K) []K {
	if values == nil {
		return nil
	}
	result := make([]K, len(values))
	for i, value := range values {
		result[i] = f(value)
	}
	return result
}

func mapSliceErr[T, K any](values []T, f func(T) (K, error)) ([]K, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]K, len(values))
	for i, value := range values {
		mapped, err := f(value)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
	}
	return result, nil
}

func toClientResourceTypes(values []string) []client.ResourceType {
	return mapSlice(values, func(value string) client.ResourceType { return client.ResourceType(value) })
}
