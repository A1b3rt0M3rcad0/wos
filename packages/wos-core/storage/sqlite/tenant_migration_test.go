package sqlite

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func tenantMigrationSource32(t *testing.T) *Store {
	t.Helper()
	return tenantMigrationSourceVersion(t, 32)
}

func tenantMigrationSourceVersion(t *testing.T, version int64) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "before-tenant.db"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	if _, err = s.db.ExecContext(ctx, `CREATE TABLE schema_migrations(version BIGINT PRIMARY KEY,name TEXT NOT NULL,checksum TEXT NOT NULL,applied_at BIGINT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if m.version <= version {
			if err = s.applyMigration(ctx, m); err != nil {
				t.Fatal(err)
			}
		}
	}
	return s
}

func TestTenantIdentityMigrationPreservesSignedBytesAndChangesOnlyCurrentConfiguration(t *testing.T) {
	s := tenantMigrationSource32(t)
	ctx := context.Background()
	tenant := domain.MustParseID("0199ff31-0000-7000-8000-000000000002")
	for _, q := range []string{
		`INSERT INTO namespaces(id,name,lifecycle,version,created_at,updated_at) VALUES('` + tenant.String() + `','Legacy customer','active',1,1,1)`,
		`INSERT INTO principals(id,display_name,kind,lifecycle,created_at) VALUES('legacy','Legacy operator','human','active',1)`,
		`INSERT INTO namespace_grants(namespace_id,principal_id,permissions_json) VALUES('` + tenant.String() + `','legacy','["state:read","namespace:admin","installation:provision"]')`,
	} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	protocol := domain.DefaultWorkProtocol(tenant)
	protocol.Phase, protocol.WriterEpoch, protocol.WritersDrained = "signed_contracts_v2", 2, true
	raw, err := json.Marshal(protocol)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.ReplaceAll(raw, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
	if _, err = s.db.ExecContext(ctx, `INSERT INTO namespace_work_protocol(namespace_id,version,phase,writer_epoch,state_json) VALUES(?,1,'signed_contracts_v2',2,?)`, tenant.String(), string(raw)); err != nil {
		t.Fatal(err)
	}
	credentialID := domain.MustParseID("0199ff31-0000-7000-8000-000000000003")
	if _, err = s.db.ExecContext(ctx, `INSERT INTO credentials(id,namespace_id,principal_id,actor_json,digest,expires_at,revoked) VALUES(?,?,'legacy','{"kind":"human","provider":"test","id":"legacy"}','legacy-digest',1,0)`, credentialID.String(), tenant.String()); err != nil {
		t.Fatal(err)
	}
	policy := domain.CredentialPolicy{TenantID: tenant, CredentialID: credentialID, PrincipalID: "legacy", Version: 1, PermittedOperations: []string{"state:read", "namespace:admin"}, AcceptanceFloor: domain.AcceptanceIndependentReview, MaxActiveWorkContracts: 1, MaxActiveReviewContracts: 1, UpdatedAt: time.Now().UTC()}
	raw, err = json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.ReplaceAll(raw, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
	if _, err = s.db.ExecContext(ctx, `INSERT INTO credential_policies(namespace_id,credential_id,principal_id,version,state_json) VALUES(?,?,'legacy',1,?)`, tenant.String(), credentialID.String(), string(raw)); err != nil {
		t.Fatal(err)
	}
	vector, err := os.ReadFile("../../signing/testdata/dsse-vector.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO security_admin_results(namespace_id,principal_id,idempotency_key,fingerprint,result_json) VALUES(?,'legacy','original-operation','original-fingerprint',?)`, tenant.String(), string(vector)); err != nil {
		t.Fatal(err)
	}
	var originalChecksum string
	if err = s.db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version=1`).Scan(&originalChecksum); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var saved, fingerprint, permissions, checksum string
	if err = s.db.QueryRowContext(ctx, `SELECT result_json,fingerprint FROM security_admin_results WHERE tenant_id=?`, tenant.String()).Scan(&saved, &fingerprint); err != nil {
		t.Fatal(err)
	}
	if saved != string(vector) || fingerprint != "original-fingerprint" {
		t.Fatal("migration changed a historical proof or exact intent fingerprint")
	}
	var v struct {
		Public string           `json:"public_key_hex"`
		Digest string           `json:"request_digest"`
		Proof  signing.Envelope `json:"envelope"`
	}
	if err = json.Unmarshal([]byte(saved), &v); err != nil {
		t.Fatal(err)
	}
	public, err := hex.DecodeString(v.Public)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := signing.VerifyHistorical(v.Proof, signing.WorkReturn, "vector-key", public)
	if err != nil || signing.Digest(verified) != v.Digest {
		t.Fatal("original independent protocol-v2 signature no longer verifies", err)
	}
	if err = s.db.QueryRowContext(ctx, `SELECT state_json FROM tenant_work_protocol WHERE tenant_id=?`, tenant.String()).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if err = unmarshalJSON(saved, &protocol); err != nil || protocol.TenantID != tenant || protocol.Phase != domain.WorkProtocolSigned || protocol.WriterEpoch != 3 || protocol.Version != 2 {
		t.Fatal("current protocol configuration did not transition", err, protocol)
	}
	if bytes.Contains([]byte(saved), []byte(`"namespace_id"`)) {
		t.Fatal("current protocol configuration retained obsolete scope spelling")
	}
	if err = s.db.QueryRowContext(ctx, `SELECT permissions_json FROM tenant_grants WHERE tenant_id=?`, tenant.String()).Scan(&permissions); err != nil {
		t.Fatal(err)
	}
	if permissions != `["state:read","tenant:admin","installation:provision"]` {
		t.Fatal("current administrator permission was not preserved", permissions)
	}
	var policyVersion uint64
	if err = s.db.QueryRowContext(ctx, `SELECT version,state_json FROM credential_policies WHERE tenant_id=? AND credential_id=?`, tenant.String(), credentialID.String()).Scan(&policyVersion, &saved); err != nil {
		t.Fatal(err)
	}
	if err = unmarshalJSON(saved, &policy); err != nil || policyVersion != 2 || policy.Version != 2 || len(policy.PermittedOperations) != 2 || policy.PermittedOperations[1] != "tenant:admin" {
		t.Fatal("credential policy lost administrator authority during cutover", err, policy)
	}
	if err = s.db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version=1`).Scan(&checksum); err != nil || checksum != originalChecksum {
		t.Fatal("historical migration ledger changed", err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal("migration is not restart-safe", err)
	}
}

func TestTenantIdentityCutoverRejectsUnsettledProtocolBeforeDDL(t *testing.T) {
	for _, version := range []int64{26, 32} {
		t.Run(strconv.FormatInt(version, 10), func(t *testing.T) {
			testTenantIdentityCutoverRejectsUnsettledProtocolBeforeDDL(t, version)
		})
	}
}

func testTenantIdentityCutoverRejectsUnsettledProtocolBeforeDDL(t *testing.T, sourceVersion int64) {
	s := tenantMigrationSourceVersion(t, sourceVersion)
	ctx := context.Background()
	// Build valid legacy Outcome/WorkItem references. The acquisition is pending
	// at the protocol layer; no live application is needed to test upgrade gating.
	for _, q := range []string{
		`INSERT INTO namespaces VALUES('t','Customer','active',1,1,1)`,
		`INSERT INTO principals VALUES('p','Participant','human','active',1)`,
		`INSERT INTO outcome_coordination VALUES('t','o',0)`,
		`INSERT INTO entity_refs(id,namespace_id,outcome_id,kind) VALUES('o','t','o','outcome')`,
		`INSERT INTO outcomes(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,desired_state,lifecycle) VALUES('o','t','o',1,1,1,'Outcome','','normal','','active')`,
		`INSERT INTO entity_refs(id,namespace_id,outcome_id,kind) VALUES('w','t','o','work_item')`,
		`INSERT INTO work_items(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,lifecycle) VALUES('w','t','o',1,1,1,'Task','','normal','todo')`,
		`INSERT INTO signed_work_contracts(id,namespace_id,outcome_id,work_item_id,credential_id,holder_principal_id,status,version,lease_version,fencing_token,expires_at,state_json,spec_json,spec_digest) VALUES('c','t','o','w','credential','p','active','1','1','1',1,'{}','{}','digest')`,
	} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Migrate(ctx); err == nil {
		t.Fatal("upgrade accepted unresolved legacy signed authority")
	} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodePreconditionFailed {
		t.Fatal("upgrade did not identify a required cutover precondition", err)
	}
	var version int64
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != sourceVersion {
		t.Fatal("failed upgrade changed the migration ledger", err, version)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM namespaces`).Scan(&count); err != nil || count != 1 {
		t.Fatal("failed upgrade renamed or removed source tables", err)
	}
	var state, spec, digest string
	if err := s.db.QueryRowContext(ctx, `SELECT state_json,spec_json,spec_digest FROM signed_work_contracts WHERE id='c'`).Scan(&state, &spec, &digest); err != nil || state != "{}" || spec != "{}" || digest != "digest" {
		t.Fatal("failed upgrade changed original contract bytes", err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE signed_work_contracts SET status='revoked'`); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal("settled legacy data could not migrate", err)
	}
}
