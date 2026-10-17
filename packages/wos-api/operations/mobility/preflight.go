package mobility

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type foreignKey struct {
	Table  string   `json:"table"`
	Parent string   `json:"parent"`
	From   []string `json:"from"`
	To     []string `json:"to"`
}

func foreignKeys(ctx context.Context, db queryer, tables []table, backend string) ([]foreignKey, error) {
	var result []foreignKey
	for _, t := range tables {
		query := "SELECT id,seq,\"table\",\"from\",\"to\" FROM pragma_foreign_key_list('" + t.Name + "') ORDER BY id,seq"
		if backend == "postgres" {
			query = `SELECT k.oid::bigint,child.ordinality-1,p.relname,a.attname,b.attname FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_class p ON p.oid=k.confrelid CROSS JOIN LATERAL unnest(k.conkey) WITH ORDINALITY child(attnum,ordinality) JOIN LATERAL unnest(k.confkey) WITH ORDINALITY parent(attnum,ordinality) ON parent.ordinality=child.ordinality JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum=child.attnum JOIN pg_attribute b ON b.attrelid=p.oid AND b.attnum=parent.attnum WHERE k.contype='f' AND n.nspname='public' AND c.relname='` + t.Name + `' ORDER BY k.oid,child.ordinality`
		}
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		var last int64 = -1
		for rows.Next() {
			var id, seq int64
			var parent, from, to string
			if err = rows.Scan(&id, &seq, &parent, &from, &to); err != nil {
				rows.Close()
				return nil, err
			}
			if !identifierPattern.MatchString(parent) || !identifierPattern.MatchString(from) || !identifierPattern.MatchString(to) {
				rows.Close()
				return nil, fmt.Errorf("unsupported foreign key identifier in %s", t.Name)
			}
			if id != last {
				result = append(result, foreignKey{Table: t.Name, Parent: parent})
				last = id
			}
			i := len(result) - 1
			result[i].From = append(result[i].From, from)
			result[i].To = append(result[i].To, to)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err = rows.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func namespaceInspection(ctx context.Context, db queryer, tables []table, backend, namespace string) (map[string]any, error) {
	placeholder := "?"
	if backend == "postgres" {
		placeholder = "$1"
	}
	count := func(query string) (int64, error) {
		var n int64
		err := db.QueryRowContext(ctx, query, namespace).Scan(&n)
		return n, err
	}
	hasColumn := func(name, col string) bool {
		for _, t := range tables {
			if t.Name == name {
				for _, c := range t.Columns {
					if c.Name == col {
						return true
					}
				}
			}
		}
		return false
	}
	keys, err := foreignKeys(ctx, db, tables, backend)
	if err != nil {
		return nil, err
	}
	type closure struct {
		foreignKey
		Missing          int64 `json:"missing_references"`
		OutsideNamespace int64 `json:"references_outside_namespace"`
		GlobalParent     bool  `json:"global_parent"`
	}
	var closureChecks []closure
	var missingTotal, outsideTotal int64
	for _, key := range keys {
		if !hasColumn(key.Table, "namespace_id") {
			continue
		}
		where := "c.namespace_id=" + placeholder
		var join []string
		for i, from := range key.From {
			where += " AND c." + quoted(from) + " IS NOT NULL"
			join = append(join, "p."+quoted(key.To[i])+"=c."+quoted(from))
		}
		missing, err := count("SELECT COUNT(*) FROM " + quoted(key.Table) + " c WHERE " + where + " AND NOT EXISTS(SELECT 1 FROM " + quoted(key.Parent) + " p WHERE " + strings.Join(join, " AND ") + ")")
		if err != nil {
			return nil, err
		}
		check := closure{foreignKey: key, Missing: missing, GlobalParent: !hasColumn(key.Parent, "namespace_id")}
		if !check.GlobalParent {
			check.OutsideNamespace, err = count("SELECT COUNT(*) FROM " + quoted(key.Table) + " c JOIN " + quoted(key.Parent) + " p ON " + strings.Join(join, " AND ") + " WHERE " + where + " AND p.namespace_id<>c.namespace_id")
			if err != nil {
				return nil, err
			}
		}
		missingTotal += check.Missing
		outsideTotal += check.OutsideNamespace
		closureChecks = append(closureChecks, check)
	}
	var principalQueries []string
	for _, t := range tables {
		if !hasColumn(t.Name, "namespace_id") {
			continue
		}
		for _, c := range t.Columns {
			if c.Name == "principal_id" || strings.HasSuffix(c.Name, "_principal_id") {
				principalQueries = append(principalQueries, "SELECT namespace_id,"+quoted(c.Name)+" AS principal_id FROM "+quoted(t.Name))
			}
		}
	}
	principalInventory := map[string]int64{}
	if len(principalQueries) > 0 {
		prefix := "WITH selected_namespace AS(SELECT " + placeholder + " AS id),principal_refs AS(" + strings.Join(principalQueries, " UNION ALL ") + ") "
		principalInventory["referenced_principals"], err = count(prefix + "SELECT COUNT(DISTINCT principal_id) FROM principal_refs WHERE namespace_id=(SELECT id FROM selected_namespace)")
		if err != nil {
			return nil, err
		}
		principalInventory["principals_also_referenced_by_other_namespaces"], err = count(prefix + "SELECT COUNT(DISTINCT selected.principal_id) FROM principal_refs selected WHERE selected.namespace_id=(SELECT id FROM selected_namespace) AND EXISTS(SELECT 1 FROM principal_refs other WHERE other.principal_id=selected.principal_id AND other.namespace_id<>selected.namespace_id)")
		if err != nil {
			return nil, err
		}
	}
	credentials := map[string]int64{}
	credentials["revoked"], err = count("SELECT COUNT(*) FROM credentials WHERE namespace_id=" + placeholder + " AND revoked<>0")
	if err != nil {
		return nil, err
	}
	credentials["browser_sessions"], err = count("SELECT COUNT(*) FROM credentials WHERE namespace_id=" + placeholder + " AND parent_digest IS NOT NULL")
	if err != nil {
		return nil, err
	}
	credentials["missing_or_cross_namespace_session_parents"], err = count("SELECT COUNT(*) FROM credentials c WHERE namespace_id=" + placeholder + " AND parent_digest IS NOT NULL AND NOT EXISTS(SELECT 1 FROM credentials p WHERE p.digest=c.parent_digest AND p.namespace_id=c.namespace_id AND p.principal_id=c.principal_id)")
	if err != nil {
		return nil, err
	}
	credentialReferences := map[string]int64{}
	for _, t := range tables {
		if !hasColumn(t.Name, "namespace_id") {
			continue
		}
		for _, c := range t.Columns {
			if c.Name != "credential_id" && c.Name != "original_credential_id" {
				continue
			}
			join := "p.id=c." + quoted(c.Name) + " AND p.namespace_id=c.namespace_id"
			principalCol := "principal_id"
			if hasColumn(t.Name, "holder_principal_id") {
				principalCol = "holder_principal_id"
			}
			if hasColumn(t.Name, principalCol) {
				join += " AND p.principal_id=c." + quoted(principalCol)
			}
			credentialReferences[t.Name+"."+c.Name], err = count("SELECT COUNT(*) FROM " + quoted(t.Name) + " c WHERE c.namespace_id=" + placeholder + " AND c." + quoted(c.Name) + " IS NOT NULL AND NOT EXISTS(SELECT 1 FROM credentials p WHERE " + join + ")")
			if err != nil {
				return nil, err
			}
		}
	}
	now := time.Now().UTC()
	if backend == "postgres" {
		var micros int64
		if err = db.QueryRowContext(ctx, "SELECT CAST(EXTRACT(EPOCH FROM statement_timestamp())*1000000 AS BIGINT)").Scan(&micros); err != nil {
			return nil, err
		}
		now = time.UnixMicro(micros).UTC()
	}
	authority := map[string]int64{}
	for _, name := range []string{"signed_work_contracts", "work_review_contracts", "work_contracts"} {
		query := "SELECT COUNT(*) FROM " + quoted(name) + " WHERE namespace_id=" + placeholder + " AND status='active' AND expires_at>" + fmt.Sprint(now.UnixMicro())
		authority[name+"_unexpired"], err = count(query)
		if err != nil {
			return nil, err
		}
	}
	outbox := map[string]int64{}
	for _, status := range []string{"pending", "leased", "exhausted", "succeeded"} {
		outbox[status], err = count("SELECT COUNT(*) FROM integration_deliveries WHERE namespace_id=" + placeholder + " AND status='" + status + "'")
		if err != nil {
			return nil, err
		}
	}
	outbox["indirect_attempts"], err = count("SELECT COUNT(*) FROM delivery_attempts a JOIN integration_deliveries d ON d.id=a.delivery_id WHERE d.namespace_id=" + placeholder)
	if err != nil {
		return nil, err
	}
	outbox["indirect_trigger_firings"], err = count("SELECT COUNT(*) FROM trigger_firings f JOIN triggers t ON t.id=f.trigger_id WHERE t.namespace_id=" + placeholder)
	if err != nil {
		return nil, err
	}
	type publicIssuer struct {
		identity
		KeyID string `json:"issuer_key_id"`
	}
	var history []publicIssuer
	rows, err := db.QueryContext(ctx, "SELECT key_id,server_id,fingerprint FROM server_issuer_public_keys ORDER BY key_id")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id publicIssuer
		if err = rows.Scan(&id.KeyID, &id.ServerID, &id.Fingerprint); err != nil {
			rows.Close()
			return nil, err
		}
		history = append(history, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	return map[string]any{"evaluated_at": now.Format(time.RFC3339Nano), "foreign_key_checks": closureChecks, "missing_scoped_references": missingTotal, "cross_namespace_references": outsideTotal, "principal_inventory": principalInventory, "credential_inventory": credentials, "invalid_credential_reference_counts": credentialReferences, "unexpired_authority": authority, "delivery_inventory": outbox, "historical_public_issuers": history, "logical_reference_limit": "SQL foreign keys and credential parent links checked; signed payload, documentary and polymorphic reference closure still requires a domain transfer protocol"}, nil
}
