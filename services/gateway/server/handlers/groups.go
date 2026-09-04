package handlers

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ListGroups(ctx context.Context) ([]gatewayapi.Group, error) {
	v, err := h.metadataLifecycle.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	out := []gatewayapi.Group{}
	for _, x := range v.Groups {
		mapped, err := toAPIProtoGroup(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func toAPIProtoGroup(v *metadatalifecyclev1.Group) (gatewayapi.Group, error) {
	if v == nil {
		return gatewayapi.Group{}, fmt.Errorf("decode metadata-lifecycle group: response is empty")
	}
	id, err := parseUUID("group.id", v.Id)
	if err != nil {
		return gatewayapi.Group{}, err
	}
	concept, err := toAPIGroupConcept(v.Concept)
	if err != nil {
		return gatewayapi.Group{}, err
	}
	return gatewayapi.Group{
		ID:          id,
		Name:        v.Name,
		Description: optionalString(v.Description),
		Label:       optionalString(v.Label),
		Logo:        optionalString(v.Logo),
		Page:        optionalString(v.Page),
		Color:       optionalString(v.Color),
		Position:    v.Position,
		Concept:     concept,
	}, nil
}

func optionalString(value *string) gatewayapi.OptNilString {
	if value == nil {
		return gatewayapi.OptNilString{}
	}
	return gatewayapi.OptNilString{Value: *value, Set: true}
}

func toAPIGroupConcept(value metadatalifecyclev1.TopLevelConcept) (gatewayapi.GroupConcept, error) {
	switch value {
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA:
		return gatewayapi.GroupConceptDATA, nil
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES:
		return gatewayapi.GroupConceptFACILITIES, nil
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE:
		return gatewayapi.GroupConceptSOFTWARE, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown group concept: %v", value)
	}
}

func (h *ogenHandler) GetGroup(ctx context.Context, q gatewayapi.GetGroupParams) (*gatewayapi.Group, error) {
	v, err := h.metadataLifecycle.GetGroup(ctx, &metadatalifecyclev1.GetGroupRequest{GroupId: q.GroupId.String()})
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	r, err := toAPIProtoGroup(v.Group)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) CreateGroup(ctx context.Context, q *gatewayapi.CreateGroupRequest) (*gatewayapi.Group, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	in := &metadatalifecyclev1.CreateGroupRequest{
		Name:        string(q.Name),
		Description: toClientStringPtr(q.Description),
		Label:       toClientStringPtr(q.Label),
		Logo:        toClientStringPtr(q.Logo),
		Page:        toClientStringPtr(q.Page),
		Color:       toClientStringPtr(q.Color),
		Concept:     string(q.Concept),
	}
	if q.Position.Set {
		in.Position = q.Position.Value
	}
	v, err := h.metadataLifecycle.CreateGroup(ctx, actor, in)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	r, err := toAPIProtoGroup(v.Group)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ReplaceGroup(ctx context.Context, q *gatewayapi.ReplaceGroupRequest, params gatewayapi.ReplaceGroupParams) (*gatewayapi.Group, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	in := &metadatalifecyclev1.ReplaceGroupRequest{
		GroupId:     params.GroupId.String(),
		Name:        string(q.Name),
		Description: toClientStringPtr(q.Description),
		Label:       toClientStringPtr(q.Label),
		Logo:        toClientStringPtr(q.Logo),
		Page:        toClientStringPtr(q.Page),
		Color:       toClientStringPtr(q.Color),
		Concept:     string(q.Concept),
	}
	if q.Position.Set {
		in.Position = q.Position.Value
	}
	v, err := h.metadataLifecycle.ReplaceGroup(ctx, a, in)
	if err != nil {
		return nil, fmt.Errorf("replace group: %w", err)
	}
	r, err := toAPIProtoGroup(v.Group)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func authMissing() error {
	// TODO: inline this
	return errMissingActor
}

func (h *ogenHandler) ListGroupCategories(ctx context.Context, q gatewayapi.ListGroupCategoriesParams) ([]gatewayapi.Category, error) {
	v, err := h.metadataLifecycle.ListCategories(ctx, &metadatalifecyclev1.ListCategoriesRequest{GroupId: q.GroupId.String()})
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	out := []gatewayapi.Category{}
	for _, x := range v.Categories {
		mapped, err := toAPIProtoCategory(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetGroupCategory(ctx context.Context, q gatewayapi.GetGroupCategoryParams) (*gatewayapi.Category, error) {
	v, err := h.metadataLifecycle.GetCategory(ctx, &metadatalifecyclev1.GetCategoryRequest{
		GroupId:    q.GroupId.String(),
		CategoryId: q.CategoryId.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	r, err := toAPIProtoCategory(v.Category)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) CreateGroupCategory(ctx context.Context, q *gatewayapi.CreateCategoryRequest, params gatewayapi.CreateGroupCategoryParams) (*gatewayapi.Category, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	in := &metadatalifecyclev1.CreateCategoryRequest{GroupId: params.GroupId.String(), Name: string(q.Name)}
	if q.ParentCategoryID.Set && !q.ParentCategoryID.Null {
		s := q.ParentCategoryID.Value.String()
		in.ParentCategoryId = &s
	}
	v, err := h.metadataLifecycle.CreateCategory(ctx, a, in)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	r, err := toAPIProtoCategory(v.Category)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ReplaceGroupCategory(ctx context.Context, q *gatewayapi.ReplaceCategoryRequest, params gatewayapi.ReplaceGroupCategoryParams) (*gatewayapi.Category, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	in := &metadatalifecyclev1.ReplaceCategoryRequest{GroupId: params.GroupId.String(), CategoryId: params.CategoryId.String(), Name: string(q.Name)}
	if !q.ParentCategoryID.Null {
		s := q.ParentCategoryID.Value.String()
		in.ParentCategoryId = &s
	}
	v, err := h.metadataLifecycle.ReplaceCategory(ctx, a, in)
	if err != nil {
		return nil, fmt.Errorf("replace category: %w", err)
	}
	r, err := toAPIProtoCategory(v.Category)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ApplyRoleRequestDecision(ctx context.Context, q *gatewayapi.RoleRequestDecisionRequest, p gatewayapi.ApplyRoleRequestDecisionParams) (*gatewayapi.GroupMembership, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.ApplyRoleRequestDecision(ctx, p.GroupId.String(), p.RequestId.String(), string(q.Status), a)
	if err != nil {
		return nil, fmt.Errorf("apply role request decision: %w", err)
	}
	r, err := toAPIMembership(*v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) AssignGroupRole(ctx context.Context, p gatewayapi.AssignGroupRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.AssignGroupRole(ctx, p.GroupId.String(), p.UserId.String(), string(p.Role), a); err != nil {
		return fmt.Errorf("assign group role: %w", err)
	}
	return nil
}

func (h *ogenHandler) RemoveGroupRole(ctx context.Context, p gatewayapi.RemoveGroupRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.RemoveGroupRole(ctx, p.GroupId.String(), p.UserId.String(), string(p.Role), a); err != nil {
		return fmt.Errorf("remove group role: %w", err)
	}
	return nil
}

func (h *ogenHandler) GrantGroupPermission(ctx context.Context, p gatewayapi.GrantGroupPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.GrantGroupPermission(ctx, p.GroupId.String(), p.UserId.String(), string(p.Permission), a); err != nil {
		return fmt.Errorf("grant group permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) RevokeGroupPermission(ctx context.Context, p gatewayapi.RevokeGroupPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.RevokeGroupPermission(ctx, p.GroupId.String(), p.UserId.String(), string(p.Permission), a); err != nil {
		return fmt.Errorf("revoke group permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) WithdrawGroupMembershipRequest(ctx context.Context, p gatewayapi.WithdrawGroupMembershipRequestParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.WithdrawMembershipRequest(ctx, p.GroupId.String(), a); err != nil {
		return fmt.Errorf("withdraw membership request: %w", err)
	}
	return nil
}

func (h *ogenHandler) WithdrawGroupRoleRequest(ctx context.Context, p gatewayapi.WithdrawGroupRoleRequestParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.WithdrawRoleRequest(ctx, p.GroupId.String(), p.RequestId.String(), a); err != nil {
		return fmt.Errorf("withdraw role request: %w", err)
	}
	return nil
}

func (h *ogenHandler) RequestGroupRole(ctx context.Context, q *gatewayapi.CreateRoleRequest, p gatewayapi.RequestGroupRoleParams) (*gatewayapi.RoleRequest, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.RequestRole(ctx, p.GroupId.String(), string(q.Role), a)
	if err != nil {
		return nil, fmt.Errorf("request role: %w", err)
	}
	r, err := toAPIRole(*v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) DeleteGroupCategory(ctx context.Context, params gatewayapi.DeleteGroupCategoryParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if _, err := h.metadataLifecycle.DeleteCategory(ctx, a, &metadatalifecyclev1.DeleteCategoryRequest{GroupId: params.GroupId.String(), CategoryId: params.CategoryId.String()}); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (h *ogenHandler) DeleteGroup(ctx context.Context, p gatewayapi.DeleteGroupParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if _, err := h.metadataLifecycle.DeleteGroup(ctx, a, &metadatalifecyclev1.DeleteGroupRequest{GroupId: p.GroupId.String()}); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}

func (h *ogenHandler) ListGroupMemberships(ctx context.Context, params gatewayapi.ListGroupMembershipsParams) ([]gatewayapi.GroupMembership, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.ListGroupMemberships(ctx, params.GroupId.String(), a)
	if err != nil {
		return nil, fmt.Errorf("list group memberships: %w", err)
	}
	out := []gatewayapi.GroupMembership{}
	for _, x := range v {
		mapped, err := toAPIMembership(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) RequestGroupMembership(ctx context.Context, q *gatewayapi.CreateMembershipRequest, params gatewayapi.RequestGroupMembershipParams) (*gatewayapi.GroupMembership, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	roles := make([]string, len(q.Roles))
	for i := range q.Roles {
		roles[i] = string(q.Roles[i])
	}
	v, err := h.metadataLifecycle.RequestMembership(ctx, params.GroupId.String(), a, roles)
	if err != nil {
		return nil, fmt.Errorf("request membership: %w", err)
	}
	r, err := toAPIMembership(*v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) DecideGroupMembership(ctx context.Context, q *gatewayapi.MembershipDecisionRequest, params gatewayapi.DecideGroupMembershipParams) (*gatewayapi.GroupMembership, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.DecideMembership(ctx, params.GroupId.String(), params.UserId.String(), string(q.Status), a)
	if err != nil {
		return nil, fmt.Errorf("decide membership: %w", err)
	}
	r, err := toAPIMembership(*v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) DeleteGroupMembership(ctx context.Context, params gatewayapi.DeleteGroupMembershipParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.RevokeMembership(ctx, params.GroupId.String(), params.UserId.String(), a); err != nil {
		return fmt.Errorf("revoke membership: %w", err)
	}
	return nil
}
