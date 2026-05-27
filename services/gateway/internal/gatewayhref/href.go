package gatewayhref

import (
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func Build(resourceType api.ResourceType, id string) string {
	return strings.TrimPrefix(fmt.Sprintf("v1/%s/%s", resourceType, id), "/")
}
