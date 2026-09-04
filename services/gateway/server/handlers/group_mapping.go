package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toAPIProtoCategory(v *metadatalifecyclev1.CanonicalCategory) (gatewayapi.Category, error) {
	if v == nil {
		return gatewayapi.Category{}, fmt.Errorf("decode metadata-lifecycle category: response is empty")
	}
	id, err := parseUUID("category.id", v.Id)
	if err != nil {
		return gatewayapi.Category{}, err
	}
	groupID, err := parseUUID("category.group_id", v.GroupId)
	if err != nil {
		return gatewayapi.Category{}, err
	}
	result := gatewayapi.Category{ID: id, GroupID: groupID, Name: v.Name}
	if v.ParentCategoryId != nil {
		parentID, err := parseUUID("category.parent_category_id", *v.ParentCategoryId)
		if err != nil {
			return gatewayapi.Category{}, err
		}
		result.ParentCategoryID.SetTo(parentID)
	} else {
		result.ParentCategoryID.SetToNull()
	}
	return result, nil
}
