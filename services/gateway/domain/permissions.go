package domain

// GlobalPermission is a permission that applies across all groups.
type GlobalPermission string

const (
	GlobalPermissionMembershipManage     GlobalPermission = "global.membership.manage"
	GlobalPermissionRoleManage           GlobalPermission = "global.role.manage"
	GlobalPermissionPermissionManage     GlobalPermission = "global.permission.manage"
	GlobalPermissionOrganizationManage   GlobalPermission = "global.organization.manage"
	GlobalPermissionContactPointManage   GlobalPermission = "global.contact_point.manage"
	GlobalPermissionCategorySchemeManage GlobalPermission = "global.category_scheme.manage"
	GlobalPermissionResourceCreate       GlobalPermission = "global.resource.create"
	GlobalPermissionResourceEdit         GlobalPermission = "global.resource.edit"
	GlobalPermissionResourceSubmit       GlobalPermission = "global.resource.submit"
	GlobalPermissionResourceWithdraw     GlobalPermission = "global.resource.withdraw"
	GlobalPermissionResourceReopen       GlobalPermission = "global.resource.reopen"
	GlobalPermissionResourceDiscard      GlobalPermission = "global.resource.discard"
	GlobalPermissionResourceReject       GlobalPermission = "global.resource.reject"
	GlobalPermissionResourcePublish      GlobalPermission = "global.resource.publish"
	GlobalPermissionResourceUnpublish    GlobalPermission = "global.resource.unpublish"
	GlobalPermissionCategoryManage       GlobalPermission = "global.category.manage"
)

// GroupPermission is a permission evaluated within a group's scope.
type GroupPermission string

const (
	GroupPermissionResourceCreate    GroupPermission = "group.resource.create"
	GroupPermissionResourceEdit      GroupPermission = "group.resource.edit"
	GroupPermissionResourceSubmit    GroupPermission = "group.resource.submit"
	GroupPermissionResourceWithdraw  GroupPermission = "group.resource.withdraw"
	GroupPermissionResourceReopen    GroupPermission = "group.resource.reopen"
	GroupPermissionResourceDiscard   GroupPermission = "group.resource.discard"
	GroupPermissionResourceReject    GroupPermission = "group.resource.reject"
	GroupPermissionResourcePublish   GroupPermission = "group.resource.publish"
	GroupPermissionResourceUnpublish GroupPermission = "group.resource.unpublish"
	GroupPermissionMembershipManage  GroupPermission = "group.membership.manage"
	GroupPermissionRoleManage        GroupPermission = "group.role.manage"
	GroupPermissionPermissionManage  GroupPermission = "group.permission.manage"
	GroupPermissionCategoryManage    GroupPermission = "group.category.manage"
)

func NewGlobalPermission(value string) (GlobalPermission, error) {
	permission := GlobalPermission(value)
	switch permission {
	case GlobalPermissionMembershipManage,
		GlobalPermissionRoleManage,
		GlobalPermissionPermissionManage,
		GlobalPermissionOrganizationManage,
		GlobalPermissionContactPointManage,
		GlobalPermissionCategorySchemeManage,
		GlobalPermissionResourceCreate,
		GlobalPermissionResourceEdit,
		GlobalPermissionResourceSubmit,
		GlobalPermissionResourceWithdraw,
		GlobalPermissionResourceReopen,
		GlobalPermissionResourceDiscard,
		GlobalPermissionResourceReject,
		GlobalPermissionResourcePublish,
		GlobalPermissionResourceUnpublish,
		GlobalPermissionCategoryManage:
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
