package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// TenantPlacementImport partitions a drained, current-version shared schema
// within the same PostgreSQL database and the same logical WOS installation.
// It is not a transfer to a different issuer, database or installation.
type TenantPlacementImport struct {
	SourceSchema   string `json:"source_schema"`
	Reason         string `json:"reason"`
	WritersStopped bool   `json:"writers_stopped"`
}

type TenantPlacementTableResult struct {
	TenantID domain.ID `json:"tenant_id"`
	Table    string    `json:"table"`
	Rows     int64     `json:"rows"`
	SHA256   string    `json:"sha256"`
}

type TenantPlacementImportResult struct {
	SourceSchema      string                       `json:"source_schema"`
	SourceWriteFenced bool                         `json:"source_write_fenced"`
	Placements        []TenantPlacement            `json:"placements"`
	Tables            []TenantPlacementTableResult `json:"tables"`
}

type placementColumn struct {
	Name, Type, Default, Identity, Generated string
	NotNull                                  bool
}
type placementTable struct {
	Name    string
	Columns []placementColumn
}
type placementConstraint struct {
	Table, Name, Kind, Definition string
}
type placementForeignKey struct {
	Table, Name, ReferencedTable string
	Columns, ReferencedColumns   []string
	Deferrable, Deferred         bool
}

const sharedSourceFence = "wos_shared_source_write_fence"

func (c *TenantCatalog) validPlacementSource(source string) bool {
	return tenantCatalogSchemaPattern.MatchString(source) && source != c.schema && !strings.HasPrefix(source, "pg_") && !strings.HasPrefix(source, "wos_t_")
}

// CheckTenantPlacementSource is a read-only precondition for the operator's
// explicit source migration. In particular, a typo must never initialize a new
// schema and a catalog must never be mistaken for application storage.
func (c *TenantCatalog) CheckTenantPlacementSource(ctx context.Context, source string) error {
	if !c.validPlacementSource(source) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "invalid shared source schema")
	}
	var exists, migrations, identity, fenced bool
	err := c.db.QueryRowContext(ctx, `SELECT
 EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1),
 EXISTS(SELECT 1 FROM pg_class t JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=$1 AND t.relname='schema_migrations' AND t.relkind='r'),
 EXISTS(SELECT 1 FROM pg_class t JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=$1 AND t.relname IN ('namespaces','tenants') AND t.relkind='r'),
 EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname=$1 AND p.proname=$2)
`, source, sharedSourceFence).Scan(&exists, &migrations, &identity, &fenced)
	if err != nil {
		return mapSQLError("inspect shared source schema", err)
	}
	if !exists || !migrations || !identity {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "source must be an existing WOS application schema")
	}
	if fenced {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "shared source was already fenced by a Tenant placement; inspect the catalog instead of importing it again")
	}
	return nil
}

// ApplyTenantPlacement atomically copies every Tenant into its own suspended
// catalog placement and installs a persistent DML/TRUNCATE fence on the source.
// Source rows, historical JSON, signatures, digests and public issuer identity
// remain unchanged. No application data or issuer private key is deleted.
//
// The operator must first stop all source APIs/workers, retire old database
// writers and take a restorable backup. Table locks exclude concurrent DML;
// ALWAYS triggers reject subsequent writes even from a stale pooled connection.
// Administrative DDL/superuser access remains an operator trust boundary.
func (c *TenantCatalog) ApplyTenantPlacement(ctx context.Context, request TenantPlacementImport) (TenantPlacementImportResult, error) {
	result := TenantPlacementImportResult{SourceSchema: request.SourceSchema}
	if !request.WritersStopped || strings.TrimSpace(request.Reason) == "" || len(request.Reason) > 4096 {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "shared import requires stopped writers and a reason of at most 4096 bytes")
	}
	if err := c.CheckTenantPlacementSource(ctx, request.SourceSchema); err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return result, mapSQLError("begin shared Tenant placement", err)
	}
	defer tx.Rollback()
	// Copy and digest scans can exceed normal request timeouts. The operation's
	// explicit context bounds the complete transaction; locks never wait.
	if _, err := tx.ExecContext(ctx, `SET LOCAL statement_timeout=0; SET LOCAL row_security=off`); err != nil {
		return result, err
	}
	var locked bool
	if err := tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock($1)`, tenantLifecycleKey(request.SourceSchema, "shared-placement")).Scan(&locked); err != nil || !locked {
		if err != nil {
			return result, err
		}
		return result, domain.NewError(domain.ErrorCodePreconditionFailed, "another shared source import is active")
	}
	sourceTables, err := placementTables(ctx, tx, request.SourceSchema)
	if err != nil {
		return result, err
	}
	var lockTables []string
	for _, table := range sourceTables {
		lockTables = append(lockTables, placementQualified(request.SourceSchema, table.Name))
	}
	if len(lockTables) == 0 {
		return result, domain.NewError(domain.ErrorCodePreconditionFailed, "shared source has no application tables")
	}
	if _, err := tx.ExecContext(ctx, `LOCK TABLE `+strings.Join(lockTables, ",")+` IN ACCESS EXCLUSIVE MODE NOWAIT`); err != nil {
		return result, mapSQLError("source writers must be stopped before placement", err)
	}
	if err := validatePlacementMigrations(ctx, tx, request.SourceSchema); err != nil {
		return result, err
	}
	var triggers int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pg_trigger g JOIN pg_class t ON t.oid=g.tgrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=$1 AND NOT g.tgisinternal`, request.SourceSchema).Scan(&triggers); err != nil {
		return result, err
	}
	if triggers != 0 {
		return result, domain.NewError(domain.ErrorCodePreconditionFailed, "shared source contains unsupported user triggers or a prior write fence")
	}
	placements, err := placementIdentities(ctx, tx, request.SourceSchema)
	if err != nil {
		return result, err
	}
	sourceConstraints, sourceFKs, err := placementConstraints(ctx, tx, request.SourceSchema)
	if err != nil {
		return result, err
	}
	for _, p := range placements {
		for _, key := range []int64{tenantLifecycleKey(c.schema, p.TenantID.String()), tenantInitializationKey(c.schema, p.TenantID)} {
			if err := tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock($1)`, key).Scan(&locked); err != nil || !locked {
				if err != nil {
					return result, err
				}
				return result, domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant is already in use by another lifecycle operation")
			}
		}
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM `+placementQualified(c.schema, "tenant_placements")+` WHERE tenant_id=$1) OR EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$2)`, p.TenantID.String(), p.Schema).Scan(&exists); err != nil {
			return result, err
		}
		if exists {
			return result, domain.NewError(domain.ErrorCodeAlreadyExists, "Tenant placement or target schema already exists; importing over existing state is forbidden")
		}
		if err := initializePlacementSchema(ctx, tx, p.Schema); err != nil {
			return result, err
		}
		targetTables, err := placementTables(ctx, tx, p.Schema)
		if err != nil {
			return result, err
		}
		targetConstraints, targetFKs, err := placementConstraints(ctx, tx, p.Schema)
		if err != nil {
			return result, err
		}
		if !reflect.DeepEqual(sourceTables, targetTables) || !reflect.DeepEqual(sourceConstraints, targetConstraints) {
			return result, domain.NewError(domain.ErrorCodePreconditionFailed, "source tables, columns or constraints differ from the compiled WOS schema")
		}
		predicates, err := placementPredicates(sourceTables, sourceFKs, request.SourceSchema)
		if err != nil {
			return result, err
		}
		for _, fk := range targetFKs {
			if _, err := tx.ExecContext(ctx, `ALTER TABLE `+placementQualified(p.Schema, fk.Table)+` ALTER CONSTRAINT `+placementQuote(fk.Name)+` DEFERRABLE INITIALLY DEFERRED`); err != nil {
				return result, err
			}
		}
		if _, err := tx.ExecContext(ctx, `SET CONSTRAINTS ALL DEFERRED`); err != nil {
			return result, err
		}
		for _, table := range sourceTables {
			if table.Name == "schema_migrations" {
				continue
			}
			predicate := predicates[table.Name]
			args := placementArguments(predicate, p.TenantID)
			query := `INSERT INTO ` + placementQualified(p.Schema, table.Name) + ` SELECT s.* FROM ` + placementQualified(request.SourceSchema, table.Name) + ` s WHERE ` + predicate
			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return result, mapSQLError("copy Tenant table "+table.Name, err)
			}
			count, digest, err := placementDigest(ctx, tx, request.SourceSchema, table.Name, predicate, args)
			if err != nil {
				return result, err
			}
			targetCount, targetDigest, err := placementDigest(ctx, tx, p.Schema, table.Name, "TRUE", nil)
			if err != nil {
				return result, err
			}
			if count != targetCount || digest != targetDigest {
				return result, domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant table copy did not preserve exact rows")
			}
			result.Tables = append(result.Tables, TenantPlacementTableResult{p.TenantID, table.Name, count, digest})
		}
		if _, err := tx.ExecContext(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); err != nil {
			return result, mapSQLError("verify Tenant foreign-key closure", err)
		}
		for _, fk := range targetFKs {
			flags := "NOT DEFERRABLE"
			if fk.Deferrable {
				flags = "DEFERRABLE INITIALLY IMMEDIATE"
				if fk.Deferred {
					flags = "DEFERRABLE INITIALLY DEFERRED"
				}
			}
			if _, err := tx.ExecContext(ctx, `ALTER TABLE `+placementQualified(p.Schema, fk.Table)+` ALTER CONSTRAINT `+placementQuote(fk.Name)+` `+flags); err != nil {
				return result, err
			}
		}
		if _, err := tx.ExecContext(ctx, `ALTER TABLE `+placementQualified(p.Schema, "tenants")+` ADD CONSTRAINT wos_tenant_identity_binding CHECK (id='`+p.TenantID.String()+`'); CREATE UNIQUE INDEX wos_single_tenant ON `+placementQualified(p.Schema, "tenants")+` ((1))`); err != nil {
			return result, err
		}
		if _, err := tx.ExecContext(ctx, `SET LOCAL search_path=`+placementQuote(c.schema)); err != nil {
			return result, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO tenant_placements(`+tenantPlacementColumns+`) VALUES($1,$2,$3,$4,$5,$6,$7)`, p.TenantID.String(), p.Schema, p.Name, p.State, p.Version, p.CreatedAt, p.UpdatedAt); err != nil {
			return result, err
		}
		if err := c.audit(ctx, tx, p, "imported_from_"+request.SourceSchema, request.Reason); err != nil {
			return result, err
		}
		result.Placements = append(result.Placements, p)
	}
	if err := fencePlacementSource(ctx, tx, request.SourceSchema, sourceTables); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, mapSQLError("commit shared Tenant placement", err)
	}
	result.SourceWriteFenced = true
	return result, nil
}

func placementQuote(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }
func placementQualified(schema, table string) string {
	return placementQuote(schema) + "." + placementQuote(table)
}
func placementArguments(predicate string, id domain.ID) []any {
	if strings.Contains(predicate, "$1") {
		return []any{id.String()}
	}
	return nil
}

func placementIdentities(ctx context.Context, tx *sql.Tx, schema string) ([]TenantPlacement, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,name FROM `+placementQualified(schema, "tenants")+` ORDER BY id COLLATE "C"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []TenantPlacement
	now := time.Now().UTC()
	for rows.Next() {
		var p TenantPlacement
		if err := rows.Scan(&p.TenantID, &p.Name); err != nil {
			return nil, err
		}
		if p.TenantID.Validate() != nil || strings.TrimSpace(p.Name) == "" {
			return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "invalid Tenant identity in source schema")
		}
		p.Schema, p.State, p.Version = tenantDataSchema(p.TenantID), TenantSuspended, 1
		p.CreatedAt, p.UpdatedAt = now, now
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "shared source contains no Tenants")
	}
	return result, nil
}

func placementTables(ctx context.Context, tx *sql.Tx, schema string) ([]placementTable, error) {
	rows, err := tx.QueryContext(ctx, `SELECT t.relname,t.relkind::text,t.relrowsecurity,t.relforcerowsecurity,EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=t.oid),EXISTS(SELECT 1 FROM pg_inherits i WHERE i.inhrelid=t.oid OR i.inhparent=t.oid) FROM pg_class t JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=$1 AND t.relkind IN ('r','p','v','m','f','S') ORDER BY t.relname COLLATE "C"`, schema)
	if err != nil {
		return nil, err
	}
	var tables []placementTable
	for rows.Next() {
		var table placementTable
		var kind string
		var rowSecurity, forcedRowSecurity, policy, inheritance bool
		if err := rows.Scan(&table.Name, &kind, &rowSecurity, &forcedRowSecurity, &policy, &inheritance); err != nil {
			rows.Close()
			return nil, err
		}
		if kind != "r" {
			rows.Close()
			return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "source contains unsupported views, partitions, sequences or foreign tables")
		}
		if rowSecurity || forcedRowSecurity || policy || inheritance {
			rows.Close()
			return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "source contains unsupported row-security policies or table inheritance")
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range tables {
		columns, err := tx.QueryContext(ctx, `SELECT a.attname,format_type(a.atttypid,a.atttypmod),a.attnotnull,COALESCE(pg_get_expr(d.adbin,d.adrelid),''),a.attidentity::text,a.attgenerated::text FROM pg_attribute a JOIN pg_class t ON t.oid=a.attrelid JOIN pg_namespace n ON n.oid=t.relnamespace LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE n.nspname=$1 AND t.relname=$2 AND a.attnum>0 AND NOT a.attisdropped ORDER BY a.attnum`, schema, tables[i].Name)
		if err != nil {
			return nil, err
		}
		for columns.Next() {
			var column placementColumn
			if err := columns.Scan(&column.Name, &column.Type, &column.NotNull, &column.Default, &column.Identity, &column.Generated); err != nil {
				columns.Close()
				return nil, err
			}
			if column.Identity != "" || column.Generated != "" {
				columns.Close()
				return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "unsupported generated source column")
			}
			tables[i].Columns = append(tables[i].Columns, column)
		}
		if err := columns.Err(); err != nil {
			columns.Close()
			return nil, err
		}
		if err := columns.Close(); err != nil {
			return nil, err
		}
	}
	return tables, nil
}

func validatePlacementMigrations(ctx context.Context, tx *sql.Tx, schema string) error {
	expected, err := loadMigrations()
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT version,name,checksum FROM `+placementQualified(schema, "schema_migrations")+` ORDER BY version`)
	if err != nil {
		return err
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		var version int64
		var name, checksum string
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			return err
		}
		if index >= len(expected) || expected[index].version != version || expected[index].name != name || expected[index].checksum != checksum {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "migration ledger does not match this binary")
		}
		index++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if index != len(expected) {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "migration ledger is incomplete; migrate the stopped schema before use")
	}
	return nil
}

func initializePlacementSchema(ctx context.Context, tx *sql.Tx, schema string) error {
	if _, err := tx.ExecContext(ctx, `CREATE SCHEMA `+placementQuote(schema)+`; SET LOCAL search_path=`+placementQuote(schema)+`; CREATE TABLE schema_migrations(version BIGINT PRIMARY KEY,name TEXT NOT NULL,checksum TEXT NOT NULL,applied_at BIGINT NOT NULL)`); err != nil {
		return err
	}
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		// Every destination is empty. Existing-data Go backfills have no rows to
		// transform; source current-state rows are copied verbatim afterwards.
		if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
			return mapSQLError(fmt.Sprintf("initialize Tenant migration %d", migration.version), err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES($1,$2,$3,$4)`, migration.version, migration.name, migration.checksum, time.Now().UTC().UnixMicro()); err != nil {
			return err
		}
	}
	return nil
}

func placementConstraints(ctx context.Context, tx *sql.Tx, schema string) ([]placementConstraint, []placementForeignKey, error) {
	rows, err := tx.QueryContext(ctx, `SELECT t.relname,c.conname,c.contype::text,pg_get_constraintdef(c.oid,true),c.convalidated,COALESCE(rn.nspname,''),COALESCE(rt.relname,''),c.condeferrable,c.condeferred,COALESCE((SELECT json_agg(a.attname ORDER BY x.ordinality)::text FROM unnest(c.conkey) WITH ORDINALITY x(attnum,ordinality) JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=x.attnum),'[]'),COALESCE((SELECT json_agg(a.attname ORDER BY x.ordinality)::text FROM unnest(c.confkey) WITH ORDINALITY x(attnum,ordinality) JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=x.attnum),'[]') FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace LEFT JOIN pg_class rt ON rt.oid=c.confrelid LEFT JOIN pg_namespace rn ON rn.oid=rt.relnamespace WHERE n.nspname=$1 ORDER BY t.relname COLLATE "C",c.conname COLLATE "C"`, schema)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var constraints []placementConstraint
	var fks []placementForeignKey
	for rows.Next() {
		var constraint placementConstraint
		var fk placementForeignKey
		var validated bool
		var referenceSchema, columns, references string
		if err := rows.Scan(&constraint.Table, &constraint.Name, &constraint.Kind, &constraint.Definition, &validated, &referenceSchema, &fk.ReferencedTable, &fk.Deferrable, &fk.Deferred, &columns, &references); err != nil {
			return nil, nil, err
		}
		if !validated || constraint.Kind == "f" && referenceSchema != schema {
			return nil, nil, domain.NewError(domain.ErrorCodePreconditionFailed, "source has unvalidated or cross-schema constraints")
		}
		constraint.Definition = strings.ReplaceAll(constraint.Definition, placementQuote(schema)+".", "")
		constraint.Definition = strings.ReplaceAll(constraint.Definition, schema+".", "")
		constraints = append(constraints, constraint)
		if constraint.Kind == "f" {
			fk.Table, fk.Name = constraint.Table, constraint.Name
			if json.Unmarshal([]byte(columns), &fk.Columns) != nil || json.Unmarshal([]byte(references), &fk.ReferencedColumns) != nil {
				return nil, nil, domain.NewError(domain.ErrorCodePreconditionFailed, "invalid foreign-key inventory")
			}
			fks = append(fks, fk)
		}
	}
	return constraints, fks, rows.Err()
}

func placementPredicates(tables []placementTable, fks []placementForeignKey, source string) (map[string]string, error) {
	result := map[string]string{}
	for _, table := range tables {
		for _, column := range table.Columns {
			if column.Name == "tenant_id" {
				result[table.Name] = "s.tenant_id=$1"
			}
		}
		switch table.Name {
		case "schema_migrations":
			result[table.Name] = "FALSE"
		case "tenants":
			result[table.Name] = "s.id=$1"
		case "server_protocol_identity", "server_issuer_public_keys", "server_issuer_recoveries":
			result[table.Name] = "TRUE"
		case "delivery_attempts":
			result[table.Name] = "s.delivery_id IN (SELECT id FROM " + placementQualified(source, "integration_deliveries") + " WHERE tenant_id=$1)"
		case "trigger_firings":
			result[table.Name] = "s.trigger_id IN (SELECT id FROM " + placementQualified(source, "triggers") + " WHERE tenant_id=$1)"
		}
	}
	var principals []string
	for _, fk := range fks {
		if fk.ReferencedTable != "principals" {
			continue
		}
		if len(fk.Columns) != 1 || len(fk.ReferencedColumns) != 1 || fk.ReferencedColumns[0] != "id" || result[fk.Table] != "s.tenant_id=$1" {
			return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "unsupported Principal reference closure")
		}
		column := placementQuote(fk.Columns[0])
		principals = append(principals, "SELECT "+column+" FROM "+placementQualified(source, fk.Table)+" WHERE tenant_id=$1 AND "+column+" IS NOT NULL")
	}
	// This is the same installation's public issuer subject, not another
	// customer's identity. The private issuer key is never present in storage.
	principals = append(principals, "SELECT 'wos-server:' || (state_json::jsonb->>'server_id') FROM "+placementQualified(source, "server_protocol_identity"))
	sort.Strings(principals)
	result["principals"] = "s.id IN (" + strings.Join(principals, " UNION ") + ")"
	for _, table := range tables {
		if result[table.Name] == "" {
			return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "unclassified source table cannot be safely assigned to a Tenant")
		}
	}
	return result, nil
}

func placementDigest(ctx context.Context, tx *sql.Tx, schema, table, predicate string, args []any) (int64, string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT row_to_json(s)::text FROM `+placementQualified(schema, table)+` s WHERE `+predicate+` ORDER BY convert_to(row_to_json(s)::text,'UTF8')`, args...)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	hash := sha256.New()
	var count int64
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			return 0, "", err
		}
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(row)))
		hash.Write(size[:])
		hash.Write([]byte(row))
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, "", err
	}
	return count, hex.EncodeToString(hash.Sum(nil)), nil
}

func fencePlacementSource(ctx context.Context, tx *sql.Tx, schema string, tables []placementTable) error {
	function := placementQualified(schema, sharedSourceFence)
	if _, err := tx.ExecContext(ctx, `CREATE FUNCTION `+function+`() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $wos_fence$ BEGIN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='shared WOS source is read-only after Tenant placement; use the catalog-backed schema'; END; $wos_fence$`); err != nil {
		return err
	}
	for _, table := range tables {
		qualified := placementQualified(schema, table.Name)
		if _, err := tx.ExecContext(ctx, `CREATE TRIGGER `+sharedSourceFence+` BEFORE INSERT OR UPDATE OR DELETE OR TRUNCATE ON `+qualified+` FOR EACH STATEMENT EXECUTE FUNCTION `+function+`(); ALTER TABLE `+qualified+` ENABLE ALWAYS TRIGGER `+sharedSourceFence); err != nil {
			return err
		}
	}
	return nil
}
