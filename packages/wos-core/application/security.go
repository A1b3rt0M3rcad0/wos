package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"time"
)

type SecurityService struct {
	Store ports.SecurityStore
	Clock ports.Clock
	IDs   ports.IDGenerator
}

func TokenDigest(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
func (s SecurityService) Authenticate(ctx context.Context, token string) (Identity, error) {
	if len(token) < 32 || len(token) > 512 {
		return Identity{}, domain.NewError(domain.ErrorCodeForbidden, "invalid credential")
	}
	c, err := s.Store.GetCredential(ctx, TokenDigest(token))
	if err != nil || s.validateCredential(ctx, c) != nil {
		return Identity{}, domain.NewError(domain.ErrorCodeForbidden, "invalid or expired credential")
	}
	return Identity{PrincipalID: c.PrincipalID, Actor: c.Actor, NamespaceID: c.NamespaceID, CredentialDigest: c.Digest}, nil
}
func (s SecurityService) Authorize(ctx context.Context, r ports.AuthorizationRequest) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if id, ok := IdentityFromContext(ctx); ok {
		if id.PrincipalID != r.PrincipalID || id.NamespaceID != r.NamespaceID {
			return domain.NewError(domain.ErrorCodeForbidden, "credential scope does not match requested namespace")
		}
		if id.CredentialDigest != "" {
			c, err := s.Store.GetCredential(ctx, id.CredentialDigest)
			if err != nil || s.validateCredential(ctx, c) != nil {
				return domain.NewError(domain.ErrorCodeForbidden, "credential is no longer active")
			}
		}
	}
	grant, err := s.Store.GetGrant(ctx, r.NamespaceID, r.PrincipalID)
	if err != nil {
		return domain.NewError(domain.ErrorCodeForbidden, "namespace permission is not granted")
	}
	for _, p := range grant.Permissions {
		if p == r.Permission {
			return nil
		}
	}
	return domain.NewError(domain.ErrorCodeForbidden, "namespace permission is not granted")
}

func (s SecurityService) validateCredential(ctx context.Context, c ports.Credential) error {
	if c.Revoked || !s.Clock.Now().Before(c.ExpiresAt) {
		return domain.NewError(domain.ErrorCodeForbidden, "credential inactive")
	}
	if c.ParentDigest != "" {
		parent, err := s.Store.GetCredential(ctx, c.ParentDigest)
		if err != nil || parent.ParentDigest != "" || parent.Revoked || !s.Clock.Now().Before(parent.ExpiresAt) || parent.PrincipalID != c.PrincipalID || parent.NamespaceID != c.NamespaceID || parent.Actor != c.Actor {
			return domain.NewError(domain.ErrorCodeForbidden, "session parent inactive")
		}
	}
	return nil
}

// CreateSession narrows an existing credential to a twelve-hour, independently revocable browser session.
func (s SecurityService) CreateSession(ctx context.Context, token string) (ports.Credential, string, error) {
	identity, err := s.Authenticate(ctx, token)
	if err != nil {
		return ports.Credential{}, "", err
	}
	source, err := s.Store.GetCredential(ctx, identity.CredentialDigest)
	if err != nil {
		return ports.Credential{}, "", err
	}
	if source.ParentDigest != "" {
		return ports.Credential{}, "", domain.NewError(domain.ErrorCodeForbidden, "a browser session cannot create another session")
	}
	ctx = WithIdentity(ctx, identity)
	if err = s.Require(ctx, identity.NamespaceID, ports.PermissionStateRead); err != nil {
		return ports.Credential{}, "", err
	}
	id, err := s.IDs.NewID()
	if err != nil {
		return ports.Credential{}, "", err
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return ports.Credential{}, "", err
	}
	session := base64.RawURLEncoding.EncodeToString(raw)
	expires := s.Clock.Now().Add(12 * time.Hour)
	if source.ExpiresAt.Before(expires) {
		expires = source.ExpiresAt
	}
	c := ports.Credential{ID: id, NamespaceID: identity.NamespaceID, PrincipalID: identity.PrincipalID, Actor: identity.Actor, Digest: TokenDigest(session), ParentDigest: source.Digest, ExpiresAt: expires}
	if err = s.Store.PutCredential(ctx, c); err != nil {
		return ports.Credential{}, "", err
	}
	return c, session, nil
}
func (s SecurityService) EndSession(ctx context.Context, token string) error {
	identity, err := s.Authenticate(ctx, token)
	if err != nil {
		return err
	}
	c, err := s.Store.GetCredential(ctx, identity.CredentialDigest)
	if err != nil {
		return err
	}
	if c.ParentDigest == "" {
		return domain.NewError(domain.ErrorCodeForbidden, "credential is not a browser session")
	}
	return s.Store.RevokeCredential(ctx, c.ID)
}
func (s SecurityService) Require(ctx context.Context, ns domain.ID, permission ports.Permission) error {
	id, ok := IdentityFromContext(ctx)
	if !ok {
		return domain.NewError(domain.ErrorCodeForbidden, "authenticated identity is required")
	}
	return s.Authorize(ctx, ports.AuthorizationRequest{NamespaceID: ns, PrincipalID: id.PrincipalID, Permission: permission})
}
func (s SecurityService) IssueCredential(ctx context.Context, namespace domain.ID, principal string, actor domain.ActorRef, expires time.Time) (ports.Credential, string, error) {
	if err := s.Require(ctx, namespace, ports.PermissionNamespaceAdmin); err != nil {
		return ports.Credential{}, "", err
	}
	if identity, ok := IdentityFromContext(ctx); ok && identity.Actor != actor {
		if err := s.Require(ctx, namespace, ports.PermissionActorDelegate); err != nil {
			return ports.Credential{}, "", err
		}
	}
	if strings.TrimSpace(principal) == "" || !expires.After(s.Clock.Now()) || expires.After(s.Clock.Now().Add(366*24*time.Hour)) {
		return ports.Credential{}, "", domain.NewError(domain.ErrorCodeInvalidArgument, "principal and expiration within one year are required")
	}
	if err := actor.Validate(); err != nil {
		return ports.Credential{}, "", err
	}
	// An administrator can issue credentials only for a principal with a grant in its namespace.
	if _, err := s.Store.GetGrant(ctx, namespace, principal); err != nil {
		return ports.Credential{}, "", domain.NewError(domain.ErrorCodeForbidden, "target principal has no namespace grant")
	}
	id, err := s.IDs.NewID()
	if err != nil {
		return ports.Credential{}, "", err
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return ports.Credential{}, "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	c := ports.Credential{NamespaceID: namespace, ID: id, PrincipalID: principal, Actor: actor, ExpiresAt: expires.UTC(), Digest: TokenDigest(token)}
	if err = s.Store.PutCredential(ctx, c); err != nil {
		return ports.Credential{}, "", err
	}
	return c, token, nil
}
func (s SecurityService) SetGrant(ctx context.Context, grant ports.NamespaceGrant) error {
	if err := s.Require(ctx, grant.NamespaceID, ports.PermissionNamespaceAdmin); err != nil {
		return err
	}
	if strings.TrimSpace(grant.PrincipalID) == "" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "principal is required")
	}
	for _, p := range grant.Permissions {
		if !p.Valid() {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "unknown permission")
		}
	}
	return s.Store.PutGrant(ctx, grant)
}
func (s SecurityService) RevokeCredential(ctx context.Context, namespace domain.ID, credential domain.ID) error {
	if err := s.Require(ctx, namespace, ports.PermissionNamespaceAdmin); err != nil {
		return err
	}
	// Store validates the credential's principal belongs to this namespace in ScopedCredentialRevoker.
	scoped, ok := s.Store.(interface {
		RevokeScopedCredential(context.Context, domain.ID, domain.ID) error
	})
	if !ok {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "scoped credential revocation is unavailable")
	}
	return scoped.RevokeScopedCredential(ctx, namespace, credential)
}
func (s SecurityService) Namespaces(ctx context.Context) ([]ports.Namespace, error) {
	id, ok := IdentityFromContext(ctx)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeForbidden, "authenticated identity is required")
	}
	if !id.NamespaceID.IsZero() {
		if err := s.Require(ctx, id.NamespaceID, ports.PermissionStateRead); err != nil {
			return nil, err
		}
	}
	namespaces, err := s.Store.ListNamespaces(ctx, id.PrincipalID)
	if err != nil {
		return nil, err
	}
	if !id.NamespaceID.IsZero() {
		filtered := []ports.Namespace{}
		for _, n := range namespaces {
			if n.ID == id.NamespaceID {
				filtered = append(filtered, n)
			}
		}
		return filtered, nil
	}
	return namespaces, nil
}

// Bootstrap is an operator-only composition action. It creates no credential on subsequent restarts.
func (s SecurityService) Bootstrap(ctx context.Context, namespace ports.Namespace, principal, token string) error {
	if len(token) < 32 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "bootstrap token must contain at least 32 characters")
	}
	if err := namespace.ID.Validate(); err != nil {
		return err
	}
	if namespace.Name == "" {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "bootstrap namespace name is required")
	}
	id, err := s.IDs.NewID()
	if err != nil {
		return err
	}
	permissions := []ports.Permission{ports.PermissionWorkContractAcquire, ports.PermissionWorkContractRevoke, ports.PermissionStateRead, ports.PermissionOutcomeWrite, ports.PermissionPlanningWrite, ports.PermissionWorkWrite, ports.PermissionRecordsWrite, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite, ports.PermissionNamespaceAdmin, ports.PermissionIntegrationWrite, ports.PermissionActorDelegate, ports.PermissionWorkAdminCancel, ports.PermissionWorkAdminComplete, ports.PermissionAssessmentWaive}
	atomic, ok := s.Store.(interface {
		BootstrapSecurity(context.Context, ports.Namespace, ports.NamespaceGrant, ports.Credential) error
	})
	if !ok {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "atomic identity bootstrap is unavailable")
	}
	return atomic.BootstrapSecurity(ctx, namespace, ports.NamespaceGrant{NamespaceID: namespace.ID, PrincipalID: principal, Permissions: permissions}, ports.Credential{NamespaceID: namespace.ID, ID: id, PrincipalID: principal, Actor: domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "wos", ID: principal}, Digest: TokenDigest(token), ExpiresAt: s.Clock.Now().Add(366 * 24 * time.Hour)})
}

func (s SecurityService) AuthorizeInUnitOfWork(ctx context.Context, uow ports.UnitOfWork, request ports.AuthorizationRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok || identity.PrincipalID != request.PrincipalID || identity.NamespaceID != request.NamespaceID {
		return domain.NewError(domain.ErrorCodeForbidden, "credential scope mismatch")
	}
	access, ok := uow.(ports.AccessSnapshotUnitOfWork)
	if !ok {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "transactional access snapshot unavailable")
	}
	return access.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: request, CredentialDigest: identity.CredentialDigest, Actor: identity.Actor, Now: s.Clock.Now().UTC()})
}
