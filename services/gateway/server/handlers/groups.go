//nolint:dupl,exhaustruct_v5,wrapcheck
package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ListGroups(ctx context.Context) ([]gatewayapi.Group, error) {
	v, err := h.groups.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []gatewayapi.Group{}
	for _, x := range v {
		mapped, err := mapGroup(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetGroup(ctx context.Context, q gatewayapi.GetGroupParams) (*gatewayapi.Group, error) {
	v, err := h.groups.Get(ctx, q.GroupId.String())
	if err != nil {
		return nil, err
	}
	r, err := mapGroup(v)
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
	in := domain.CreateGroupInput{Name: string(q.Name), Description: fromOptNilString(q.Description), Label: fromOptNilString(q.Label), Logo: fromOptNilString(q.Logo), Page: fromOptNilString(q.Page), Color: fromOptNilString(q.Color), Concept: string(q.Concept)}
	if q.Position.Set {
		in.Position = &q.Position.Value
	}
	v, err := h.groups.Create(ctx, actor, in)
	if err != nil {
		return nil, err
	}
	r, err := mapGroup(v)
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
	in := domain.ReplaceGroupInput{Name: string(q.Name), Description: fromOptNilString(q.Description), Label: fromOptNilString(q.Label), Logo: fromOptNilString(q.Logo), Page: fromOptNilString(q.Page), Color: fromOptNilString(q.Color), Concept: string(q.Concept)}
	if q.Position.Set {
		in.Position = q.Position.Value
	}
	v, err := h.groups.Replace(ctx, a, params.GroupId.String(), in)
	if err != nil {
		return nil, err
	}
	r, err := mapGroup(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
func authMissing() error { return errMissingActor }
func (h *ogenHandler) ListGroupCategories(ctx context.Context, q gatewayapi.ListGroupCategoriesParams) ([]gatewayapi.Category, error) {
	v, err := h.groups.ListCategories(ctx, q.GroupId.String())
	if err != nil {
		return nil, err
	}
	out := []gatewayapi.Category{}
	for _, x := range v {
		mapped, err := mapCategory(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetGroupCategory(ctx context.Context, q gatewayapi.GetGroupCategoryParams) (*gatewayapi.Category, error) {
	v, err := h.groups.GetCategory(ctx, q.GroupId.String(), q.CategoryId.String())
	if err != nil {
		return nil, err
	}
	r, err := mapCategory(v)
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
	in := domain.CreateCategoryInput{Name: string(q.Name)}
	if q.ParentCategoryID.Set && !q.ParentCategoryID.Null {
		s := q.ParentCategoryID.Value.String()
		in.ParentCategoryID = &s
	}
	v, err := h.groups.CreateCategory(ctx, a, params.GroupId.String(), in)
	if err != nil {
		return nil, err
	}
	r, err := mapCategory(v)
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
	in := domain.ReplaceCategoryInput{Name: string(q.Name)}
	if !q.ParentCategoryID.Null {
		s := q.ParentCategoryID.Value.String()
		in.ParentCategoryID = &s
	}
	v, err := h.groups.ReplaceCategory(ctx, a, params.GroupId.String(), params.CategoryId.String(), in)
	if err != nil {
		return nil, err
	}
	r, err := mapCategory(v)
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
	v, err := h.groups.ApplyRoleRequestDecision(ctx, p.GroupId.String(), p.RequestId.String(), string(q.Status), a)
	if err != nil {
		return nil, err
	}
	r, err := mapMembership(v)
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
	return h.groups.AssignGroupRole(ctx, p.GroupId.String(), p.UserId.String(), string(p.Role), a)
}

func (h *ogenHandler) RemoveGroupRole(ctx context.Context, p gatewayapi.RemoveGroupRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return h.groups.RemoveGroupRole(ctx, p.GroupId.String(), p.UserId.String(), string(p.Role), a)
}

func (h *ogenHandler) GrantGroupPermission(ctx context.Context, p gatewayapi.GrantGroupPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return h.groups.GrantGroupPermission(ctx, p.GroupId.String(), p.UserId.String(), string(p.Permission), a)
}

func (h *ogenHandler) RevokeGroupPermission(ctx context.Context, p gatewayapi.RevokeGroupPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return h.groups.RevokeGroupPermission(ctx, p.GroupId.String(), p.UserId.String(), string(p.Permission), a)
}

func (h *ogenHandler) WithdrawGroupMembershipRequest(ctx context.Context, p gatewayapi.WithdrawGroupMembershipRequestParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return h.groups.WithdrawMembershipRequest(ctx, p.GroupId.String(), a)
}

func (h *ogenHandler) WithdrawGroupRoleRequest(ctx context.Context, p gatewayapi.WithdrawGroupRoleRequestParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return h.groups.WithdrawRoleRequest(ctx, p.GroupId.String(), p.RequestId.String(), a)
}

func (h *ogenHandler) RequestGroupRole(ctx context.Context, q *gatewayapi.CreateRoleRequest, p gatewayapi.RequestGroupRoleParams) (*gatewayapi.RoleRequest, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.groups.RequestRole(ctx, p.GroupId.String(), string(q.Role), a)
	if err != nil {
		return nil, err
	}
	r, err := mapRole(v)
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
	if err := h.groups.DeleteCategory(ctx, a, params.GroupId.String(), params.CategoryId.String()); err != nil {
		return err
	}
	return nil
}

func (h *ogenHandler) DeleteGroup(ctx context.Context, p gatewayapi.DeleteGroupParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.groups.Delete(ctx, a, p.GroupId.String()); err != nil {
		return err
	}
	return nil
}

func (h *ogenHandler) ListGroupMemberships(ctx context.Context, params gatewayapi.ListGroupMembershipsParams) ([]gatewayapi.GroupMembership, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.groups.ListMemberships(ctx, params.GroupId.String(), a)
	if err != nil {
		return nil, err
	}
	out := []gatewayapi.GroupMembership{}
	for _, x := range v {
		mapped, err := mapMembership(x)
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
	v, err := h.groups.RequestMembership(ctx, params.GroupId.String(), a, roles)
	if err != nil {
		return nil, err
	}
	r, err := mapMembership(v)
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
	v, err := h.groups.DecideMembership(ctx, params.GroupId.String(), params.UserId.String(), a, string(q.Status))
	if err != nil {
		return nil, err
	}
	r, err := mapMembership(v)
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
	if err := h.groups.RevokeMembership(ctx, params.GroupId.String(), params.UserId.String(), a); err != nil {
		return err
	}
	return nil
}
