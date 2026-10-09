package sharing

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func TestCanManageConfigurations(t *testing.T) {
	service := &Service{}
	for _, test := range []struct {
		name          string
		user          domain.User
		authenticated bool
		want          bool
	}{
		{name: "anonymous", want: false},
		{name: "missing actor", authenticated: true, user: domain.User{UserSummary: domain.UserSummary{GlobalGrantedPermissions: []domain.GlobalPermission{domain.GlobalPermissionConfigurationManage}}}},
		{name: "authenticated without permission", authenticated: true, user: domain.User{UserSummary: domain.UserSummary{ID: "user-1"}}},
		{name: "direct permission", authenticated: true, user: domain.User{UserSummary: domain.UserSummary{ID: "user-1", GlobalGrantedPermissions: []domain.GlobalPermission{domain.GlobalPermissionConfigurationManage}}}, want: true},
		{name: "role permission", authenticated: true, user: domain.User{UserSummary: domain.UserSummary{ID: "user-1", GlobalRolePermissions: []domain.GlobalPermission{domain.GlobalPermissionConfigurationManage}}}, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := service.CanManageConfigurations(test.user, test.authenticated); got != test.want {
				t.Fatalf("CanManageConfigurations() = %v, want %v", got, test.want)
			}
		})
	}
}
