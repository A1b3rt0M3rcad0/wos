package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Store) ReadPermissionMigration(ctx context.Context, r ports.PermissionMigrationRead) (ports.PermissionMigrationSnapshot, error) {
	var out ports.PermissionMigrationSnapshot
	if err := validatePermissionMigrationRead(r); err != nil {
		return out, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true})
	if err != nil {
		return out, mapSQLError("begin permission migration preview", err)
	}
	defer tx.Rollback()
	var version int64
	if err = tx.QueryRowContext(ctx, `SELECT n.version FROM namespaces AS n WHERE n.id=? AND n.lifecycle='active'`, r.NamespaceID.String()).Scan(&version); err != nil {
		return out, mapSQLError("read permission migration namespace", err)
	}
	var raw string
	var compatibility uint32
	if err = tx.QueryRowContext(ctx, `SELECT permissions_json,permission_compatibility_version FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, r.NamespaceID.String(), r.PrincipalID).Scan(&raw, &compatibility); err != nil {
		return out, mapSQLError("read permission migration grant", err)
	}
	out.NamespaceVersion = domain.Version(version)
	out.Grant = ports.NamespaceGrant{NamespaceID: r.NamespaceID, PrincipalID: r.PrincipalID, PermissionCompatibilityVersion: compatibility}
	if err = unmarshalJSON(raw, &out.Grant.Permissions); err != nil {
		return ports.PermissionMigrationSnapshot{}, err
	}
	if r.CredentialID.IsZero() {
		return out, tx.Commit()
	}
	credential, err := permissionMigrationCredential(ctx, tx, r.NamespaceID, r.PrincipalID, r.CredentialID)
	if err != nil {
		return ports.PermissionMigrationSnapshot{}, err
	}
	out.Credential = &credential
	var policyJSON string
	err = tx.QueryRowContext(ctx, `SELECT state_json FROM credential_policies WHERE namespace_id=? AND credential_id=?`, r.NamespaceID.String(), r.CredentialID.String()).Scan(&policyJSON)
	if err == nil {
		var policy domain.CredentialPolicy
		if err = json.Unmarshal([]byte(policyJSON), &policy); err != nil {
			return ports.PermissionMigrationSnapshot{}, err
		}
		if err = policy.Validate(); err != nil {
			return ports.PermissionMigrationSnapshot{}, err
		}
		out.Policy = &policy
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.PermissionMigrationSnapshot{}, mapSQLError("read permission migration policy", err)
	}
	return out, tx.Commit()
}

func (s *Store) ApplyPermissionMigration(ctx context.Context, r ports.PermissionMigrationRequest) (ports.PermissionMigrationResult, error) {
	var out ports.PermissionMigrationResult
	c := r.Command
	if err := validatePermissionMigrationApply(c); err != nil {
		return out, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return out, mapSQLError("begin permission migration", err)
	}
	defer tx.Rollback()
	var namespaceVersion int64
	if err = tx.QueryRowContext(ctx, `SELECT version FROM namespaces WHERE id=? AND lifecycle='active'`, c.NamespaceID.String()).Scan(&namespaceVersion); err != nil {
		return out, mapSQLError("guard permission migration namespace", err)
	}
	if err = authorizePermissionMigration(ctx, tx, c); err != nil {
		return out, err
	}
	var fingerprint, payload string
	err = tx.QueryRowContext(ctx, `SELECT fingerprint,result_json FROM permission_migration_results WHERE namespace_id=? AND principal_id=? AND idempotency_key=?`, c.NamespaceID.String(), c.CallerPrincipalID, c.IdempotencyKey).Scan(&fingerprint, &payload)
	if err == nil {
		if fingerprint != c.Fingerprint {
			return out, domain.NewError(domain.ErrorCodeIdempotencyConflict, "idempotency key represents another permission migration")
		}
		if err = json.Unmarshal([]byte(payload), &out); err != nil {
			return out, err
		}
		out.IdempotentReplay = true
		return out, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return out, mapSQLError("read permission migration receipt", err)
	}
	if domain.Version(namespaceVersion) != c.ExpectedNamespaceVersion {
		return out, domain.NewError(domain.ErrorCodeVersionConflict, "namespace version changed")
	}
	var grantJSON string
	var grantCompatibility uint32
	if err = tx.QueryRowContext(ctx, `SELECT permissions_json,permission_compatibility_version FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, c.NamespaceID.String(), c.PrincipalID).Scan(&grantJSON, &grantCompatibility); err != nil {
		return out, mapSQLError("read permission migration target grant", err)
	}
	var grantPermissions []ports.Permission
	if err = unmarshalJSON(grantJSON, &grantPermissions); err != nil {
		return out, err
	}
	var policy *domain.CredentialPolicy
	if !c.CredentialID.IsZero() {
		_, readErr := permissionMigrationCredential(ctx, tx, c.NamespaceID, c.PrincipalID, c.CredentialID)
		if readErr != nil {
			return out, readErr
		}
		var policyJSON string
		readErr = tx.QueryRowContext(ctx, `SELECT state_json FROM credential_policies WHERE namespace_id=? AND credential_id=?`, c.NamespaceID.String(), c.CredentialID.String()).Scan(&policyJSON)
		if readErr == nil {
			var value domain.CredentialPolicy
			if readErr = json.Unmarshal([]byte(policyJSON), &value); readErr != nil {
				return out, readErr
			}
			if readErr = value.Validate(); readErr != nil {
				return out, readErr
			}
			policy = &value
		} else if !errors.Is(readErr, sql.ErrNoRows) {
			return out, mapSQLError("read permission migration target policy", readErr)
		}
		actualPolicyVersion := domain.Version(0)
		if policy != nil {
			actualPolicyVersion = policy.Version
		}
		if !c.PolicyVersionPresent || actualPolicyVersion != c.ExpectedPolicyVersion {
			return out, domain.NewError(domain.ErrorCodeVersionConflict, "credential policy version changed")
		}
	} else if c.PolicyVersionPresent || c.ExpectedPolicyVersion != 0 {
		return out, domain.NewError(domain.ErrorCodeInvalidArgument, "policy expectation requires a credential target")
	}
	if grantCompatibility == ports.PermissionCompatibilityLegacyV1 && (policy == nil || policy.PermissionCompatibilityVersion == domain.PermissionCompatibilityLegacyV1) {
		out = ports.PermissionMigrationResult{NamespaceID: c.NamespaceID, PrincipalID: c.PrincipalID, CredentialID: c.CredentialID, NamespaceVersion: domain.Version(namespaceVersion), MapVersion: "legacy-v1", AlreadyMigrated: true}
		if policy != nil {
			out.PolicyVersion = policy.Version
		}
		if err = savePermissionMigrationReceipt(ctx, tx, c, out); err != nil {
			return ports.PermissionMigrationResult{}, err
		}
		return out, tx.Commit()
	}
	nextNamespace := domain.Version(namespaceVersion) + 1
	if nextNamespace == 0 {
		return out, domain.NewError(domain.ErrorCodeVersionConflict, "namespace version exhausted")
	}
	if grantCompatibility != ports.PermissionCompatibilityLegacyV1 {
		if _, err = tx.ExecContext(ctx, `UPDATE namespace_grants SET permission_compatibility_version=? WHERE namespace_id=? AND principal_id=?`, ports.PermissionCompatibilityLegacyV1, c.NamespaceID.String(), c.PrincipalID); err != nil {
			return out, mapSQLError("mark grant compatibility", err)
		}
	}
	policyMigrated := false
	policyVersion := domain.Version(0)
	if policy != nil {
		policyVersion = policy.Version
		if policy.PermissionCompatibilityVersion != domain.PermissionCompatibilityLegacyV1 {
			if policy.Version == ^domain.Version(0) {
				return out, domain.NewError(domain.ErrorCodeVersionConflict, "credential policy version exhausted")
			}
			previous := policy.Version
			policy.Version++
			policy.PermissionCompatibilityVersion = domain.PermissionCompatibilityLegacyV1
			encoded, marshalErr := json.Marshal(policy)
			if marshalErr != nil {
				return out, marshalErr
			}
			result, updateErr := tx.ExecContext(ctx, `UPDATE credential_policies SET version=?,state_json=? WHERE namespace_id=? AND credential_id=? AND principal_id=? AND version=?`, policy.Version, string(encoded), c.NamespaceID.String(), c.CredentialID.String(), c.PrincipalID, previous)
			if updateErr != nil {
				return out, mapSQLError("update permission migration policy", updateErr)
			}
			rows, rowsErr := result.RowsAffected()
			if rowsErr != nil || rows != 1 {
				return out, domain.NewError(domain.ErrorCodeVersionConflict, "credential policy changed")
			}
			policyMigrated = true
			policyVersion = policy.Version
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE namespaces SET version=?,updated_at=? WHERE id=? AND version=? AND lifecycle='active'`, nextNamespace, encodeTime(c.Now), c.NamespaceID.String(), namespaceVersion); err != nil {
		return out, mapSQLError("advance permission migration namespace", err)
	}
	var actorJSON string
	actorJSON, err = marshalJSON(c.CallerActor)
	if err != nil {
		return out, err
	}
	target := fmt.Sprintf("principal=%s credential=%s reason=%s", c.PrincipalID, c.CredentialID.String(), strings.TrimSpace(c.Reason))
	if _, err = tx.ExecContext(ctx, `INSERT INTO security_audit(namespace_id,namespace_version,principal_id,actor_json,operation,target,recorded_at) VALUES(?,?,?,?,?,?,?)`, c.NamespaceID.String(), nextNamespace, c.CallerPrincipalID, actorJSON, "permission_compatibility_migrate", target, encodeTime(c.Now)); err != nil {
		return out, mapSQLError("audit permission migration", err)
	}
	out = ports.PermissionMigrationResult{NamespaceID: c.NamespaceID, PrincipalID: c.PrincipalID, CredentialID: c.CredentialID, NamespaceVersion: nextNamespace, PolicyVersion: policyVersion, PolicyMigrated: policyMigrated, MapVersion: "legacy-v1"}
	if err = savePermissionMigrationReceipt(ctx, tx, c, out); err != nil {
		return ports.PermissionMigrationResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return ports.PermissionMigrationResult{}, mapSQLError("commit permission migration", err)
	}
	return out, nil
}

func savePermissionMigrationReceipt(ctx context.Context, tx *sql.Tx, c ports.PermissionMigrationApply, result ports.PermissionMigrationResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO permission_migration_results(namespace_id,principal_id,idempotency_key,fingerprint,result_json) VALUES(?,?,?,?,?)`, c.NamespaceID.String(), c.CallerPrincipalID, c.IdempotencyKey, c.Fingerprint, string(payload))
	if err != nil {
		return mapSQLError("save permission migration receipt", err)
	}
	return nil
}

func authorizePermissionMigration(ctx context.Context, tx *sql.Tx, c ports.PermissionMigrationApply) error {
	var grantJSON string
	var activePrincipal string
	err := tx.QueryRowContext(ctx, `SELECT g.permissions_json,p.lifecycle FROM namespace_grants g JOIN principals p ON p.id=g.principal_id WHERE g.namespace_id=? AND g.principal_id=?`, c.NamespaceID.String(), c.CallerPrincipalID).Scan(&grantJSON, &activePrincipal)
	var permissions []ports.Permission
	if err != nil || activePrincipal != "active" || unmarshalJSON(grantJSON, &permissions) != nil {
		return domain.NewError(domain.ErrorCodeForbidden, "permission migration authority denied")
	}
	allowed := false
	for _, permission := range permissions {
		allowed = allowed || permission == ports.PermissionPermissionsManage
	}
	if !allowed {
		return domain.NewError(domain.ErrorCodeForbidden, "permission migration requires permissions:manage")
	}
	if c.CallerCredentialDigest == "" {
		return nil
	}
	var principal, namespace, actorJSON, parent string
	policyCredentialDigest := c.CallerCredentialDigest
	var revoked int
	var expiry int64
	err = tx.QueryRowContext(ctx, `SELECT principal_id,namespace_id,actor_json,COALESCE(parent_digest,''),revoked,expires_at FROM credentials WHERE digest=?`, c.CallerCredentialDigest).Scan(&principal, &namespace, &actorJSON, &parent, &revoked, &expiry)
	var actor domain.ActorRef
	if err != nil || principal != c.CallerPrincipalID || namespace != c.NamespaceID.String() || revoked != 0 || expiry <= encodeTime(c.Now) || unmarshalJSON(actorJSON, &actor) != nil || actor != c.CallerActor {
		return domain.NewError(domain.ErrorCodeForbidden, "permission migration credential inactive")
	}
	if parent != "" {
		policyCredentialDigest = parent
		err = tx.QueryRowContext(ctx, `SELECT principal_id,namespace_id,actor_json,COALESCE(parent_digest,''),revoked,expires_at FROM credentials WHERE digest=?`, parent).Scan(&principal, &namespace, &actorJSON, &parent, &revoked, &expiry)
		if err != nil || parent != "" || principal != c.CallerPrincipalID || namespace != c.NamespaceID.String() || revoked != 0 || expiry <= encodeTime(c.Now) || unmarshalJSON(actorJSON, &actor) != nil || actor != c.CallerActor {
			return domain.NewError(domain.ErrorCodeForbidden, "permission migration session parent inactive")
		}
	}
	var policyJSON string
	err = tx.QueryRowContext(ctx, `SELECT state_json FROM credential_policies WHERE namespace_id=? AND credential_id=(SELECT id FROM credentials WHERE digest=?)`, c.NamespaceID.String(), policyCredentialDigest).Scan(&policyJSON)
	if err == nil {
		var policy domain.CredentialPolicy
		if json.Unmarshal([]byte(policyJSON), &policy) != nil || policy.Validate() != nil || !policy.Permits(string(ports.PermissionPermissionsManage), domain.ID("")) {
			return domain.NewError(domain.ErrorCodeForbidden, "permission migration credential policy denies authority")
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return mapSQLError("authorize permission migration policy", err)
	}
	return nil
}

func permissionMigrationCredential(ctx context.Context, tx *sql.Tx, ns domain.ID, principal string, id domain.ID) (ports.Credential, error) {
	var credential ports.Credential
	var rawID, rawNS, actor string
	var expiry int64
	err := tx.QueryRowContext(ctx, `SELECT id,namespace_id,principal_id,actor_json,expires_at,revoked,COALESCE(parent_digest,''),digest FROM credentials WHERE namespace_id=? AND id=?`, ns.String(), id.String()).Scan(&rawID, &rawNS, &credential.PrincipalID, &actor, &expiry, &credential.Revoked, &credential.ParentDigest, &credential.Digest)
	if err != nil {
		return credential, mapSQLError("read permission migration credential", err)
	}
	if credential.PrincipalID != principal {
		return credential, domain.NewError(domain.ErrorCodeInvalidScope, "credential does not belong to target principal")
	}
	credential.ID, err = domain.ParseID(rawID)
	if err != nil {
		return credential, err
	}
	credential.NamespaceID, err = domain.ParseID(rawNS)
	if err != nil {
		return credential, err
	}
	credential.ExpiresAt = decodeTime(expiry)
	if err = unmarshalJSON(actor, &credential.Actor); err != nil {
		return credential, err
	}
	return credential, nil
}

func validatePermissionMigrationRead(r ports.PermissionMigrationRead) error {
	if r.NamespaceID.Validate() != nil || strings.TrimSpace(r.PrincipalID) == "" || len(r.PrincipalID) > 256 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "permission migration target is invalid")
	}
	if !r.CredentialID.IsZero() {
		return r.CredentialID.Validate()
	}
	return nil
}

func validatePermissionMigrationApply(c ports.PermissionMigrationApply) error {
	if c.NamespaceID.Validate() != nil || strings.TrimSpace(c.PrincipalID) == "" || c.CallerPrincipalID == "" || c.CallerPrincipalID == " " || c.ExpectedNamespaceVersion == 0 || strings.TrimSpace(c.IdempotencyKey) == "" || strings.TrimSpace(c.Fingerprint) == "" || len(strings.TrimSpace(c.Reason)) < 3 || len(strings.TrimSpace(c.Reason)) > 1000 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "invalid permission migration request")
	}
	if !c.CredentialID.IsZero() {
		if err := c.CredentialID.Validate(); err != nil {
			return err
		}
		if !c.PolicyVersionPresent {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "expected policy version presence is required")
		}
	} else if c.PolicyVersionPresent || c.ExpectedPolicyVersion != 0 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "policy expectation requires a credential target")
	}
	if err := c.CallerActor.Validate(); err != nil {
		return err
	}
	return nil
}

var _ ports.PermissionCompatibilityStore = (*Store)(nil)
