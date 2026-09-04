package handlers

import (
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toAPIGroup(v metadatalifecycleclient.Group) (gatewayapi.Group, error) {
	id, err := parseUUID("group.id", v.Id)
	if err != nil {
		return gatewayapi.Group{}, err
	}
	return gatewayapi.Group{
		ID:   id,
		Name: v.Name,
		Description: gatewayapi.OptNilString{
			Value: value(v.Description.Get()),
			Set:   v.Description.Get() != nil,
		},
		Label: gatewayapi.OptNilString{
			Value: value(v.Label.Get()),
			Set:   v.Label.Get() != nil,
		},
		Logo: gatewayapi.OptNilString{
			Value: value(v.Logo.Get()),
			Set:   v.Logo.Get() != nil,
		},
		Page: gatewayapi.OptNilString{
			Value: value(v.Page.Get()),
			Set:   v.Page.Get() != nil,
		},
		Color: gatewayapi.OptNilString{
			Value: value(v.Color.Get()),
			Set:   v.Color.Get() != nil,
		},
		Position: v.Position,
		Concept:  gatewayapi.GroupConcept(v.Concept),
	}, nil
}

func toAPICategory(v metadatalifecycleclient.CanonicalCategory) (gatewayapi.Category, error) {
	id, err := parseUUID("category.id", v.Id)
	if err != nil {
		return gatewayapi.Category{}, err
	}
	groupID, err := parseUUID("category.group_id", v.GroupId)
	if err != nil {
		return gatewayapi.Category{}, err
	}
	r := gatewayapi.Category{
		ID:      id,
		GroupID: groupID,
		Name:    v.Name,
	}
	if parent := v.ParentCategoryId.Get(); parent != nil {
		parentID, err := parseUUID("category.parent_category_id", *parent)
		if err != nil {
			return gatewayapi.Category{}, err
		}
		r.ParentCategoryID.SetTo(parentID)
	}
	return r, nil
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
