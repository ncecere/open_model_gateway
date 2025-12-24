package rbac

import "github.com/ncecere/open_model_gateway/backend/internal/db"

// Capability represents a tenant-scoped permission that can be granted by role.
type Capability string

const (
	CapabilityManageMemberships      Capability = "memberships.manage"
	CapabilityManageTenantKeys       Capability = "tenant_keys.manage"
	CapabilityManageBillingWebhooks  Capability = "billing_webhooks.manage"
	CapabilityManageBatches          Capability = "batches.manage"
	CapabilityManageMemberBudgets    Capability = "member_budget.manage"
	CapabilityManageTenantLimits     Capability = "tenant_limits.manage"
	CapabilityAttachModels           Capability = "models.attach"
	CapabilityManageTenantGuardrails Capability = "guardrails.manage"
)

var roleCapabilities = map[db.MembershipRole]map[Capability]struct{}{
	db.MembershipRoleOwner: {
		CapabilityManageMemberships:      {},
		CapabilityManageTenantKeys:       {},
		CapabilityManageBillingWebhooks:  {},
		CapabilityManageBatches:          {},
		CapabilityManageMemberBudgets:    {},
		CapabilityManageTenantLimits:     {},
		CapabilityAttachModels:           {},
		CapabilityManageTenantGuardrails: {},
	},
	db.MembershipRoleAdmin: {
		CapabilityManageMemberships:      {},
		CapabilityManageTenantKeys:       {},
		CapabilityManageBillingWebhooks:  {},
		CapabilityManageBatches:          {},
		CapabilityManageMemberBudgets:    {},
		CapabilityManageTenantLimits:     {},
		CapabilityAttachModels:           {},
		CapabilityManageTenantGuardrails: {},
	},
	db.MembershipRoleViewer: {},
	db.MembershipRoleUser:   {},
}

// HasCapability returns true if the role grants the requested capability.
func HasCapability(role db.MembershipRole, capability Capability) bool {
	caps, ok := roleCapabilities[role]
	if !ok {
		return false
	}
	_, ok = caps[capability]
	return ok
}
