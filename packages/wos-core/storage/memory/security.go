package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

func (s *Store) GetCredential(ctx context.Context, digest string) (ports.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.security.Credentials[digest]
	if !ok {
		return c, missingSigning()
	}
	return c, ctx.Err()
}
func (s *Store) PutCredential(ctx context.Context, c ports.Credential) error {
	if c.ID.Validate() != nil || c.TenantID.Validate() != nil || c.Actor.Validate() != nil || c.PrincipalID == "" || c.Digest == "" {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid credential")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.security.Tenants[c.TenantID]; !ok {
		return missingSigning()
	}
	if _, ok := s.security.Credentials[c.Digest]; ok {
		return d.NewError(d.ErrorCodeAlreadyExists, "credential exists")
	}
	for _, other := range s.security.Credentials {
		if c.ID == other.ID {
			return d.NewError(d.ErrorCodeAlreadyExists, "credential ID exists")
		}
	}
	s.security.Credentials[c.Digest] = c
	return ctx.Err()
}
func (s *Store) RevokeCredential(ctx context.Context, id d.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for digest, c := range s.security.Credentials {
		if c.ID == id {
			c.Revoked = true
			s.security.Credentials[digest] = c
			return ctx.Err()
		}
	}
	return missingSigning()
}
func (s *Store) RevokeScopedCredential(ctx context.Context, ns, id d.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for digest, c := range s.security.Credentials {
		if c.ID == id && c.TenantID == ns {
			c.Revoked = true
			s.security.Credentials[digest] = c
			return ctx.Err()
		}
	}
	return missingSigning()
}

func (s *Store) ScopedCredentialPrincipal(ctx context.Context, ns, id d.ID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.security.Credentials {
		if c.ID == id && c.TenantID == ns {
			return c.PrincipalID, ctx.Err()
		}
	}
	return "", missingSigning()
}
func (s *Store) GetGrant(ctx context.Context, ns d.ID, principal string) (ports.TenantGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.security.Grants[securityKey(ns, principal)]
	if !ok {
		return g, missingSigning()
	}
	return deepCopy(g), ctx.Err()
}
func (s *Store) PutGrant(ctx context.Context, g ports.TenantGrant) error {
	if g.TenantID.Validate() != nil || g.PrincipalID == "" {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid grant")
	}
	for _, p := range g.Permissions {
		if !p.Valid() {
			return d.NewError(d.ErrorCodeInvalidArgument, "invalid permission")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.security.Tenants[g.TenantID]; !ok {
		return missingSigning()
	}
	s.security.Grants[securityKey(g.TenantID, g.PrincipalID)] = deepCopy(g)
	return ctx.Err()
}
func (s *Store) PutTenant(ctx context.Context, n ports.Tenant) error {
	if n.ID.Validate() != nil || n.Name == "" {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid tenant")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.security.Tenants[n.ID]; ok {
		return d.NewError(d.ErrorCodeAlreadyExists, "tenant exists")
	}
	s.security.Tenants[n.ID] = n
	s.security.Versions[n.ID] = 1
	return ctx.Err()
}
func (s *Store) ListTenants(ctx context.Context, principal string) ([]ports.Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ports.Tenant{}
	for ns, n := range s.security.Tenants {
		if _, ok := s.security.Grants[securityKey(ns, principal)]; ok {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, ctx.Err()
}
func (s *Store) BootstrapSecurity(ctx context.Context, n ports.Tenant, g ports.TenantGrant, c ports.Credential) error {
	if n.ID.Validate() != nil || g.TenantID != n.ID || c.TenantID != n.ID || g.PrincipalID != c.PrincipalID || c.Actor.Validate() != nil || c.ID.Validate() != nil {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid bootstrap binding")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.security.Tenants[n.ID]; ok {
		return ctx.Err()
	}
	s.security.Tenants[n.ID] = n
	s.security.Versions[n.ID] = 1
	s.security.Grants[securityKey(n.ID, g.PrincipalID)] = deepCopy(g)
	s.security.Credentials[c.Digest] = c
	return ctx.Err()
}
func (tx *transaction) AuthorizeAccessSnapshot(ctx context.Context, r ports.AccessSnapshotRequest) error {
	denied := d.NewError(d.ErrorCodeForbidden, "credential or permission inactive")
	if _, ok := tx.security.Tenants[r.Authorization.TenantID]; !ok {
		return denied
	}
	grant, ok := tx.security.Grants[securityKey(r.Authorization.TenantID, r.Authorization.PrincipalID)]
	if !ok {
		return denied
	}
	allowed := false
	for _, p := range grant.Permissions {
		allowed = allowed || p == r.Authorization.Permission
	}
	if !allowed {
		return denied
	}
	c, ok := tx.security.Credentials[r.CredentialDigest]
	if !ok || c.Revoked || !r.Now.Before(c.ExpiresAt) || c.TenantID != r.Authorization.TenantID || c.PrincipalID != r.Authorization.PrincipalID || c.Actor != r.Actor {
		return denied
	}
	if c.ParentDigest != "" {
		parent, ok := tx.security.Credentials[c.ParentDigest]
		if !ok || parent.ParentDigest != "" || parent.Revoked || !r.Now.Before(parent.ExpiresAt) || parent.TenantID != c.TenantID || parent.PrincipalID != c.PrincipalID || parent.Actor != c.Actor {
			return denied
		}
		c = parent
	}
	if policy, ok := tx.security.Policies[securityKey(c.TenantID, c.ID.String())]; ok {
		if !policy.Permits(string(r.Authorization.Permission), r.Authorization.OutcomeID) || (r.Authorization.Permission.InstallationScoped() && len(policy.AllowedOutcomeIDs) > 0) {
			return denied
		}
	}
	return ctx.Err()
}

var _ ports.SecurityStore = (*Store)(nil)
var _ ports.AccessSnapshotUnitOfWork = (*transaction)(nil)

func (s *Store) CredentialPolicy(ctx context.Context, ns, id d.ID) (d.CredentialPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.security.Policies[securityKey(ns, id.String())]
	if !ok {
		return p, missingSigning()
	}
	return deepCopy(p), ctx.Err()
}
