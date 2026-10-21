package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Identity is resolved by a trusted adapter, never decoded from a domain payload.
type Identity struct {
	CredentialID             domain.ID
	CredentialPolicyRevision domain.Version
	PrincipalID              string
	NamespaceID              domain.ID
	CredentialDigest         string
	Actor                    domain.ActorRef
}
type identityKey struct{}

func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}

// NewAuthorizedService requires a trusted request identity for every public command and query.
// NewService remains the explicit trusted embedded composition used by local applications.
func NewAuthorizedService(tx ports.TransactionManager, clock ports.Clock, ids ports.IDGenerator, auth ports.Authorizer) (*Service, error) {
	s, err := NewServiceWithAuthorizer(tx, clock, ids, auth)
	if err != nil {
		return nil, err
	}
	s.requireIdentity = true
	return s, nil
}
func (s *Service) Authorize(ctx context.Context, namespace domain.ID, permission ports.Permission) error {
	return s.authorizePermission(ctx, ports.AuthorizationRequest{NamespaceID: namespace, PrincipalID: identityPrincipal(ctx), Permission: permission})
}

func identityPrincipal(ctx context.Context) string {
	id, _ := IdentityFromContext(ctx)
	return id.PrincipalID
}

func (s *Service) authorizePermission(ctx context.Context, request ports.AuthorizationRequest) error {
	id, ok := IdentityFromContext(ctx)
	if !ok || id.PrincipalID == "" || request.PrincipalID != id.PrincipalID {
		return domain.NewError(domain.ErrorCodeForbidden, "authenticated identity is required")
	}
	return s.authorizer.Authorize(ctx, request)
}
func (s *Service) authorizeRead(ctx context.Context, namespace domain.ID) error {
	if err := s.authorizeNamespaceMetadataRead(ctx, namespace); err != nil {
		return err
	}
	if !s.requireIdentity {
		return nil
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer u.Rollback()
	allowed, err := s.readOutcomeRestrictions(ctx, u, namespace)
	if err != nil {
		return err
	}
	if len(allowed) != 0 {
		return domain.NewError(domain.ErrorCodeForbidden, "Outcome-restricted credential requires a scoped read")
	}
	return nil
}

func (s *Service) authorizeMutation(ctx context.Context, cc domain.CommandContext, meta commandMetadata) error {
	// Resolve authorization before the trusted-embedded shortcut too, so an
	// unregistered application command cannot inherit implicit authority.
	requirement, registered := commandAuthorizationRequirement(meta.Name)
	if !registered {
		return domain.NewError(domain.ErrorCodeForbidden, "command authorization requirement is not registered")
	}
	if !s.requireIdentity {
		return nil
	}
	id, ok := IdentityFromContext(ctx)
	if !ok || id.PrincipalID != cc.PrincipalID || id.Actor != cc.Actor {
		return domain.NewError(domain.ErrorCodeForbidden, "command identity does not match authenticated identity")
	}
	return s.authorizePermission(ctx, ports.AuthorizationRequest{
		NamespaceID: meta.NamespaceID, PrincipalID: id.PrincipalID, Permission: requirement.PrimaryPermission,
		LegacyPermission: legacyPermissionForCommand(meta.Name, requirement.PrimaryPermission),
	})
}
func commandPermission(name string) ports.Permission {
	requirement, ok := commandAuthorizationRequirement(name)
	if !ok {
		return ""
	}
	return requirement.PrimaryPermission
}

// CommandPermissionForPresentation exposes existing authorization metadata only.
// It neither grants authority nor replaces command authorization.
func CommandPermissionForPresentation(name string) ports.Permission { return commandPermission(name) }
