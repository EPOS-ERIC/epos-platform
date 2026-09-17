package domain

// GlobalPermission is a permission that applies across all groups.
type GlobalPermission string

const (
	GlobalPermissionMembershipManage     GlobalPermission = "global.membership.manage"
	GlobalPermissionRoleManage           GlobalPermission = "global.role.manage"
	GlobalPermissionPermissionManage     GlobalPermission = "global.permission.manage"
	GlobalPermissionOrganizationManage   GlobalPermission = "organization.manage"
	GlobalPermissionContactPointManage   GlobalPermission = "contact_point.manage"
	GlobalPermissionCategorySchemeManage GlobalPermission = "category_scheme.manage"
)

// GroupPermission is a permission evaluated within a group's scope.
type GroupPermission string

const (
	GroupPermissionResourceCreate    GroupPermission = "resource.create"
	GroupPermissionResourceEdit      GroupPermission = "resource.edit"
	GroupPermissionResourceSubmit    GroupPermission = "resource.submit"
	GroupPermissionResourceWithdraw  GroupPermission = "resource.withdraw"
	GroupPermissionResourceReopen    GroupPermission = "resource.reopen"
	GroupPermissionResourceDiscard   GroupPermission = "resource.discard"
	GroupPermissionResourceReject    GroupPermission = "resource.reject"
	GroupPermissionResourcePublish   GroupPermission = "resource.publish"
	GroupPermissionResourceUnpublish GroupPermission = "resource.unpublish"
	GroupPermissionMembershipManage  GroupPermission = "group.membership.manage"
	GroupPermissionRoleManage        GroupPermission = "group.role.manage"
	GroupPermissionPermissionManage  GroupPermission = "group.permission.manage"
	GroupPermissionCategoryManage    GroupPermission = "category.manage"
)

func NewGlobalPermission(value string) (GlobalPermission, error) {
	permission := GlobalPermission(value)
	switch permission {
	case GlobalPermissionMembershipManage,
		GlobalPermissionRoleManage,
		GlobalPermissionPermissionManage,
		GlobalPermissionOrganizationManage,
		GlobalPermissionContactPointManage,
		GlobalPermissionCategorySchemeManage:
		return permission, nil
	default:
		return "", ErrBadRequest
	}
}

func NewGroupPermission(value string) (GroupPermission, error) {
	permission := GroupPermission(value)
	switch permission {
	case GroupPermissionResourceCreate,
		GroupPermissionResourceEdit,
		GroupPermissionResourceSubmit,
		GroupPermissionResourceWithdraw,
		GroupPermissionResourceReopen,
		GroupPermissionResourceDiscard,
		GroupPermissionResourceReject,
		GroupPermissionResourcePublish,
		GroupPermissionResourceUnpublish,
		GroupPermissionMembershipManage,
		GroupPermissionRoleManage,
		GroupPermissionPermissionManage,
		GroupPermissionCategoryManage:
		return permission, nil
	default:
		return "", ErrBadRequest
	}
}
