package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

func (s *Store) GetCredential(ctx context.Context, digest string) (ports.Credential, error) {
	var c ports.Credential
	var rawID, rawNS, actor string
	var expires int64
	err := s.db.QueryRowContext(ctx, `SELECT c.id,c.namespace_id,c.principal_id,c.actor_json,c.expires_at,c.revoked,COALESCE(c.parent_digest,'') FROM credentials c JOIN principals p ON p.id=c.principal_id WHERE c.digest=? AND p.lifecycle='active'`, digest).Scan(&rawID, &rawNS, &c.PrincipalID, &actor, &expires, &c.Revoked, &c.ParentDigest)
	if err != nil {
		return c, mapSQLError("resolve credential", err)
	}
	c.NamespaceID, err = domain.ParseID(rawNS)
	if err != nil {
		return c, err
	}
	c.ID, err = domain.ParseID(rawID)
	if err != nil {
		return c, err
	}
	err = unmarshalJSON(actor, &c.Actor)
	c.ExpiresAt = decodeTime(expires)
	c.Digest = digest
	return c, err
}
func (s *Store) PutCredential(ctx context.Context, c ports.Credential) error {
	if err := c.ID.Validate(); err != nil {
		return err
	}
	if err := c.Actor.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = ensurePrincipal(ctx, tx, c.PrincipalID); err != nil {
		return err
	}
	actor, err := marshalJSON(c.Actor)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO credentials(id,namespace_id,principal_id,actor_json,digest,expires_at,revoked,parent_digest) VALUES(?,?,?,?,?,?,?,?)`, c.ID.String(), c.NamespaceID.String(), c.PrincipalID, actor, c.Digest, encodeTime(c.ExpiresAt), boolInt(c.Revoked), nullableString(c.ParentDigest))
	if err != nil {
		return mapSQLError("create credential", err)
	}
	return tx.Commit()
}
func (s *Store) RevokeCredential(ctx context.Context, id domain.ID) error {
	_, err := s.db.ExecContext(ctx, `UPDATE credentials SET revoked=1 WHERE id=?`, id.String())
	return err
}
func (s *Store) RevokeScopedCredential(ctx context.Context, ns, id domain.ID) error {
	result, err := s.db.ExecContext(ctx, `UPDATE credentials SET revoked=1 WHERE id=? AND namespace_id=?`, id.String(), ns.String())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return domain.NewError(domain.ErrorCodeNotFound, "credential not found")
	}
	return err
}
func (s *Store) PutGrant(ctx context.Context, g ports.NamespaceGrant) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = ensurePrincipal(ctx, tx, g.PrincipalID); err != nil {
		return err
	}
	payload, err := marshalJSON(g.Permissions)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO namespace_grants(namespace_id,principal_id,permissions_json) VALUES(?,?,?) ON CONFLICT(namespace_id,principal_id) DO UPDATE SET permissions_json=excluded.permissions_json`, g.NamespaceID.String(), g.PrincipalID, payload)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) GetGrant(ctx context.Context, ns domain.ID, principal string) (ports.NamespaceGrant, error) {
	g := ports.NamespaceGrant{NamespaceID: ns, PrincipalID: principal}
	var permissions string
	err := s.db.QueryRowContext(ctx, `SELECT g.permissions_json FROM namespace_grants g JOIN namespaces n ON n.id=g.namespace_id JOIN principals p ON p.id=g.principal_id WHERE g.namespace_id=? AND g.principal_id=? AND n.lifecycle='active' AND p.lifecycle='active'`, ns.String(), principal).Scan(&permissions)
	if err != nil {
		return g, mapSQLError("read namespace grant", err)
	}
	return g, unmarshalJSON(permissions, &g.Permissions)
}
func (s *Store) ListNamespaces(ctx context.Context, principal string) ([]ports.Namespace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT n.id,n.name,g.permissions_json FROM namespaces n JOIN namespace_grants g ON g.namespace_id=n.id WHERE g.principal_id=? AND n.lifecycle='active' ORDER BY n.name,n.id`, principal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ports.Namespace{}
	for rows.Next() {
		var raw, name, payload string
		if err = rows.Scan(&raw, &name, &payload); err != nil {
			return nil, err
		}
		var ps []ports.Permission
		if err = unmarshalJSON(payload, &ps); err != nil {
			return nil, err
		}
		allowed := false
		for _, p := range ps {
			if p == ports.PermissionStateRead {
				allowed = true
			}
		}
		if !allowed {
			continue
		}
		id, err := domain.ParseID(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, ports.Namespace{ID: id, Name: name})
	}
	return out, rows.Err()
}
func (s *Store) PutNamespace(ctx context.Context, n ports.Namespace) error {
	if err := n.ID.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO namespaces(id,name,lifecycle,version,created_at,updated_at) VALUES(?,?,'active',1,?,?)`, n.ID.String(), n.Name, time.Now().UnixMicro(), time.Now().UnixMicro())
	return mapSQLError("create namespace", err)
}

var _ ports.SecurityStore = (*Store)(nil)

func (s *Store) BootstrapSecurity(ctx context.Context, n ports.Namespace, g ports.NamespaceGrant, c ports.Credential) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM credentials WHERE principal_id=?`, g.PrincipalID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().UnixMicro()
	if _, err = tx.ExecContext(ctx, `INSERT INTO namespaces(id,name,lifecycle,version,created_at,updated_at) VALUES(?,?,'active',1,?,?) ON CONFLICT(id) DO NOTHING`, n.ID.String(), n.Name, now, now); err != nil {
		return err
	}
	if err = ensurePrincipal(ctx, tx, g.PrincipalID); err != nil {
		return err
	}
	perms, err := marshalJSON(g.Permissions)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO namespace_grants(namespace_id,principal_id,permissions_json) VALUES(?,?,?)`, g.NamespaceID.String(), g.PrincipalID, perms); err != nil {
		return err
	}
	actor, err := marshalJSON(c.Actor)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO credentials(id,namespace_id,principal_id,actor_json,digest,expires_at,revoked) VALUES(?,?,?,?,?,?,0)`, c.ID.String(), c.NamespaceID.String(), c.PrincipalID, actor, c.Digest, encodeTime(c.ExpiresAt)); err != nil {
		return err
	}
	return tx.Commit()
}
