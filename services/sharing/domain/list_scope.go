package domain

import "github.com/EPOS-ERIC/epos-platform/packages/go/apperror"

type ListScope string

const (
	ListScopeOwn      ListScope = "OWN"
	ListScopeOwner    ListScope = "OWNER"
	ListScopeAll      ListScope = "ALL"
	ListScopeOfficial ListScope = "OFFICIAL"
)

func NewListScope(value string) (ListScope, error) {
	scope := ListScope(value)
	switch scope {
	case ListScopeOwn, ListScopeOwner, ListScopeAll, ListScopeOfficial:
		return scope, nil
	default:
		return "", apperror.Field("scope", "Select own, owner, all, or official configurations.", nil)
	}
}
