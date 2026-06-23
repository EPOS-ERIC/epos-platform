package api

type WriteResult struct {
	ResourceType ResourceType     `json:"resourceType"`
	ID           string           `json:"id"`
	Status       VersioningStatus `json:"status"`
}
