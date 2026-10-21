package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"time"
	"unicode/utf8"
)

func (s *Store) ApplySecurityAdmin(ctx context.Context, r ports.SecurityAdminRequest) (ports.SecurityAdminResult, error) {
	var result ports.SecurityAdminResult
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return result, mapSQLError("begin security transaction", err)
	}
	defer tx.Rollback()
	var version int64
	err = tx.QueryRowContext(ctx, `SELECT version FROM namespaces WHERE id=? AND lifecycle='active'`, r.Command.NamespaceID.String()).Scan(&version)
	if err != nil {
		return result, mapSQLError("guard namespace", err)
	}
	// Authorization and parent-session revocation are rechecked in the same snapshot as the write/replay.
	var grantJSON string
	err = tx.QueryRowContext(ctx, `SELECT g.permissions_json FROM namespace_grants g JOIN principals p ON p.id=g.principal_id WHERE g.namespace_id=? AND g.principal_id=? AND p.lifecycle='active'`, r.Command.NamespaceID.String(), r.PrincipalID).Scan(&grantJSON)
	var permissions []ports.Permission
	if err != nil || unmarshalJSON(grantJSON, &permissions) != nil {
		return result, domain.NewError(domain.ErrorCodeForbidden, "namespace administration denied")
	}
	required, registered := ports.SecurityAdminPermission(r.Command.Operation)
	if !registered {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "unknown security operation")
	}
	authorized, delegate := false, false
	for _, p := range permissions {
		authorized = authorized || p == required
		delegate = delegate || p == ports.PermissionActorDelegate
	}
	if !authorized {
		return result, domain.NewError(domain.ErrorCodeForbidden, "security operation permission denied")
	}
	if r.Command.Credential != nil && r.Command.Credential.Actor != r.Actor && !delegate {
		return result, domain.NewError(domain.ErrorCodeForbidden, "actor delegation denied")
	}
	rootCredentialID, err := authorizeSecurityAdminReadTx(ctx, tx, ports.SecurityAdminReadAccess{NamespaceID: r.Command.NamespaceID, PrincipalID: r.PrincipalID, CredentialID: r.CallerCredentialID, CredentialDigest: r.CredentialDigest, Actor: r.Actor, Permissions: []ports.Permission{required}, Now: r.Now})
	if err != nil {
		return result, err
	}
	var fingerprint, payload string
	err = tx.QueryRowContext(ctx, `SELECT fingerprint,result_json FROM security_admin_results WHERE namespace_id=? AND principal_id=? AND idempotency_key=?`, r.Command.NamespaceID.String(), r.PrincipalID, r.IdempotencyKey).Scan(&fingerprint, &payload)
	if err == nil {
		if fingerprint != r.Fingerprint {
			return result, domain.NewError(domain.ErrorCodeIdempotencyConflict, "key already represents a different intent")
		}
		if err = unmarshalJSON(payload, &result); err != nil {
			return result, err
		}
		result.IdempotentReplay = true
		result.TokenOmitted = result.Credential != nil
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, mapSQLError("read security receipt", err)
	}
	if domain.Version(version) != r.Command.ExpectedVersion {
		return result, domain.NewError(domain.ErrorCodeVersionConflict, "namespace version changed")
	}
	beforeState, err := securityAdminBeforeTx(ctx, tx, r.Command, r.Now)
	if err != nil {
		return result, err
	}
	target := ""
	switch r.Command.Operation {
	case "create_namespace":
		n, c := r.Command.NewNamespace, r.Command.Credential
		if n == nil || c == nil {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "namespace and initial credential required")
		}
		target = n.ID.String()
		granted, err := marshalJSON(permissions)
		if err != nil {
			return result, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO namespaces(id,name,lifecycle,version,created_at,updated_at) VALUES(?,?,'active',1,?,?)`, n.ID.String(), n.Name, encodeTime(r.Now), encodeTime(r.Now)); err != nil {
			return result, mapSQLError("create namespace", err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO namespace_grants(namespace_id,principal_id,permissions_json) VALUES(?,?,?)`, n.ID.String(), r.PrincipalID, granted); err != nil {
			return result, err
		}
		initialGrant := ports.NamespaceGrant{NamespaceID: n.ID, PrincipalID: c.PrincipalID, Permissions: permissions}
		if err = persistCredentialBundle(ctx, tx, c, r.Command.CredentialPolicy, r.Command.CredentialMetadata, initialGrant, r.Now, r.PrincipalID, r.Actor); err != nil {
			return result, err
		}
		result.Credential = c
		result.CredentialMetadata = r.Command.CredentialMetadata
		result.CreatedNamespace = n
	case "set_grant":
		g := r.Command.Grant
		if g == nil || g.NamespaceID != r.Command.NamespaceID || strings.TrimSpace(g.PrincipalID) == "" {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "grant required")
		}
		if err = ports.ValidatePermissionValues(g.Permissions); err != nil {
			return result, err
		}
		target = g.PrincipalID
		if err = ensurePrincipal(ctx, tx, g.PrincipalID); err != nil {
			return result, err
		}
		p, err := marshalJSON(g.Permissions)
		if err != nil {
			return result, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO namespace_grants(namespace_id,principal_id,permissions_json,permission_compatibility_version) VALUES(?,?,?,0) ON CONFLICT(namespace_id,principal_id) DO UPDATE SET permissions_json=excluded.permissions_json,permission_compatibility_version=0`, g.NamespaceID.String(), g.PrincipalID, p)
		if err != nil {
			return result, mapSQLError("write grant", err)
		}
	case "issue_credential":
		c := r.Command.Credential
		if c == nil || c.NamespaceID != r.Command.NamespaceID || c.ID.Validate() != nil || c.Actor.Validate() != nil || c.Digest == "" || c.Revoked || c.ParentDigest != "" || !c.ExpiresAt.After(r.Now) || c.ExpiresAt.After(r.Now.Add(366*24*time.Hour)) {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "valid root credential required")
		}
		target = c.ID.String()
		var exists int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM namespace_grants g JOIN principals p ON p.id=g.principal_id WHERE g.namespace_id=? AND g.principal_id=? AND p.lifecycle='active'`, c.NamespaceID.String(), c.PrincipalID).Scan(&exists)
		if err != nil || exists != 1 {
			return result, domain.NewError(domain.ErrorCodeForbidden, "target principal has no namespace grant")
		}
		var rawGrant string
		if err = tx.QueryRowContext(ctx, `SELECT permissions_json FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, c.NamespaceID.String(), c.PrincipalID).Scan(&rawGrant); err != nil {
			return result, mapSQLError("read target grant", err)
		}
		var targetPermissions []ports.Permission
		if err = unmarshalJSON(rawGrant, &targetPermissions); err != nil {
			return result, err
		}
		grant := ports.NamespaceGrant{NamespaceID: c.NamespaceID, PrincipalID: c.PrincipalID, Permissions: targetPermissions}
		if err = persistCredentialBundle(ctx, tx, c, r.Command.CredentialPolicy, r.Command.CredentialMetadata, grant, r.Now, r.PrincipalID, r.Actor); err != nil {
			return result, err
		}
		result.Credential = c
		result.CredentialMetadata = r.Command.CredentialMetadata
	case "update_credential":
		metadata, credential, err := updateCredentialTx(ctx, tx, r)
		if err != nil {
			return result, err
		}
		target = credential.ID.String()
		result.Credential = &credential
		result.CredentialMetadata = &metadata
	case "rotate_credential":
		metadata, credential, targetID, err := rotateCredentialTx(ctx, tx, r)
		if err != nil {
			return result, err
		}
		target = r.Command.CredentialID.String() + "->" + targetID.String()
		result.Credential = &credential
		result.CredentialMetadata = &metadata
	case "revoke_credential":
		target = r.Command.CredentialID.String()
		impact, impactErr := credentialRevocationImpactTx(ctx, tx, r.Command.NamespaceID, r.Command.CredentialID, r.Now)
		if impactErr != nil {
			return result, impactErr
		}
		result.RevocationImpact = &impact
		updated, err := tx.ExecContext(ctx, `UPDATE credentials SET revoked=1 WHERE namespace_id=? AND id=?`, r.Command.NamespaceID.String(), target)
		if err != nil {
			return result, err
		}
		n, _ := updated.RowsAffected()
		if n != 1 {
			return result, domain.NewError(domain.ErrorCodeNotFound, "credential not found")
		}
	default:
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "unknown security operation")
	}
	adminNamespace := r.Command.NamespaceID
	if r.Command.Operation == "create_namespace" {
		if result.CreatedNamespace == nil {
			return result, domain.NewError(domain.ErrorCodeInvalidTransition, "created Namespace missing from result")
		}
		adminNamespace = result.CreatedNamespace.ID
	}
	if err = ensureRecoverableAdminTx(ctx, tx, adminNamespace, r.Now); err != nil {
		return result, err
	}
	result.NamespaceVersion = domain.Version(version + 1)
	updated, err := tx.ExecContext(ctx, `UPDATE namespaces SET version=version+1,updated_at=? WHERE id=? AND version=?`, encodeTime(r.Now), r.Command.NamespaceID.String(), version)
	if err != nil {
		return result, mapSQLError("advance namespace version", err)
	}
	n, _ := updated.RowsAffected()
	if n != 1 {
		return result, domain.NewError(domain.ErrorCodeVersionConflict, "namespace version changed")
	}
	actor, err := marshalJSON(r.Actor)
	if err != nil {
		return result, err
	}
	afterState, err := securityAdminAfterTx(ctx, tx, r.Command, result, r.Now)
	if err != nil {
		return result, err
	}
	beforeJSON, err := marshalJSON(beforeState)
	if err != nil {
		return result, err
	}
	afterJSON, err := marshalJSON(afterState)
	if err != nil {
		return result, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO security_audit(namespace_id,namespace_version,principal_id,actor_json,author_credential_id,root_credential_id,operation,target,reason,before_json,after_json,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, r.Command.NamespaceID.String(), int64(result.NamespaceVersion), r.PrincipalID, actor, r.CallerCredentialID.String(), rootCredentialID.String(), r.Command.Operation, target, r.Command.Reason, string(beforeJSON), string(afterJSON), encodeTime(r.Now))
	if err != nil {
		return result, mapSQLError("append security audit", err)
	}
	payload, err = marshalJSON(result)
	if err != nil {
		return result, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO security_admin_results(namespace_id,principal_id,idempotency_key,fingerprint,result_json) VALUES(?,?,?,?,?)`, r.Command.NamespaceID.String(), r.PrincipalID, r.IdempotencyKey, r.Fingerprint, payload)
	if err != nil {
		return result, mapSQLError("persist security receipt", err)
	}
	return result, mapSQLError("commit security administration", tx.Commit())
}

func securityAdminBeforeTx(ctx context.Context, tx *sql.Tx, command ports.SecurityAdminCommand, now time.Time) (map[string]any, error) {
	before := map[string]any{"exists": false}
	switch command.Operation {
	case "set_grant":
		if command.Grant == nil {
			return before, nil
		}
		var permissionsJSON string
		var compatibility uint32
		err := tx.QueryRowContext(ctx, `SELECT permissions_json,permission_compatibility_version FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, command.NamespaceID.String(), command.Grant.PrincipalID).Scan(&permissionsJSON, &compatibility)
		if errors.Is(err, sql.ErrNoRows) {
			return ports.SecurityAuditGrantState(false, nil, 0), nil
		}
		if err != nil {
			return nil, mapSQLError("read prior namespace grant", err)
		}
		var permissions []ports.Permission
		if unmarshalJSON(permissionsJSON, &permissions) != nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidTransition, "stored namespace grant is corrupt")
		}
		return ports.SecurityAuditGrantState(true, permissions, compatibility), nil
	case "issue_credential", "create_namespace":
		return before, nil
	case "update_credential", "rotate_credential", "revoke_credential":
		record, err := readManagedCredential(ctx, tx, command.NamespaceID, command.CredentialID, now)
		if err != nil {
			return nil, err
		}
		return ports.SecurityAuditCredentialState(record), nil
	default:
		return before, nil
	}
}

func securityAdminAfterTx(ctx context.Context, tx *sql.Tx, command ports.SecurityAdminCommand, result ports.SecurityAdminResult, now time.Time) (map[string]any, error) {
	switch command.Operation {
	case "set_grant":
		if command.Grant == nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "grant required for audit")
		}
		var raw string
		var compatibility uint32
		if err := tx.QueryRowContext(ctx, `SELECT permissions_json,permission_compatibility_version FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, command.NamespaceID.String(), command.Grant.PrincipalID).Scan(&raw, &compatibility); err != nil {
			return nil, err
		}
		var permissions []ports.Permission
		if unmarshalJSON(raw, &permissions) != nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidTransition, "stored namespace grant is corrupt")
		}
		return ports.SecurityAuditGrantState(true, permissions, compatibility), nil
	case "issue_credential", "update_credential", "rotate_credential", "revoke_credential":
		credentialID := command.CredentialID
		if command.Operation == "issue_credential" || command.Operation == "rotate_credential" {
			if result.Credential == nil {
				return nil, domain.NewError(domain.ErrorCodeInvalidTransition, "credential result missing for audit")
			}
			credentialID = result.Credential.ID
		}
		record, err := readManagedCredential(ctx, tx, command.NamespaceID, credentialID, now)
		if err != nil {
			return nil, err
		}
		state := ports.SecurityAuditCredentialState(record)
		if result.RevocationImpact != nil {
			state["revocation_impact"] = result.RevocationImpact
		}
		return state, nil
	case "create_namespace":
		if result.CreatedNamespace == nil || result.Credential == nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidTransition, "namespace result missing for audit")
		}
		record, err := readManagedCredential(ctx, tx, result.CreatedNamespace.ID, result.Credential.ID, now)
		if err != nil {
			return nil, err
		}
		var raw string
		var compatibility uint32
		if err = tx.QueryRowContext(ctx, `SELECT permissions_json,permission_compatibility_version FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, result.CreatedNamespace.ID.String(), result.Credential.PrincipalID).Scan(&raw, &compatibility); err != nil {
			return nil, err
		}
		var permissions []ports.Permission
		if unmarshalJSON(raw, &permissions) != nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidTransition, "stored initial Namespace grant is corrupt")
		}
		return map[string]any{"exists": true, "namespace_id": result.CreatedNamespace.ID, "name": result.CreatedNamespace.Name, "initial_grant": ports.SecurityAuditGrantState(true, permissions, compatibility), "initial_credential": ports.SecurityAuditCredentialState(record)}, nil
	default:
		return map[string]any{"exists": true}, nil
	}
}

func ensureRecoverableAdminTx(ctx context.Context, tx *sql.Tx, ns domain.ID, now time.Time) error {
	rows, err := tx.QueryContext(ctx, `SELECT c.id,c.expires_at,g.permissions_json,p.state_json FROM credentials c JOIN namespace_grants g ON g.namespace_id=c.namespace_id AND g.principal_id=c.principal_id JOIN principals pr ON pr.id=c.principal_id AND pr.lifecycle='active' LEFT JOIN credential_policies p ON p.namespace_id=c.namespace_id AND p.credential_id=c.id WHERE c.namespace_id=? AND c.parent_digest IS NULL AND c.revoked=0 AND c.expires_at>?`, ns.String(), encodeTime(now))
	if err != nil {
		return mapSQLError("check recoverable administrator", err)
	}
	defer rows.Close()
	adminPermissions := []ports.Permission{ports.PermissionPermissionsManage, ports.PermissionCredentialsIssue, ports.PermissionCredentialsUpdate, ports.PermissionCredentialsRotate}
	for rows.Next() {
		var id string
		var expires int64
		var grantJSON string
		var policyJSON sql.NullString
		if err = rows.Scan(&id, &expires, &grantJSON, &policyJSON); err != nil {
			return err
		}
		var granted []ports.Permission
		if unmarshalJSON(grantJSON, &granted) != nil {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "stored namespace grant is corrupt")
		}
		var policy domain.CredentialPolicy
		if policyJSON.Valid && unmarshalJSON(policyJSON.String, &policy) != nil {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "stored credential policy is corrupt")
		}
		for _, candidate := range adminPermissions {
			grantHas := false
			for _, grantedPermission := range granted {
				grantHas = grantHas || candidate == grantedPermission
			}
			policyHas := !policyJSON.Valid || policy.PermitsWithCompatibility(string(candidate), "", "")
			if grantHas && policyHas {
				return nil
			}
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return domain.NewError(domain.ErrorCodeInvalidTransition, "operation would remove the last recoverable administrator credential")
}

func persistCredentialBundle(ctx context.Context, tx *sql.Tx, credential *ports.Credential, policy *domain.CredentialPolicy, metadata *ports.CredentialMetadata, grant ports.NamespaceGrant, now time.Time, issuerPrincipal string, issuerActor domain.ActorRef) error {
	if credential == nil || policy == nil || metadata == nil {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "credential, explicit policy, and metadata are required")
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	if credential.NamespaceID != grant.NamespaceID || credential.PrincipalID != grant.PrincipalID || policy.NamespaceID != credential.NamespaceID || policy.CredentialID != credential.ID || policy.PrincipalID != credential.PrincipalID || policy.PermissionCompatibilityVersion != 0 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "credential policy scope does not match its grant")
	}
	if metadata.NamespaceID != credential.NamespaceID || metadata.CredentialID != credential.ID || metadata.PolicyRevision != policy.Version || metadata.MetadataRevision != 1 || metadata.CreatedByPrincipalID != issuerPrincipal || metadata.CreatedByActor != issuerActor || metadata.CreatedAt.IsZero() || !metadata.CreatedAt.Equal(now) || !metadata.UpdatedAt.Equal(now) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "credential metadata does not match issuance")
	}
	if utf8.RuneCountInString(metadata.Name) < 1 || utf8.RuneCountInString(metadata.Name) > 100 || utf8.RuneCountInString(metadata.Description) > 1000 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "credential metadata is outside the supported length")
	}
	if credential.ID.Validate() != nil || credential.Actor.Validate() != nil || credential.Digest == "" || credential.Revoked || credential.ParentDigest != "" || !credential.ExpiresAt.After(now) || credential.ExpiresAt.After(now.Add(366*24*time.Hour)) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "credential must be an active root credential with a bounded expiration")
	}
	if metadata.CreatedByPrincipalID == "" || metadata.CreatedByActor.Validate() != nil {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "credential issuer metadata is invalid")
	}
	granted := make(map[ports.Permission]struct{}, len(grant.Permissions))
	for _, permission := range grant.Permissions {
		granted[permission] = struct{}{}
	}
	for _, operation := range policy.PermittedOperations {
		if _, ok := granted[ports.Permission(operation)]; !ok {
			return domain.NewError(domain.ErrorCodeForbidden, "credential policy exceeds the target Principal grant")
		}
	}
	for _, outcomeID := range policy.AllowedOutcomeIDs {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM outcomes WHERE namespace_id=? AND id=?`, credential.NamespaceID.String(), outcomeID.String()).Scan(&count); err != nil || count != 1 {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "credential Outcome scope must reference Outcomes in the same Namespace")
		}
	}
	for _, keyID := range policy.AllowedSigningKeyIDs {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM signing_keys WHERE namespace_id=? AND id=? AND principal_id=? AND status='active'`, credential.NamespaceID.String(), keyID.String(), credential.PrincipalID).Scan(&count); err != nil || count != 1 {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "credential signing-key scope must reference active keys of the target Principal")
		}
	}
	actorJSON, err := marshalJSON(credential.Actor)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO credentials(id,namespace_id,principal_id,actor_json,digest,expires_at,revoked,parent_digest) VALUES(?,?,?,?,?,?,0,NULL)`, credential.ID.String(), credential.NamespaceID.String(), credential.PrincipalID, actorJSON, credential.Digest, encodeTime(credential.ExpiresAt)); err != nil {
		return mapSQLError("issue credential", err)
	}
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO credential_policies(namespace_id,credential_id,principal_id,version,state_json) VALUES(?,?,?,?,?)`, policy.NamespaceID.String(), policy.CredentialID.String(), policy.PrincipalID, int64(policy.Version), string(policyJSON)); err != nil {
		return mapSQLError("persist credential policy", err)
	}
	createdActorJSON, err := marshalJSON(metadata.CreatedByActor)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO credential_metadata(namespace_id,credential_id,name,description,created_at,created_by,created_by_actor_json,role_template_id,role_template_version,policy_revision,updated_at,metadata_revision,replaced_by_credential_id,replaces_credential_id,rotation_overlap_until) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, metadata.NamespaceID.String(), metadata.CredentialID.String(), metadata.Name, metadata.Description, encodeTime(metadata.CreatedAt), metadata.CreatedByPrincipalID, createdActorJSON, nullableString(string(metadata.RoleTemplateID)), metadata.RoleTemplateVersion, int64(metadata.PolicyRevision), encodeTime(metadata.UpdatedAt), int64(metadata.MetadataRevision), nullableCredentialID(metadata.ReplacedByCredentialID), nullableCredentialID(metadata.ReplacesCredentialID), nullableTime(metadata.RotationOverlapUntil)); err != nil {
		return mapSQLError("persist credential metadata", err)
	}
	return nil
}

func (s *Store) ReadSecurityAdmin(ctx context.Context, ns domain.ID) (ports.SecurityAdminSnapshot, error) {
	return s.readSecurityAdmin(ctx, ns, nil)
}

func (s *Store) ReadSecurityAdminAuthorized(ctx context.Context, access ports.SecurityAdminReadAccess) (ports.SecurityAdminSnapshot, error) {
	return s.readSecurityAdmin(ctx, access.NamespaceID, &access)
}

func (s *Store) readSecurityAdmin(ctx context.Context, ns domain.ID, access *ports.SecurityAdminReadAccess) (ports.SecurityAdminSnapshot, error) {
	var result ports.SecurityAdminSnapshot
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelSerializable})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if access != nil {
		if _, err = authorizeSecurityAdminReadTx(ctx, tx, *access); err != nil {
			return result, err
		}
	}
	result, err = readSecurityAdminTx(ctx, tx, ns)
	if err != nil {
		return ports.SecurityAdminSnapshot{}, err
	}
	return result, tx.Commit()
}

func readSecurityAdminTx(ctx context.Context, tx *sql.Tx, ns domain.ID) (ports.SecurityAdminSnapshot, error) {
	var result ports.SecurityAdminSnapshot
	var err error
	result.Namespace.ID = ns
	err = tx.QueryRowContext(ctx, `SELECT name,version FROM namespaces WHERE id=? AND lifecycle='active'`, ns.String()).Scan(&result.Namespace.Name, &result.NamespaceVersion)
	if err != nil {
		return result, mapSQLError("read namespace", err)
	}
	result.Grants = []ports.NamespaceGrant{}
	result.Credentials = []ports.Credential{}
	result.Audit = []ports.SecurityAudit{}
	rows, err := tx.QueryContext(ctx, `SELECT principal_id,permissions_json,permission_compatibility_version FROM namespace_grants WHERE namespace_id=? ORDER BY principal_id LIMIT 101`, ns.String())
	if err != nil {
		return result, err
	}
	for rows.Next() {
		g := ports.NamespaceGrant{NamespaceID: ns}
		var p string
		if err = rows.Scan(&g.PrincipalID, &p, &g.PermissionCompatibilityVersion); err != nil {
			rows.Close()
			return result, err
		}
		if err = unmarshalJSON(p, &g.Permissions); err != nil {
			rows.Close()
			return result, err
		}
		result.Grants = append(result.Grants, g)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT c.id,c.principal_id,c.actor_json,c.expires_at,c.revoked,m.name,m.description,m.created_at,m.created_by,m.created_by_actor_json,m.role_template_id,m.role_template_version,m.policy_revision FROM credentials c LEFT JOIN credential_metadata m ON m.namespace_id=c.namespace_id AND m.credential_id=c.id WHERE c.namespace_id=? AND c.parent_digest IS NULL ORDER BY c.id LIMIT 101`, ns.String())
	if err != nil {
		return result, err
	}
	for rows.Next() {
		c := ports.Credential{NamespaceID: ns}
		var id, a string
		var expires int64
		var name, description, createdBy, createdActor, templateID sql.NullString
		var createdAt, templateVersion, policyRevision sql.NullInt64
		if err = rows.Scan(&id, &c.PrincipalID, &a, &expires, &c.Revoked, &name, &description, &createdAt, &createdBy, &createdActor, &templateID, &templateVersion, &policyRevision); err != nil {
			rows.Close()
			return result, err
		}
		c.ID, err = domain.ParseID(id)
		if err != nil {
			rows.Close()
			return result, err
		}
		if err = unmarshalJSON(a, &c.Actor); err != nil {
			rows.Close()
			return result, err
		}
		c.ExpiresAt = decodeTime(expires)
		result.Credentials = append(result.Credentials, c)
		if name.Valid {
			metadata := ports.CredentialMetadata{NamespaceID: ns, CredentialID: c.ID, Name: name.String, Description: description.String, CreatedAt: decodeTime(createdAt.Int64), CreatedByPrincipalID: createdBy.String, RoleTemplateID: ports.PermissionTemplateID(templateID.String), RoleTemplateVersion: uint32(templateVersion.Int64), PolicyRevision: domain.Version(policyRevision.Int64)}
			if err = unmarshalJSON(createdActor.String, &metadata.CreatedByActor); err != nil {
				rows.Close()
				return result, err
			}
			result.CredentialMetadata = append(result.CredentialMetadata, metadata)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT namespace_version,principal_id,actor_json,author_credential_id,root_credential_id,operation,target,reason,before_json,after_json,recorded_at FROM security_audit WHERE namespace_id=? ORDER BY namespace_version DESC LIMIT 101`, ns.String())
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var a ports.SecurityAudit
		var actor, reason, beforeJSON, afterJSON string
		var authorID, rootID sql.NullString
		var at int64
		if err = rows.Scan(&a.NamespaceVersion, &a.PrincipalID, &actor, &authorID, &rootID, &a.Operation, &a.Target, &reason, &beforeJSON, &afterJSON, &at); err != nil {
			rows.Close()
			return result, err
		}
		if authorID.Valid {
			a.AuthorCredentialID, err = domain.ParseID(authorID.String)
			if err != nil {
				rows.Close()
				return result, err
			}
		}
		if rootID.Valid {
			a.RootCredentialID, err = domain.ParseID(rootID.String)
			if err != nil {
				rows.Close()
				return result, err
			}
		}
		if err = unmarshalJSON(actor, &a.Actor); err != nil {
			rows.Close()
			return result, err
		}
		a.Reason = reason
		a.Before, a.After = json.RawMessage(append([]byte(nil), beforeJSON...)), json.RawMessage(append([]byte(nil), afterJSON...))
		a.RecordedAt = decodeTime(at)
		result.Audit = append(result.Audit, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	result.Truncated = len(result.Grants) > 100 || len(result.Credentials) > 100 || len(result.CredentialMetadata) > 100 || len(result.Audit) > 100
	result.Grants = result.Grants[:min(len(result.Grants), 100)]
	result.Credentials = result.Credentials[:min(len(result.Credentials), 100)]
	result.CredentialMetadata = result.CredentialMetadata[:min(len(result.CredentialMetadata), 100)]
	result.Audit = result.Audit[:min(len(result.Audit), 100)]
	return result, nil
}

func (s *Store) ReadSecurityAdminVersion(ctx context.Context, ns domain.ID) (domain.Version, error) {
	var version domain.Version
	err := s.db.QueryRowContext(ctx, `SELECT version FROM namespaces WHERE id=? AND lifecycle='active'`, ns.String()).Scan(&version)
	if err != nil {
		return 0, mapSQLError("read namespace version", err)
	}
	return version, nil
}

// ReadSecurityAdminReceipt authorizes a credential issuance or rotation replay
// before the application prepares a new secret from mutable target state.
func (s *Store) ReadSecurityAdminReceipt(ctx context.Context, r ports.SecurityAdminRequest) (ports.SecurityAdminResult, bool, error) {
	var result ports.SecurityAdminResult
	if err := domain.ValidateIdempotencyKey(r.IdempotencyKey); err != nil {
		return result, false, err
	}
	if (r.Command.Operation != "issue_credential" && r.Command.Operation != "rotate_credential") || r.Command.NamespaceID.Validate() != nil || r.Command.ExpectedVersion == 0 || strings.TrimSpace(r.PrincipalID) == "" || r.Fingerprint == "" || r.Now.IsZero() {
		return result, false, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid credential receipt request")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true})
	if err != nil {
		return result, false, mapSQLError("begin security receipt read", err)
	}
	defer tx.Rollback()
	var activeNamespace string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM namespaces WHERE id=? AND lifecycle='active'`, r.Command.NamespaceID.String()).Scan(&activeNamespace); err != nil {
		return result, false, mapSQLError("guard namespace receipt", err)
	}
	var grantJSON string
	if err = tx.QueryRowContext(ctx, `SELECT g.permissions_json FROM namespace_grants g JOIN principals p ON p.id=g.principal_id WHERE g.namespace_id=? AND g.principal_id=? AND p.lifecycle='active'`, r.Command.NamespaceID.String(), r.PrincipalID).Scan(&grantJSON); err != nil {
		return result, false, domain.NewError(domain.ErrorCodeForbidden, "namespace administration denied")
	}
	var permissions []ports.Permission
	if err = unmarshalJSON(grantJSON, &permissions); err != nil {
		return result, false, domain.NewError(domain.ErrorCodeForbidden, "namespace administration denied")
	}
	required, registered := ports.SecurityAdminPermission(r.Command.Operation)
	if !registered {
		return result, false, domain.NewError(domain.ErrorCodeInvalidArgument, "unknown security operation")
	}
	authorized, delegate := false, false
	for _, permission := range permissions {
		authorized = authorized || permission == required
		delegate = delegate || permission == ports.PermissionActorDelegate
	}
	if !authorized {
		return result, false, domain.NewError(domain.ErrorCodeForbidden, "security operation permission denied")
	}
	if r.Command.Credential != nil && r.Command.Credential.Actor != r.Actor && !delegate {
		return result, false, domain.NewError(domain.ErrorCodeForbidden, "actor delegation denied")
	}
	if _, err = authorizeSecurityAdminReadTx(ctx, tx, ports.SecurityAdminReadAccess{NamespaceID: r.Command.NamespaceID, PrincipalID: r.PrincipalID, CredentialID: r.CallerCredentialID, CredentialDigest: r.CredentialDigest, Actor: r.Actor, Permissions: []ports.Permission{required}, Now: r.Now}); err != nil {
		return result, false, err
	}
	var fingerprint, payload string
	err = tx.QueryRowContext(ctx, `SELECT fingerprint,result_json FROM security_admin_results WHERE namespace_id=? AND principal_id=? AND idempotency_key=?`, r.Command.NamespaceID.String(), r.PrincipalID, r.IdempotencyKey).Scan(&fingerprint, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return result, false, tx.Commit()
	}
	if err != nil {
		return result, false, mapSQLError("read security receipt", err)
	}
	if fingerprint != r.Fingerprint {
		return result, false, domain.NewError(domain.ErrorCodeIdempotencyConflict, "key already represents a different intent")
	}
	if err = unmarshalJSON(payload, &result); err != nil {
		return result, false, err
	}
	result.IdempotentReplay = true
	result.TokenOmitted = result.Credential != nil
	return result, true, tx.Commit()
}

var _ ports.SecurityAdministrationStore = (*Store)(nil)
