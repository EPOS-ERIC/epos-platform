package resourcehref

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func Build(resourceType api.ResourceType, id string) string {
	return fmt.Sprintf("v1/resources/%s/%s", resourceType, id)
}
