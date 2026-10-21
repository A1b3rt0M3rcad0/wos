package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"time"
)

func (s SecurityService) AdministrativeSnapshot(ctx context.Context, ns domain.ID) (ports.SecurityAdminSnapshot, error) {
	if err := s.Require(ctx, ns, ports.PermissionCredentialsRead); err != nil {
		return ports.SecurityAdminSnapshot{}, err
	}
	if err := s.Require(ctx, ns, ports.PermissionAuditRead); err != nil {
		return ports.SecurityAdminSnapshot{}, err
	}
	store, ok := s.Store.(ports.SecurityAdministrationStore)
	if !ok {
		return ports.SecurityAdminSnapshot{}, domain.NewError(domain.ErrorCodeInvalidConfig, "security administration unavailable")
	}
	return store.ReadSecurityAdmin(ctx, ns)
}

// AdministrativeMutation persists only hashed credentials. A replay returns a receipt;
// a lost initial secret must be revoked and replaced, never recovered from storage.
func (s SecurityService) AdministrativeMutation(ctx context.Context, key string, cmd ports.SecurityAdminCommand) (ports.SecurityAdminResult, string, error) {
	var empty ports.SecurityAdminResult
	permission, err := securityAdminPermission(cmd.Operation)
	if err != nil {
		return empty, "", err
	}
	if err := s.Require(ctx, cmd.NamespaceID, permission); err != nil {
		return empty, "", err
	}
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return empty, "", err
	}
	if cmd.ExpectedVersion < 1 {
		return empty, "", domain.NewError(domain.ErrorCodeInvalidVersion, "expected_namespace_version is required")
	}
	id, _ := IdentityFromContext(ctx)
	now := s.Clock.Now().UTC()
	// Hash intent before creating ephemeral credential identity/secret.
	intent := map[string]any{
		"namespace_id": cmd.NamespaceID, "expected_namespace_version": cmd.ExpectedVersion,
		"operation": cmd.Operation, "grant": cmd.Grant, "namespace_name": cmd.NamespaceName,
		"credential_name": cmd.CredentialName, "credential_description": cmd.CredentialDescription,
		"role_template_id": cmd.RoleTemplateID, "role_template_version": cmd.RoleTemplateVersion,
		"permissions": cmd.CredentialPermissions, "allowed_outcome_ids": cmd.AllowedOutcomeIDs,
		"allowed_signing_key_ids": cmd.AllowedSigningKeyIDs,
	}
	if cmd.Credential != nil {
		c := cmd.Credential
		intent["credential"] = map[string]any{"namespace_id": c.NamespaceID, "principal_id": c.PrincipalID, "actor_ref": c.Actor, "expires_at": c.ExpiresAt}
	}
	if !cmd.CredentialID.IsZero() {
		intent["credential_id"] = cmd.CredentialID
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, "", err
	}
	hash := sha256.Sum256(raw)
	store, ok := s.Store.(ports.SecurityAdministrationStore)
	if !ok {
		return empty, "", domain.NewError(domain.ErrorCodeInvalidConfig, "security administration unavailable")
	}
	request := ports.SecurityAdminRequest{Command: cmd, PrincipalID: id.PrincipalID, Actor: id.Actor, CredentialDigest: id.CredentialDigest, IdempotencyKey: key, Fingerprint: hex.EncodeToString(hash[:]), Now: now}
	if cmd.Operation == "issue_credential" {
		// A receipt is independent of mutable target state. Check it before
		// reading the target Principal's current grant or generating a new secret.
		// The adapter still revalidates the caller's current permission/session.
		if replay, found, readErr := store.ReadSecurityAdminReceipt(ctx, request); readErr != nil {
			return empty, "", readErr
		} else if found {
			return replay, "", nil
		}
	}
	token := ""
	switch cmd.Operation {
	case "create_namespace":
		if strings.TrimSpace(cmd.NamespaceName) == "" || len(cmd.NamespaceName) > 256 {
			return empty, "", domain.NewError(domain.ErrorCodeInvalidArgument, "namespace name is required and limited to 256 bytes")
		}
		ns, err := s.IDs.NewID()
		if err != nil {
			return empty, "", err
		}
		credentialID, err := s.IDs.NewID()
		if err != nil {
			return empty, "", err
		}
		secret := make([]byte, 32)
		if _, err = rand.Read(secret); err != nil {
			return empty, "", err
		}
		token = base64.RawURLEncoding.EncodeToString(secret)
		cmd.NewNamespace = &ports.Namespace{ID: ns, Name: strings.TrimSpace(cmd.NamespaceName)}
		cmd.Credential = &ports.Credential{NamespaceID: ns, ID: credentialID, PrincipalID: id.PrincipalID, Actor: id.Actor, ExpiresAt: now.Add(12 * time.Hour), Digest: TokenDigest(token)}
		grant, err := s.Store.GetGrant(ctx, cmd.NamespaceID, id.PrincipalID)
		if err != nil {
			return empty, "", err
		}
		grant.NamespaceID = ns
		prepared, prepareErr := PrepareCredentialIssuance(CredentialIssuanceSelection{NamespaceID: ns, CredentialID: credentialID, PrincipalID: id.PrincipalID, Actor: id.Actor, ExpiresAt: cmd.Credential.ExpiresAt, Name: "Initial administrator credential", CreatedByPrincipalID: id.PrincipalID, CreatedByActor: id.Actor}, grant, now)
		if prepareErr != nil {
			return empty, "", prepareErr
		}
		cmd.CredentialPolicy = &prepared.Policy
		cmd.CredentialMetadata = &prepared.Metadata
	case "set_grant":
		if cmd.Grant == nil || cmd.Grant.NamespaceID != cmd.NamespaceID || strings.TrimSpace(cmd.Grant.PrincipalID) == "" || len(cmd.Grant.PrincipalID) > 256 {
			return empty, "", domain.NewError(domain.ErrorCodeInvalidArgument, "invalid namespace grant")
		}
		if err := ports.ValidatePermissionValues(cmd.Grant.Permissions); err != nil {
			return empty, "", err
		}
	case "issue_credential":
		c := cmd.Credential
		if c == nil || !c.ID.IsZero() || c.Revoked || c.NamespaceID != cmd.NamespaceID || strings.TrimSpace(c.PrincipalID) == "" || len(c.PrincipalID) > 256 || !c.ExpiresAt.After(now) || c.ExpiresAt.After(now.Add(366*24*time.Hour)) {
			return empty, "", domain.NewError(domain.ErrorCodeInvalidArgument, "invalid credential intent")
		}
		if err = c.Actor.Validate(); err != nil {
			return empty, "", err
		}
		if c.Actor != id.Actor {
			if err = s.Require(ctx, cmd.NamespaceID, ports.PermissionActorDelegate); err != nil {
				return empty, "", err
			}
		}
		generated, err := s.IDs.NewID()
		if err != nil {
			return empty, "", err
		}
		secret := make([]byte, 32)
		if _, err = rand.Read(secret); err != nil {
			return empty, "", err
		}
		token = base64.RawURLEncoding.EncodeToString(secret)
		prepared := *c
		prepared.ID = generated
		prepared.Digest = TokenDigest(token)
		prepared.ParentDigest = ""
		prepared.Revoked = false
		cmd.Credential = &prepared
		grant, err := s.Store.GetGrant(ctx, c.NamespaceID, c.PrincipalID)
		if err != nil {
			return empty, "", domain.NewError(domain.ErrorCodeForbidden, "target principal has no namespace grant")
		}
		issuance, prepareErr := PrepareCredentialIssuance(CredentialIssuanceSelection{
			NamespaceID: c.NamespaceID, CredentialID: prepared.ID, PrincipalID: c.PrincipalID, Actor: c.Actor,
			ExpiresAt: c.ExpiresAt, Name: cmd.CredentialName, Description: cmd.CredentialDescription,
			TemplateID: cmd.RoleTemplateID, TemplateVersion: cmd.RoleTemplateVersion,
			EditedPermissions: cmd.CredentialPermissions, AllowedOutcomeIDs: cmd.AllowedOutcomeIDs,
			AllowedSigningKeyIDs: cmd.AllowedSigningKeyIDs, CreatedByPrincipalID: id.PrincipalID, CreatedByActor: id.Actor,
		}, grant, now)
		if prepareErr != nil {
			return empty, "", prepareErr
		}
		cmd.CredentialPolicy = &issuance.Policy
		cmd.CredentialMetadata = &issuance.Metadata
	case "revoke_credential":
		if err := cmd.CredentialID.Validate(); err != nil {
			return empty, "", err
		}
	default:
		return empty, "", domain.NewError(domain.ErrorCodeInvalidArgument, "unknown security operation")
	}
	request.Command = cmd
	result, err := store.ApplySecurityAdmin(ctx, request)
	if err != nil || result.IdempotentReplay {
		token = ""
	}
	return result, token, err
}

func securityAdminPermission(operation string) (ports.Permission, error) {
	permission, ok := ports.SecurityAdminPermission(operation)
	if !ok {
		return "", domain.NewError(domain.ErrorCodeInvalidArgument, "unknown security operation")
	}
	return permission, nil
}

// AdministrativeIntent is the shared flat HTTP/MCP administration contract.
type AdministrativeIntent struct {
	NamespaceID           domain.ID                  `json:"namespace_id"`
	ExpectedVersion       domain.Version             `json:"expected_namespace_version"`
	Operation             string                     `json:"operation"`
	PrincipalID           string                     `json:"principal_id,omitempty"`
	Permissions           []ports.Permission         `json:"permissions,omitempty"`
	CredentialName        string                     `json:"credential_name,omitempty"`
	CredentialDescription string                     `json:"credential_description,omitempty"`
	RoleTemplateID        ports.PermissionTemplateID `json:"role_template_id,omitempty"`
	RoleTemplateVersion   uint32                     `json:"role_template_version,omitempty"`
	AllowedOutcomeIDs     []domain.ID                `json:"allowed_outcome_ids,omitempty"`
	AllowedSigningKeyIDs  []domain.ID                `json:"allowed_signing_key_ids,omitempty"`
	Actor                 *domain.ActorRef           `json:"actor_ref,omitempty"`
	ExpiresAt             *time.Time                 `json:"expires_at,omitempty"`
	CredentialID          domain.ID                  `json:"credential_id,omitempty"`
	NamespaceName         string                     `json:"namespace_name,omitempty"`
}

func (s SecurityService) Administer(ctx context.Context, key string, intent AdministrativeIntent) (ports.SecurityAdminResult, string, error) {
	cmd := ports.SecurityAdminCommand{NamespaceID: intent.NamespaceID, ExpectedVersion: intent.ExpectedVersion, Operation: intent.Operation, CredentialID: intent.CredentialID, NamespaceName: intent.NamespaceName, CredentialName: intent.CredentialName, CredentialDescription: intent.CredentialDescription, RoleTemplateID: intent.RoleTemplateID, RoleTemplateVersion: intent.RoleTemplateVersion, AllowedOutcomeIDs: intent.AllowedOutcomeIDs, AllowedSigningKeyIDs: intent.AllowedSigningKeyIDs}
	if intent.Operation == "set_grant" {
		cmd.Grant = &ports.NamespaceGrant{NamespaceID: intent.NamespaceID, PrincipalID: intent.PrincipalID, Permissions: intent.Permissions}
	}
	if intent.Operation == "issue_credential" {
		if intent.Actor == nil || intent.ExpiresAt == nil {
			return ports.SecurityAdminResult{}, "", domain.NewError(domain.ErrorCodeInvalidArgument, "actor_ref and expires_at required")
		}
		cmd.Credential = &ports.Credential{NamespaceID: intent.NamespaceID, PrincipalID: intent.PrincipalID, Actor: *intent.Actor, ExpiresAt: *intent.ExpiresAt}
		cmd.CredentialPermissions = intent.Permissions
	}
	return s.AdministrativeMutation(ctx, key, cmd)
}
