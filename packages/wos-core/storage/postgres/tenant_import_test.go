package postgres

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func importTestID(t *testing.T) domain.ID {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatal(err)
	}
	raw[6], raw[8] = raw[6]&15|0x70, raw[8]&63|0x80
	return domain.MustParseID(fmt.Sprintf("%x-%x-%x-%x-%x", raw[:4], raw[4:6], raw[6:8], raw[8:10], raw[10:]))
}

type importFixture struct {
	source   *Store
	catalog  *TenantCatalog
	tenants  []domain.ID
	outcomes []domain.ID
	fact     domain.ID
	factRaw  string
	public   ed25519.PublicKey
	key      domain.ID
	request  TenantPlacementImport
}

func newImportFixture(t *testing.T) importFixture {
	t.Helper()
	requirePostgres(t)
	ctx := context.Background()
	f := importFixture{source: openTestStore(t, filepath.Join(t.TempDir(), "shared.db")), tenants: []domain.ID{importTestID(t), importTestID(t)}}
	if f.tenants[0] > f.tenants[1] {
		f.tenants[0], f.tenants[1] = f.tenants[1], f.tenants[0]
	}
	schema := "wos_import_" + strings.ReplaceAll(importTestID(t).String(), "-", "")
	var err error
	f.catalog, err = OpenTenantCatalog(ctx, os.Getenv("WOS_TEST_POSTGRES_DSN"), CatalogOptions{Schema: schema, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, id := range f.tenants {
			_, _ = f.source.db.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+placementQuote(tenantDataSchema(id))+` CASCADE`)
		}
		_, _ = f.source.db.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+placementQuote(schema)+` CASCADE`)
		_, _ = f.source.db.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+placementQuote(f.source.path)+` CASCADE`)
		_ = f.catalog.Close()
	})
	now := time.Now().UTC().Truncate(time.Microsecond)
	for i, id := range f.tenants {
		if _, err := f.source.db.ExecContext(ctx, `INSERT INTO tenants(id,name,lifecycle,version,created_at,updated_at) VALUES($1,$2,'active',1,$3,$3)`, id.String(), fmt.Sprintf("Customer %d", i), now.UnixMicro()); err != nil {
			t.Fatal(err)
		}
		principal := fmt.Sprintf("customer-%d", i)
		if _, err := f.source.db.ExecContext(ctx, `INSERT INTO principals(id,display_name,kind,lifecycle,created_at) VALUES($1,$1,'human','active',$2); INSERT INTO tenant_grants(tenant_id,principal_id,permissions_json) VALUES($3,$1,'["state:read"]')`, principal, now.UnixMicro(), id.String()); err != nil {
			t.Fatal(err)
		}
		outcome, err := domain.NewOutcome(importTestID(t), id, "Tenant work", "", "Verified delivery", domain.PriorityNormal, now)
		if err != nil {
			t.Fatal(err)
		}
		uow, err := f.source.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := uow.Outcomes().Insert(ctx, outcome); err != nil {
			uow.Rollback()
			t.Fatal(err)
		}
		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}
		f.outcomes = append(f.outcomes, outcome.ID)
		trigger, event, firing, endpoint, delivery := importTestID(t), importTestID(t), importTestID(t), importTestID(t), importTestID(t)
		if _, err := f.source.db.ExecContext(ctx, `
INSERT INTO webhook_endpoints(id,tenant_id,url,secret_ref,key_id) VALUES($1,$2,'https://example.invalid/hook','fixture-secret','fixture-key');
INSERT INTO triggers(id,tenant_id,outcome_id,version,enabled,record_json) VALUES($3,$2,$4,1,1,'{"authored":"namespace_id remains authored"}');
INSERT INTO integration_events(id,tenant_id,outcome_id,record_json) VALUES($5,$2,$4,'{"observation":"unchanged"}');
INSERT INTO trigger_firings(id,trigger_id,trigger_version,source_event_id,record_json) VALUES($6,$3,1,$5,'{"history":"unchanged"}');
INSERT INTO integration_deliveries(id,tenant_id,outcome_id,integration_event_id,endpoint_id,body_json,url,secret_ref,key_id,status,attempts,next_attempt,fencing_token) VALUES($7,$2,$4,$6,$1,'{"namespace_id":"authored"}','https://example.invalid/hook','fixture-secret','fixture-key','pending',1,0,1);
INSERT INTO delivery_attempts(delivery_id,fencing_token,lease_id,started_at,finished_at,result) VALUES($7,1,'old-attempt',1,2,'retry');
`, endpoint.String(), id.String(), trigger.String(), outcome.ID.String(), event.String(), firing.String(), delivery.String()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.source.db.ExecContext(ctx, `INSERT INTO principals(id,display_name,kind,lifecycle,created_at) VALUES('orphan','not part of any Tenant','human','active',1)`); err != nil {
		t.Fatal(err)
	}
	// A genuine historical signature stored as immutable bytes. This low-level
	// storage fixture tests copying/verification, not authorization to execute it.
	f.key, f.fact = importTestID(t), importTestID(t)
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	f.public = private.Public().(ed25519.PublicKey)
	fingerprint, _ := signing.Fingerprint(f.public)
	server := domain.ServerIdentity{ID: importTestID(t), IssuerKeyID: importTestID(t), PublicKey: base64.StdEncoding.EncodeToString(f.public), Fingerprint: fingerprint, CreatedAt: now}
	serverRaw, _ := json.Marshal(server)
	if _, err := f.source.db.ExecContext(ctx, `INSERT INTO server_protocol_identity(singleton,state_json) VALUES(1,$1); INSERT INTO server_issuer_public_keys(key_id,server_id,fingerprint,state_json) VALUES($2,$3,$4,$1); INSERT INTO principals(id,display_name,kind,lifecycle,created_at) VALUES($5,'Public installation issuer','service','active',$6)`, string(serverRaw), server.IssuerKeyID.String(), server.ID.String(), server.Fingerprint, server.PrincipalID(), now.UnixMicro()); err != nil {
		t.Fatal(err)
	}
	payload, err := signing.Canonical(map[string]any{"protocol_version": 2, "namespace_id": f.tenants[0].String(), "server_id": "historical-installation", "principal_id": "customer-0", "signer_key_id": f.key.String(), "outcome_id": f.outcomes[0].String(), "contract_kind": "execution", "contract_id": importTestID(t).String(), "work_item_id": importTestID(t).String(), "spec": map[string]any{"summary": "immutable legacy namespace_id"}})
	if err != nil {
		t.Fatal(err)
	}
	proof, _ := json.Marshal(signing.Proof{PayloadType: signing.LegacyContractSpec, KeyID: f.key.String(), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, signing.PAE(signing.LegacyContractSpec, payload)))})
	fact := domain.SignedFact{ID: f.fact, Scope: domain.Scope{TenantID: f.tenants[0], OutcomeID: f.outcomes[0]}, ContractID: importTestID(t), ContractKind: "execution", Kind: "specification", PrincipalID: "customer-0", KeyID: f.key, PayloadDigest: signing.Digest(payload), Payload: payload, Proof: proof, RecordedAt: now}
	raw, _ := json.Marshal(fact)
	var legacy map[string]json.RawMessage
	_ = json.Unmarshal(raw, &legacy)
	legacy["scope"] = json.RawMessage(fmt.Sprintf(`{"namespace_id":%q,"outcome_id":%q}`, f.tenants[0], f.outcomes[0]))
	raw, _ = json.Marshal(legacy)
	f.factRaw = string(raw)
	if _, err := f.source.db.ExecContext(ctx, `INSERT INTO signing_keys(tenant_id,id,principal_id,purpose,fingerprint,status,version,state_json) VALUES($1,$2,'customer-0','issuer',$3,'active',1,'{}'); INSERT INTO signed_protocol_facts(id,tenant_id,outcome_id,contract_id,contract_kind,kind,principal_id,key_id,idempotency_key,request_id,payload_digest,state_json) VALUES($4,$1,$5,$6,'execution','specification','customer-0',$2,'','',$7,$8)`, f.tenants[0].String(), f.key.String(), fingerprint, f.fact.String(), f.outcomes[0].String(), fact.ContractID.String(), fact.PayloadDigest, f.factRaw); err != nil {
		t.Fatal(err)
	}
	f.request = TenantPlacementImport{SourceSchema: f.source.path, Reason: "Offline fixture placement", WritersStopped: true}
	return f
}

func TestTenantPlacementImportPreservesProofsAndIsolatesRows(t *testing.T) {
	f := newImportFixture(t)
	ctx := context.Background()
	result, err := f.catalog.ApplyTenantPlacement(ctx, f.request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.SourceWriteFenced || len(result.Placements) != 2 || len(result.Tables) == 0 {
		t.Fatalf("incomplete placement result: %+v", result)
	}
	for i, placement := range result.Placements {
		if placement.TenantID != f.tenants[i] || placement.State != TenantSuspended || placement.Version != 1 {
			t.Fatalf("unexpected placement: %+v", placement)
		}
		if _, err := f.catalog.AcquireTenantLease(ctx, placement.TenantID); err == nil {
			t.Fatal("imported Tenant was exposed before activation")
		}
		store, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: placement.Schema, TenantID: placement.TenantID})
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		for _, table := range []string{"tenants", "outcomes", "trigger_firings", "integration_deliveries", "delivery_attempts", "server_protocol_identity", "server_issuer_public_keys"} {
			var count int
			if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM `+placementQuote(table)).Scan(&count); err != nil || count != 1 {
				t.Fatalf("%s crossed Tenant boundaries: count=%d err=%v", table, count, err)
			}
		}
		var principal string
		if err := store.db.QueryRowContext(ctx, `SELECT id FROM principals WHERE kind='human'`).Scan(&principal); err != nil || principal != fmt.Sprintf("customer-%d", i) {
			t.Fatalf("another Tenant's Principal was copied: %s %v", principal, err)
		}
		var principalCount int
		if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM principals`).Scan(&principalCount); err != nil || principalCount != 2 {
			t.Fatal("Principal closure must contain only this customer and the shared installation issuer", err)
		}
		if _, err := store.db.ExecContext(ctx, `INSERT INTO tenants(id,name,lifecycle,version,created_at,updated_at) VALUES($1,'wrong','active',1,1,1)`, importTestID(t).String()); err == nil {
			t.Fatal("target schema accepted a second Tenant identity")
		}
		if i == 0 {
			var copied string
			if err := store.db.QueryRowContext(ctx, `SELECT state_json FROM signed_protocol_facts WHERE id=$1`, f.fact.String()).Scan(&copied); err != nil || copied != f.factRaw {
				t.Fatal("immutable historical record changed", err)
			}
			var fact domain.SignedFact
			if err := signing.DecodeStoredJSON([]byte(copied), &fact, 0); err != nil || fact.Validate() != nil || fact.Scope.TenantID != placement.TenantID {
				t.Fatal("historical fact could not be read", err)
			}
			var proof signing.Proof
			_ = json.Unmarshal(fact.Proof, &proof)
			document := signing.Document{Payload: fact.Payload, Proof: proof}
			envelope, err := document.Envelope()
			if err != nil {
				t.Fatal(err)
			}
			verified, err := signing.VerifyHistorical(envelope, signing.ContractSpec, f.key.String(), f.public)
			if err != nil || !bytes.Equal(verified, fact.Payload) {
				t.Fatal("migrated historical signature failed exact verification", err)
			}
			if _, err := signing.Verify(envelope, signing.ContractSpec, f.key.String(), f.public); err == nil {
				t.Fatal("migrated historical proof became executable current intent")
			}
		}
	}
	for _, query := range []string{`UPDATE tenants SET name=name`, `DELETE FROM principals WHERE false`, `INSERT INTO principals(id,display_name,kind,lifecycle,created_at) VALUES('stale-writer','stale','human','active',1)`, `TRUNCATE tenants CASCADE`} {
		if _, err := f.source.db.ExecContext(ctx, query); err == nil {
			t.Fatalf("stale source writer passed persistent fence: %s", query)
		}
	}
	connection, err := f.source.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(ctx, `SET session_replication_role=replica`); err != nil {
		connection.Close()
		t.Fatal(err)
	}
	_, staleErr := connection.ExecContext(ctx, `UPDATE tenants SET name=name`)
	_, resetErr := connection.ExecContext(ctx, `SET session_replication_role=origin`)
	connection.Close()
	if staleErr == nil || resetErr != nil {
		t.Fatal("source fence did not survive replication-role trigger bypass", staleErr, resetErr)
	}
	var source string
	if err := f.source.db.QueryRowContext(ctx, `SELECT state_json FROM signed_protocol_facts WHERE id=$1`, f.fact.String()).Scan(&source); err != nil || source != f.factRaw {
		t.Fatal("source recovery data lost", err)
	}
	if _, err := f.catalog.ApplyTenantPlacement(ctx, f.request); err == nil {
		t.Fatal("re-imported a fenced source")
	}
}

func TestTenantPlacementImportPreservesAnExistingTarget(t *testing.T) {
	f := newImportFixture(t)
	ctx := context.Background()
	existing := tenantDataSchema(f.tenants[1])
	if _, err := f.source.db.ExecContext(ctx, `CREATE SCHEMA `+placementQuote(existing)+`; CREATE TABLE `+placementQualified(existing, "sentinel")+`(content TEXT); INSERT INTO `+placementQualified(existing, "sentinel")+` VALUES('must survive')`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.catalog.ApplyTenantPlacement(ctx, f.request); err == nil {
		t.Fatal("import adopted or replaced an existing schema")
	}
	var content string
	if err := f.source.db.QueryRowContext(ctx, `SELECT content FROM `+placementQualified(existing, "sentinel")).Scan(&content); err != nil || content != "must survive" {
		t.Fatal("preexisting target data changed", err)
	}
	var firstExists bool
	if err := f.source.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, tenantDataSchema(f.tenants[0])).Scan(&firstExists); err != nil || firstExists {
		t.Fatal("collision during the second Tenant left the first schema committed", err)
	}
	placements, err := f.catalog.ListTenants(ctx, "", 100)
	if err != nil || len(placements) != 0 {
		t.Fatal("collision committed partial catalog state", err)
	}
	if _, err := f.source.db.ExecContext(ctx, `UPDATE tenants SET name=name`); err != nil {
		t.Fatal("collision installed a partial source fence", err)
	}
}

func assertImportRolledBack(t *testing.T, f importFixture) {
	t.Helper()
	ctx := context.Background()
	placements, err := f.catalog.ListTenants(ctx, "", 100)
	if err != nil || len(placements) != 0 {
		t.Fatal("failed import exposed catalog entries", err)
	}
	for _, id := range f.tenants {
		var exists bool
		if err := f.source.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, tenantDataSchema(id)).Scan(&exists); err != nil || exists {
			t.Fatal("failed import left a target schema", err)
		}
	}
	if _, err := f.source.db.ExecContext(ctx, `UPDATE tenants SET name=name`); err != nil {
		t.Fatal("failed import installed a source fence", err)
	}
}

func TestTenantPlacementImportRejectsUnknownSchemaAndRollsBackClosureFailure(t *testing.T) {
	for _, scenario := range []string{"unknown_table", "extra_fk", "cross_tenant_delivery", "row_security", "disabled_row_policy"} {
		t.Run(scenario, func(t *testing.T) {
			f := newImportFixture(t)
			var err error
			switch scenario {
			case "unknown_table":
				_, err = f.source.db.Exec(`CREATE TABLE extra_data(tenant_id TEXT NOT NULL,body TEXT NOT NULL)`)
			case "extra_fk":
				_, err = f.source.db.Exec(`ALTER TABLE tenant_grants ADD CONSTRAINT unsupported_extra_fk FOREIGN KEY(principal_id) REFERENCES principals(id)`)
			case "cross_tenant_delivery":
				// The first Tenant copies successfully; failure while copying the
				// second must roll back its predecessor's schema and catalog row.
				_, err = f.source.db.Exec(`UPDATE trigger_firings SET source_event_id=(SELECT id FROM integration_events WHERE tenant_id=$1) WHERE trigger_id IN(SELECT id FROM triggers WHERE tenant_id=$2)`, f.tenants[0].String(), f.tenants[1].String())
			case "row_security":
				_, err = f.source.db.Exec(`ALTER TABLE tenants ENABLE ROW LEVEL SECURITY; ALTER TABLE tenants FORCE ROW LEVEL SECURITY; CREATE POLICY hidden_tenant ON tenants USING(id='` + f.tenants[0].String() + `')`)
			case "disabled_row_policy":
				_, err = f.source.db.Exec(`CREATE POLICY hidden_tenant ON tenants USING(id='` + f.tenants[0].String() + `')`)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.catalog.ApplyTenantPlacement(context.Background(), f.request); err == nil {
				t.Fatalf("unsafe import succeeded: %s", scenario)
			}
			assertImportRolledBack(t, f)
		})
	}
}

func TestTenantPlacementImportRequiresStoppedWritersAndExistingSource(t *testing.T) {
	f := newImportFixture(t)
	ctx := context.Background()
	for _, source := range []string{"not_present", f.catalog.schema, "public;drop schema public", "pg_catalog"} {
		if err := f.catalog.CheckTenantPlacementSource(ctx, source); err == nil {
			t.Fatalf("invalid source accepted: %s", source)
		}
	}
	request := f.request
	request.WritersStopped = false
	if _, err := f.catalog.ApplyTenantPlacement(ctx, request); err == nil {
		t.Fatal("import did not require explicit writer drain")
	}
	writer, err := f.source.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Rollback()
	if _, err := writer.ExecContext(ctx, `UPDATE tenants SET name=name WHERE id=$1`, f.tenants[0].String()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.catalog.ApplyTenantPlacement(ctx, f.request); err == nil {
		t.Fatal("import overlapped an active source writer")
	}
	if err := writer.Rollback(); err != nil {
		t.Fatal(err)
	}
	assertImportRolledBack(t, f)
}

func TestStoredJSONCustomWorkItemCodecRetainsLegacyScope(t *testing.T) {
	id := importTestID(t)
	outcome := importTestID(t)
	work, err := domain.NewWorkItem(importTestID(t), domain.Scope{TenantID: id, OutcomeID: outcome}, "Historic work", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(work)
	legacy := bytes.ReplaceAll(raw, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
	var copied domain.WorkItem
	if err := signing.DecodeStoredJSON(legacy, &copied, 0); err != nil || copied.Scope != work.Scope {
		t.Fatalf("custom WorkItem codec lost old scope: %+v %v", copied.Scope, err)
	}
}

func TestTenantPlacementImportAfterExplicitVersion32Migration(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	id := importTestID(t)
	sourceName := "wos_legacy_" + strings.ReplaceAll(id.String(), "-", "")
	catalogName := "wos_import_" + strings.ReplaceAll(id.String(), "-", "")
	source, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: sourceName})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	catalog, err := OpenTenantCatalog(ctx, os.Getenv("WOS_TEST_POSTGRES_DSN"), CatalogOptions{Schema: catalogName, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	defer func() {
		for _, schema := range []string{tenantDataSchema(id), sourceName, catalogName} {
			_, _ = source.db.ExecContext(ctx, `DROP SCHEMA IF EXISTS `+placementQuote(schema)+` CASCADE`)
		}
	}()
	tx, err := source.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE SCHEMA `+placementQuote(sourceName)+`; CREATE TABLE schema_migrations(version BIGINT PRIMARY KEY,name TEXT NOT NULL,checksum TEXT NOT NULL,applied_at BIGINT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.version > 32 {
			break
		}
		if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES($1,$2,$3,1)`, migration.version, migration.name, migration.checksum); err != nil {
			t.Fatal(err)
		}
	}
	protocol := domain.DefaultWorkProtocol(id)
	protocol.Phase, protocol.WriterEpoch, protocol.WritersDrained = "signed_contracts_v2", 2, true
	protocolRaw, _ := json.Marshal(protocol)
	protocolRaw = bytes.ReplaceAll(protocolRaw, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
	if _, err := tx.ExecContext(ctx, `INSERT INTO namespaces(id,name,lifecycle,version,created_at,updated_at) VALUES($1,'Existing installation','active',1,1,1); INSERT INTO namespace_work_protocol(namespace_id,version,phase,writer_epoch,state_json) VALUES($1,1,'signed_contracts_v2',2,$2)`, id.String(), string(protocolRaw)); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	request := TenantPlacementImport{SourceSchema: sourceName, Reason: "Existing signed installation, all authority settled", WritersStopped: true}
	if err := catalog.CheckTenantPlacementSource(ctx, sourceName); err != nil {
		t.Fatal("old source must be recognized before migration", err)
	}
	if _, err := catalog.ApplyTenantPlacement(ctx, request); err == nil {
		t.Fatal("import silently migrated an old source")
	}
	if err := source.Migrate(ctx); err != nil {
		t.Fatal("explicit source migration failed", err)
	}
	result, err := catalog.ApplyTenantPlacement(ctx, request)
	if err != nil || len(result.Placements) != 1 || !result.SourceWriteFenced {
		t.Fatal("current migrated source could not be partitioned", err)
	}
	var phase string
	var epoch int
	if err := source.db.QueryRowContext(ctx, `SELECT phase,writer_epoch FROM `+placementQualified(tenantDataSchema(id), "tenant_work_protocol")).Scan(&phase, &epoch); err != nil || phase != "signed_contracts_v3" || epoch != 3 {
		t.Fatal("cutover metadata was not carried into the isolated schema", err)
	}
}
