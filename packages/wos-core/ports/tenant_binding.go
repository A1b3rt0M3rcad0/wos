package ports

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"

// TenantBinding identifies a store whose physical storage is dedicated to one
// Tenant. A zero ID denotes an unbound embedded store. Provisioning additional
// Tenants belongs to installation operations, never to a bound child store.
type TenantBinding interface {
	BoundTenantID() domain.ID
}
