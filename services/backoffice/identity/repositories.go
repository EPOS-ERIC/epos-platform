package identity

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/google/uuid"
)

type UserRepository interface {
	ListUsers(context.Context) ([]sqlc.ListUsersRow, error)
	GetUserByID(context.Context, uuid.UUID) (sqlc.GetUserByIDRow, error)
	GetUserByAuthIdentifier(context.Context, string) (sqlc.GetUserByAuthIdentifierRow, error)
	CreateUser(context.Context, sqlc.CreateUserParams) (sqlc.CreateUserRow, error)
	UpdateUser(context.Context, sqlc.UpdateUserParams) (sqlc.UpdateUserRow, error)
	DeleteUser(context.Context, uuid.UUID) error
}

type GroupRepository interface {
	ListGroups(context.Context) ([]sqlc.ListGroupsRow, error)
	GetGroupByID(context.Context, uuid.UUID) (sqlc.GetGroupByIDRow, error)
	CreateGroup(context.Context, sqlc.CreateGroupParams) (sqlc.CreateGroupRow, error)
	UpdateGroup(context.Context, sqlc.UpdateGroupParams) (sqlc.UpdateGroupRow, error)
	DeleteGroup(context.Context, uuid.UUID) error
}

type MembershipRepository interface {
	GetGroupByID(context.Context, uuid.UUID) (sqlc.GetGroupByIDRow, error)
	ListMembershipsByGroupID(context.Context, uuid.UUID) ([]sqlc.GroupMembership, error)
	GetUserByID(context.Context, uuid.UUID) (sqlc.GetUserByIDRow, error)
	UpsertMembership(context.Context, sqlc.UpsertMembershipParams) (sqlc.GroupMembership, error)
	GetMembershipByIDs(context.Context, sqlc.GetMembershipByIDsParams) (sqlc.GroupMembership, error)
	DeleteMembership(context.Context, sqlc.DeleteMembershipParams) error
	IsActiveMember(context.Context, sqlc.IsActiveMemberParams) (bool, error)
}
