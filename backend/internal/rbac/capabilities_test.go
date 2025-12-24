package rbac

import (
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

func TestRoleCapabilitiesCoverage(t *testing.T) {
	expectedRoles := []db.MembershipRole{
		db.MembershipRoleOwner,
		db.MembershipRoleAdmin,
		db.MembershipRoleViewer,
		db.MembershipRoleUser,
	}

	if len(roleCapabilities) != len(expectedRoles) {
		t.Fatalf("expected %d role capability entries, got %d", len(expectedRoles), len(roleCapabilities))
	}
	if len(roleOrder) != len(expectedRoles) {
		t.Fatalf("expected %d role order entries, got %d", len(expectedRoles), len(roleOrder))
	}

	for _, role := range expectedRoles {
		if _, ok := roleCapabilities[role]; !ok {
			t.Fatalf("missing role capability mapping for %q", role)
		}
		if _, ok := roleOrder[role]; !ok {
			t.Fatalf("missing role order mapping for %q", role)
		}
	}
}

func TestRoleCapabilitiesMatrix(t *testing.T) {
	allCaps := []Capability{
		CapabilityManageMemberships,
		CapabilityManageTenantKeys,
		CapabilityManageBillingWebhooks,
		CapabilityManageBatches,
		CapabilityManageMemberBudgets,
		CapabilityManageTenantLimits,
		CapabilityAttachModels,
		CapabilityManageTenantGuardrails,
	}

	for _, cap := range allCaps {
		if !HasCapability(db.MembershipRoleOwner, cap) {
			t.Fatalf("expected owner to have %q", cap)
		}
		if !HasCapability(db.MembershipRoleAdmin, cap) {
			t.Fatalf("expected admin to have %q", cap)
		}
		if HasCapability(db.MembershipRoleViewer, cap) {
			t.Fatalf("expected viewer to lack %q", cap)
		}
		if HasCapability(db.MembershipRoleUser, cap) {
			t.Fatalf("expected user to lack %q", cap)
		}
	}
}
