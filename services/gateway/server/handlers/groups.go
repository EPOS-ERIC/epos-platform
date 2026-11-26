package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ListGroups(ctx context.Context) ([]gatewayapi.Group, error) {
	v, err := h.metadataLifecycleClient.ListGroups(ctx)
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
	concept, err := parseUUID("group.concept_id", v.ConceptId)
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
		ConceptID:   concept,
	}, nil
}

func optionalString(value *string) gatewayapi.OptNilString {
	if value == nil {
		return gatewayapi.OptNilString{}
	}
	return gatewayapi.OptNilString{Value: *value, Set: true}
}

func (h *ogenHandler) GetGroup(ctx context.Context, q gatewayapi.GetGroupParams) (*gatewayapi.Group, error) {
	v, err := h.metadataLifecycleClient.GetGroup(ctx, &metadatalifecyclev1.GetGroupRequest{GroupId: q.GroupId.String()})
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
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	in := &metadatalifecyclev1.CreateGroupRequest{
		Name:        q.Name,
		Description: toClientStringPtr(q.Description),
		Label:       toClientStringPtr(q.Label),
		Logo:        toClientStringPtr(q.Logo),
		Page:        toClientStringPtr(q.Page),
		Color:       toClientStringPtr(q.Color),
		ConceptId:   q.ConceptID.String(),
	}
	if q.Position.Set {
		in.Position = q.Position.Value
	}
	v, err := h.metadataLifecycleClient.CreateGroup(ctx, actor, in)
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	in := &metadatalifecyclev1.ReplaceGroupRequest{
		GroupId:     params.GroupId.String(),
		Name:        q.Name,
		Description: toClientStringPtr(q.Description),
		Label:       toClientStringPtr(q.Label),
		Logo:        toClientStringPtr(q.Logo),
		Page:        toClientStringPtr(q.Page),
		Color:       toClientStringPtr(q.Color),
		ConceptId:   q.ConceptID.String(),
	}
	if q.Position.Set {
		in.Position = q.Position.Value
	}
	v, err := h.metadataLifecycleClient.ReplaceGroup(ctx, a, in)
	if err != nil {
		return nil, fmt.Errorf("replace group: %w", err)
	}
	r, err := toAPIProtoGroup(v.Group)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ListGroupCategories(ctx context.Context, q gatewayapi.ListGroupCategoriesParams) ([]gatewayapi.Category, error) {
	v, err := h.metadataLifecycleClient.ListCategories(ctx, &metadatalifecyclev1.ListCategoriesRequest{GroupId: q.GroupId.String()})
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
	v, err := h.metadataLifecycleClient.GetCategory(ctx, &metadatalifecyclev1.GetCategoryRequest{
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	in := &metadatalifecyclev1.CreateCategoryRequest{
		GroupId: params.GroupId.String(),
		Name:    q.Name,
	}
	if q.ParentCategoryID.Set && !q.ParentCategoryID.Null {
		in.ParentCategoryId = new(q.ParentCategoryID.Value.String())
	}
	v, err := h.metadataLifecycleClient.CreateCategory(ctx, a, in)
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	in := &metadatalifecyclev1.ReplaceCategoryRequest{
		GroupId:    params.GroupId.String(),
		CategoryId: params.CategoryId.String(),
		Name:       q.Name,
	}
	if !q.ParentCategoryID.Null {
		in.ParentCategoryId = new(q.ParentCategoryID.Value.String())
	}
	v, err := h.metadataLifecycleClient.ReplaceCategory(ctx, a, in)
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	decision, err := toProtoDecision(p.Decision)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.DecideRoleRequest(ctx, p.GroupId.String(), p.RequestId.String(), decision, a)
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	role, err := toProtoGroupRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycleClient.AssignGroupRole(ctx, p.GroupId.String(), p.UserId, role, a); err != nil {
		return fmt.Errorf("assign group role: %w", err)
	}
	return nil
}

func (h *ogenHandler) RemoveGroupRole(ctx context.Context, p gatewayapi.RemoveGroupRoleParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	role, err := toProtoGroupRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycleClient.RemoveGroupRole(ctx, p.GroupId.String(), p.UserId, role, a); err != nil {
		return fmt.Errorf("remove group role: %w", err)
	}
	return nil
}

func (h *ogenHandler) GrantGroupPermission(ctx context.Context, p gatewayapi.GrantGroupPermissionParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	permission, err := domain.NewGroupPermission(string(p.Permission))
	if err != nil {
		return fmt.Errorf("map group permission: %w", err)
	}
	if err := h.metadataLifecycleClient.GrantGroupPermission(ctx, p.GroupId.String(), p.UserId, permission, a); err != nil {
		return fmt.Errorf("grant group permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) RevokeGroupPermission(ctx context.Context, p gatewayapi.RevokeGroupPermissionParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	permission, err := domain.NewGroupPermission(string(p.Permission))
	if err != nil {
		return fmt.Errorf("map group permission: %w", err)
	}
	if err := h.metadataLifecycleClient.RevokeGroupPermission(ctx, p.GroupId.String(), p.UserId, permission, a); err != nil {
		return fmt.Errorf("revoke group permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) WithdrawGroupMembershipRequest(ctx context.Context, p gatewayapi.WithdrawGroupMembershipRequestParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if err := h.metadataLifecycleClient.WithdrawMembershipRequest(ctx, p.GroupId.String(), a); err != nil {
		return fmt.Errorf("withdraw membership request: %w", err)
	}
	return nil
}

func (h *ogenHandler) WithdrawGroupRoleRequest(ctx context.Context, p gatewayapi.WithdrawGroupRoleRequestParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if err := h.metadataLifecycleClient.WithdrawRoleRequest(ctx, p.GroupId.String(), p.RequestId.String(), a); err != nil {
		return fmt.Errorf("withdraw role request: %w", err)
	}
	return nil
}

func (h *ogenHandler) RequestGroupRole(ctx context.Context, q *gatewayapi.CreateRoleRequest, p gatewayapi.RequestGroupRoleParams) (*gatewayapi.RoleRequest, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	role, err := toProtoRoleRequestRole(q.Role)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.RequestRole(ctx, p.GroupId.String(), role, a)
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if _, err := h.metadataLifecycleClient.DeleteCategory(ctx, a, &metadatalifecyclev1.DeleteCategoryRequest{GroupId: params.GroupId.String(), CategoryId: params.CategoryId.String()}); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (h *ogenHandler) DeleteGroup(ctx context.Context, p gatewayapi.DeleteGroupParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if _, err := h.metadataLifecycleClient.DeleteGroup(ctx, a, &metadatalifecyclev1.DeleteGroupRequest{GroupId: p.GroupId.String()}); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}

func (h *ogenHandler) ListGroupMemberships(ctx context.Context, params gatewayapi.ListGroupMembershipsParams) ([]gatewayapi.GroupMembership, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	v, err := h.metadataLifecycleClient.ListGroupMemberships(ctx, a, &metadatalifecyclev1.ListGroupMembershipsRequest{GroupId: params.GroupId.String()})
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	roles := make([]metadatalifecyclev1.GroupRole, len(q.Roles))
	for i := range q.Roles {
		role, err := toProtoMembershipRole(q.Roles[i])
		if err != nil {
			return nil, err
		}
		roles[i] = role
	}
	v, err := h.metadataLifecycleClient.RequestMembership(ctx, params.GroupId.String(), a, roles)
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	decision, err := toProtoDecision(params.Decision)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.DecideMembership(ctx, params.GroupId.String(), params.UserId, decision, a)
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
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if err := h.metadataLifecycleClient.RevokeMembership(ctx, params.GroupId.String(), params.UserId, a); err != nil {
		return fmt.Errorf("revoke membership: %w", err)
	}
	return nil
}
