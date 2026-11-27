package domain

import "github.com/EPOS-ERIC/epos-platform/packages/go/apperror"

// GlobalPermission is a permission that applies across all groups.
type GlobalPermission string

const (
	GlobalPermissionMembershipManage    GlobalPermission = "global.membership.manage"
	GlobalPermissionRoleManage          GlobalPermission = "global.role.manage"
	GlobalPermissionPermissionManage    GlobalPermission = "global.permission.manage"
	GlobalPermissionAdminManage         GlobalPermission = "global.admin.manage"
	GlobalPermissionUserBlock           GlobalPermission = "global.user.block"
	GlobalPermissionUserAPITokenRevoke  GlobalPermission = "global.user_api_token.revoke"
	GlobalPermissionNotificationManage  GlobalPermission = "global.notification.manage"
	GlobalPermissionOrganizationManage  GlobalPermission = "global.organization.manage"
	GlobalPermissionContactPointManage  GlobalPermission = "global.contact_point.manage"
	GlobalPermissionGroupManage         GlobalPermission = "global.group.manage"
	GlobalPermissionConceptManage       GlobalPermission = "global.concept.manage"
	GlobalPermissionResourceRead        GlobalPermission = "global.resource.read"
	GlobalPermissionResourceCreate      GlobalPermission = "global.resource.create"
	GlobalPermissionResourceEdit        GlobalPermission = "global.resource.edit"
	GlobalPermissionResourceSubmit      GlobalPermission = "global.resource.submit"
	GlobalPermissionResourceWithdraw    GlobalPermission = "global.resource.withdraw"
	GlobalPermissionResourceReopen      GlobalPermission = "global.resource.reopen"
	GlobalPermissionResourceDiscard     GlobalPermission = "global.resource.discard"
	GlobalPermissionResourceReject      GlobalPermission = "global.resource.reject"
	GlobalPermissionResourcePublish     GlobalPermission = "global.resource.publish"
	GlobalPermissionResourceUnpublish   GlobalPermission = "global.resource.unpublish"
	GlobalPermissionCategoryManage      GlobalPermission = "global.category.manage"
	GlobalPermissionConfigurationManage GlobalPermission = "global.configuration.manage"
	GlobalPermissionConverterManage     GlobalPermission = "global.converter.manage"
	GlobalPermissionMonitoringRead      GlobalPermission = "global.monitoring.read"
)

// GroupPermission is a permission evaluated within a group's scope.
type GroupPermission string

const (
	GroupPermissionResourceCreate     GroupPermission = "group.resource.create"
	GroupPermissionResourceEdit       GroupPermission = "group.resource.edit"
	GroupPermissionResourceSubmit     GroupPermission = "group.resource.submit"
	GroupPermissionResourceWithdraw   GroupPermission = "group.resource.withdraw"
	GroupPermissionResourceReopen     GroupPermission = "group.resource.reopen"
	GroupPermissionResourceDiscard    GroupPermission = "group.resource.discard"
	GroupPermissionResourceReject     GroupPermission = "group.resource.reject"
	GroupPermissionResourcePublish    GroupPermission = "group.resource.publish"
	GroupPermissionResourceUnpublish  GroupPermission = "group.resource.unpublish"
	GroupPermissionMembershipManage   GroupPermission = "group.membership.manage"
	GroupPermissionRoleManage         GroupPermission = "group.role.manage"
	GroupPermissionPermissionManage   GroupPermission = "group.permission.manage"
	GroupPermissionNotificationManage GroupPermission = "group.notification.manage"
	GroupPermissionCategoryManage     GroupPermission = "group.category.manage"
)

func NewGlobalPermission(value string) (GlobalPermission, error) {
	permission := GlobalPermission(value)
	switch permission {
	case GlobalPermissionMembershipManage,
		GlobalPermissionRoleManage,
		GlobalPermissionPermissionManage,
		GlobalPermissionAdminManage,
		GlobalPermissionUserBlock,
		GlobalPermissionResourceRead,
		GlobalPermissionUserAPITokenRevoke,
		GlobalPermissionNotificationManage,
		GlobalPermissionOrganizationManage,
		GlobalPermissionContactPointManage,
		GlobalPermissionGroupManage,
		GlobalPermissionConceptManage,
		GlobalPermissionResourceCreate,
		GlobalPermissionResourceEdit,
		GlobalPermissionResourceSubmit,
		GlobalPermissionResourceWithdraw,
		GlobalPermissionResourceReopen,
		GlobalPermissionResourceDiscard,
		GlobalPermissionResourceReject,
		GlobalPermissionResourcePublish,
		GlobalPermissionResourceUnpublish,
		GlobalPermissionCategoryManage,
		GlobalPermissionConfigurationManage,
		GlobalPermissionConverterManage,
		GlobalPermissionMonitoringRead:
		return permission, nil
	default:
		return "", apperror.Default(apperror.CodeValidationFailed)
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
		GroupPermissionNotificationManage,
		GroupPermissionCategoryManage:
		return permission, nil
	default:
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
}
