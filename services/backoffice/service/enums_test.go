package service

import "testing"

func TestEnumNormalizationUsesLowercaseValues(t *testing.T) {
	if got := NormalizeUserStatus(" ACTIVE "); got != UserStatusActive {
		t.Fatalf("NormalizeUserStatus() = %q, want %q", got, UserStatusActive)
	}
	if got := NormalizeMembershipRole("EDITOR"); got != MembershipRoleEditor {
		t.Fatalf("NormalizeMembershipRole() = %q, want %q", got, MembershipRoleEditor)
	}
	if got := NormalizeMembershipStatus("Pending"); got != MembershipStatusPending {
		t.Fatalf("NormalizeMembershipStatus() = %q, want %q", got, MembershipStatusPending)
	}
}
