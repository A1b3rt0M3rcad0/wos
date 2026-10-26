package domain

// MaxPermissionValues bounds explicit permission lists in grants and
// CredentialPolicy values. Empty lists are valid and grant no authority.
const MaxPermissionValues = 128

var permissionOperationCatalog = [...]string{
	// Published permissions retained for stored grants and policies. They remain
	// exact values until the separately versioned compatibility work is applied.
	"work.contract.return", "work.contract.complete_direct", "work.review.acquire", "work.review.decide",
	"identity.signing_key.enroll", "identity.signing_key.rotate", "identity.signing_key.revoke",
	"work.contract.acquire", "work.contract.revoke", "state:read", "outcome:write", "planning:write",
	"work:write", "records:write", "assessment:write", "conclusion:write", "namespace:admin",
	"integration:write", "actor:delegate", "work:admin_cancel", "work:admin_complete", "assessment:waive",
	// Granular operational permissions.
	"credentials:read", "audit:read",
	"objective:create", "objective:update", "objective:start", "objective:conclude", "objective:reopen",
	"objective:cancel", "objective:configure_completion",
	"task:create", "task:update_definition", "task:activate", "task:defer", "task:assign",
	"task:claim", "task:renew", "task:release", "task:deliver", "task:review", "task:accept",
	"task:request_changes", "task:withdraw_delivery", "task:cancel", "task:reopen", "task:intervene",
	"criterion:define", "criterion:assess", "criterion:waive", "dependency:write",
	"issue:create", "issue:update", "issue:assign", "issue:investigate", "issue:resolve", "issue:reopen",
	"blocker:create", "blocker:release", "blocker:cancel",
	"credentials:issue", "credentials:update", "credentials:revoke", "credentials:rotate",
	"permissions:manage", "policy:manage",
}

// PermissionOperationNames returns the explicit catalog without exposing its
// backing storage.
func PermissionOperationNames() []string {
	return append([]string(nil), permissionOperationCatalog[:]...)
}

// PermissionOperationValid reports whether name is a registered permission.
func PermissionOperationValid(name string) bool {
	for _, permission := range permissionOperationCatalog {
		if name == permission {
			return true
		}
	}
	return false
}
