package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
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
	admin, delegate := false, false
	for _, p := range permissions {
		admin = admin || p == ports.PermissionNamespaceAdmin
		delegate = delegate || p == ports.PermissionActorDelegate
	}
	if !admin {
		return result, domain.NewError(domain.ErrorCodeForbidden, "namespace administration denied")
	}
	if r.Command.Credential != nil && r.Command.Credential.Actor != r.Actor && !delegate {
		return result, domain.NewError(domain.ErrorCodeForbidden, "actor delegation denied")
	}
	if r.CredentialDigest != "" {
		var principal, ns, actor, parent string
		var expires int64
		var revoked int
		err = tx.QueryRowContext(ctx, `SELECT principal_id,namespace_id,actor_json,COALESCE(parent_digest,''),expires_at,revoked FROM credentials WHERE digest=?`, r.CredentialDigest).Scan(&principal, &ns, &actor, &parent, &expires, &revoked)
		var binding domain.ActorRef
		if err != nil || principal != r.PrincipalID || ns != r.Command.NamespaceID.String() || revoked != 0 || expires <= encodeTime(r.Now) || unmarshalJSON(actor, &binding) != nil || binding != r.Actor {
			return result, domain.NewError(domain.ErrorCodeForbidden, "credential inactive")
		}
		if parent != "" {
			var active int
			err = tx.QueryRowContext(ctx, `SELECT count(*) FROM credentials WHERE digest=? AND principal_id=? AND namespace_id=? AND revoked=0 AND expires_at>? AND parent_digest IS NULL`, parent, principal, ns, encodeTime(r.Now)).Scan(&active)
			if err != nil || active != 1 {
				return result, domain.NewError(domain.ErrorCodeForbidden, "session parent inactive")
			}
		}
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
	target := ""
	switch r.Command.Operation {
	case "create_namespace":
		n, c := r.Command.NewNamespace, r.Command.Credential
		if n == nil || c == nil {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "namespace and initial credential required")
		}
		target = n.ID.String()
		actor, err := marshalJSON(c.Actor)
		if err != nil {
			return result, err
		}
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
		if _, err = tx.ExecContext(ctx, `INSERT INTO credentials(id,namespace_id,principal_id,actor_json,digest,expires_at,revoked) VALUES(?,?,?,?,?,?,0)`, c.ID.String(), n.ID.String(), c.PrincipalID, actor, c.Digest, encodeTime(c.ExpiresAt)); err != nil {
			return result, mapSQLError("create namespace credential", err)
		}
		result.Credential = c
		result.CreatedNamespace = n
	case "set_grant":
		g := r.Command.Grant
		if g == nil {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "grant required")
		}
		target = g.PrincipalID
		if err = ensurePrincipal(ctx, tx, g.PrincipalID); err != nil {
			return result, err
		}
		p, err := marshalJSON(g.Permissions)
		if err != nil {
			return result, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO namespace_grants(namespace_id,principal_id,permissions_json) VALUES(?,?,?) ON CONFLICT(namespace_id,principal_id) DO UPDATE SET permissions_json=excluded.permissions_json`, g.NamespaceID.String(), g.PrincipalID, p)
		if err != nil {
			return result, mapSQLError("write grant", err)
		}
	case "issue_credential":
		c := r.Command.Credential
		if c == nil {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential required")
		}
		target = c.ID.String()
		var exists int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, c.NamespaceID.String(), c.PrincipalID).Scan(&exists)
		if err != nil || exists != 1 {
			return result, domain.NewError(domain.ErrorCodeForbidden, "target principal has no namespace grant")
		}
		a, err := marshalJSON(c.Actor)
		if err != nil {
			return result, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO credentials(id,namespace_id,principal_id,actor_json,digest,expires_at,revoked) VALUES(?,?,?,?,?,?,0)`, c.ID.String(), c.NamespaceID.String(), c.PrincipalID, a, c.Digest, encodeTime(c.ExpiresAt))
		if err != nil {
			return result, mapSQLError("issue credential", err)
		}
		result.Credential = c
	case "revoke_credential":
		target = r.Command.CredentialID.String()
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
	_, err = tx.ExecContext(ctx, `INSERT INTO security_audit(namespace_id,namespace_version,principal_id,actor_json,operation,target,recorded_at) VALUES(?,?,?,?,?,?,?)`, r.Command.NamespaceID.String(), int64(result.NamespaceVersion), r.PrincipalID, actor, r.Command.Operation, target, encodeTime(r.Now))
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

func (s *Store) ReadSecurityAdmin(ctx context.Context, ns domain.ID) (ports.SecurityAdminSnapshot, error) {
	var result ports.SecurityAdminSnapshot
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelSerializable})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	result.Namespace.ID = ns
	err = tx.QueryRowContext(ctx, `SELECT name,version FROM namespaces WHERE id=? AND lifecycle='active'`, ns.String()).Scan(&result.Namespace.Name, &result.NamespaceVersion)
	if err != nil {
		return result, mapSQLError("read namespace", err)
	}
	result.Grants = []ports.NamespaceGrant{}
	result.Credentials = []ports.Credential{}
	result.Audit = []ports.SecurityAudit{}
	rows, err := tx.QueryContext(ctx, `SELECT principal_id,permissions_json FROM namespace_grants WHERE namespace_id=? ORDER BY principal_id LIMIT 101`, ns.String())
	if err != nil {
		return result, err
	}
	for rows.Next() {
		g := ports.NamespaceGrant{NamespaceID: ns}
		var p string
		if err = rows.Scan(&g.PrincipalID, &p); err != nil {
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
	rows, err = tx.QueryContext(ctx, `SELECT id,principal_id,actor_json,expires_at,revoked FROM credentials WHERE namespace_id=? AND parent_digest IS NULL ORDER BY id LIMIT 101`, ns.String())
	if err != nil {
		return result, err
	}
	for rows.Next() {
		c := ports.Credential{NamespaceID: ns}
		var id, a string
		var expires int64
		if err = rows.Scan(&id, &c.PrincipalID, &a, &expires, &c.Revoked); err != nil {
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
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT namespace_version,principal_id,actor_json,operation,target,recorded_at FROM security_audit WHERE namespace_id=? ORDER BY namespace_version DESC LIMIT 101`, ns.String())
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var a ports.SecurityAudit
		var actor string
		var at int64
		if err = rows.Scan(&a.NamespaceVersion, &a.PrincipalID, &actor, &a.Operation, &a.Target, &at); err != nil {
			rows.Close()
			return result, err
		}
		if err = unmarshalJSON(actor, &a.Actor); err != nil {
			rows.Close()
			return result, err
		}
		a.RecordedAt = decodeTime(at)
		result.Audit = append(result.Audit, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	result.Truncated = len(result.Grants) > 100 || len(result.Credentials) > 100 || len(result.Audit) > 100
	result.Grants = result.Grants[:min(len(result.Grants), 100)]
	result.Credentials = result.Credentials[:min(len(result.Credentials), 100)]
	result.Audit = result.Audit[:min(len(result.Audit), 100)]
	return result, tx.Commit()
}

var _ ports.SecurityAdministrationStore = (*Store)(nil)
