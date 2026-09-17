package handlers

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
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

func toProtoGroupConcept(value string) metadatalifecyclev1.TopLevelConcept {
	switch value {
	case string(gatewayapi.GroupConceptDATA):
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA
	case string(gatewayapi.GroupConceptFACILITIES):
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES
	case string(gatewayapi.GroupConceptSOFTWARE):
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE
	default:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_UNSPECIFIED
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
		Concept:     toProtoGroupConcept(string(q.Concept)),
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
		Concept:     toProtoGroupConcept(string(q.Concept)),
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
		in.ParentCategoryId = new(q.ParentCategoryID.Value.String())
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
		in.ParentCategoryId = new(q.ParentCategoryID.Value.String())
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

func (h *ogenHandler) DecideGroupRoleRequest(ctx context.Context, p gatewayapi.DecideGroupRoleRequestParams) (*gatewayapi.GroupMembership, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	decision, err := toProtoDecision(p.Decision)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycle.DecideRoleRequest(ctx, p.GroupId.String(), p.RequestId.String(), decision, a)
	if err != nil {
		return nil, fmt.Errorf("decide role request: %w", err)
	}
	r, err := toAPIProtoGroupMembership(v.Membership)
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
	role, err := toProtoGroupRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycle.AssignGroupRole(ctx, p.GroupId.String(), p.UserId, role, a); err != nil {
		return fmt.Errorf("assign group role: %w", err)
	}
	return nil
}

func (h *ogenHandler) RemoveGroupRole(ctx context.Context, p gatewayapi.RemoveGroupRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	role, err := toProtoGroupRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycle.RemoveGroupRole(ctx, p.GroupId.String(), p.UserId, role, a); err != nil {
		return fmt.Errorf("remove group role: %w", err)
	}
	return nil
}

func (h *ogenHandler) GrantGroupPermission(ctx context.Context, p gatewayapi.GrantGroupPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	permission, err := domain.NewGroupPermission(string(p.Permission))
	if err != nil {
		return fmt.Errorf("map group permission: %w", err)
	}
	if err := h.metadataLifecycle.GrantGroupPermission(ctx, p.GroupId.String(), p.UserId, permission, a); err != nil {
		return fmt.Errorf("grant group permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) RevokeGroupPermission(ctx context.Context, p gatewayapi.RevokeGroupPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	permission, err := domain.NewGroupPermission(string(p.Permission))
	if err != nil {
		return fmt.Errorf("map group permission: %w", err)
	}
	if err := h.metadataLifecycle.RevokeGroupPermission(ctx, p.GroupId.String(), p.UserId, permission, a); err != nil {
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
	role, err := toProtoRoleRequestRole(q.Role)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycle.RequestRole(ctx, p.GroupId.String(), role, a)
	if err != nil {
		return nil, fmt.Errorf("request role: %w", err)
	}
	r, err := toAPIProtoRoleRequest(v.Request)
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
	v, err := h.metadataLifecycle.ListGroupMemberships(ctx, a, &metadatalifecyclev1.ListGroupMembershipsRequest{GroupId: params.GroupId.String()})
	if err != nil {
		return nil, fmt.Errorf("list group memberships: %w", err)
	}
	out := []gatewayapi.GroupMembership{}
	for _, x := range v.Memberships {
		mapped, err := toAPIProtoGroupMembership(x)
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
	roles := make([]metadatalifecyclev1.GroupRole, len(q.Roles))
	for i := range q.Roles {
		role, err := toProtoMembershipRole(q.Roles[i])
		if err != nil {
			return nil, err
		}
		roles[i] = role
	}
	v, err := h.metadataLifecycle.RequestMembership(ctx, params.GroupId.String(), a, roles)
	if err != nil {
		return nil, fmt.Errorf("request membership: %w", err)
	}
	r, err := toAPIProtoGroupMembership(v.Membership)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) DecideGroupMembership(ctx context.Context, params gatewayapi.DecideGroupMembershipParams) (*gatewayapi.GroupMembership, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	decision, err := toProtoDecision(params.Decision)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycle.DecideMembership(ctx, params.GroupId.String(), params.UserId, decision, a)
	if err != nil {
		return nil, fmt.Errorf("decide membership: %w", err)
	}
	r, err := toAPIProtoGroupMembership(v.Membership)
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
	if err := h.membership.RevokeMembership(ctx, a, params.GroupId.String(), params.UserId); err != nil {
		return fmt.Errorf("revoke membership: %w", err)
	}
	return nil
}
