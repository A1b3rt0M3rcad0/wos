package mobility

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	wossqlite "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

const (
	nsID         = "0199a221-0000-7000-8000-000000000001"
	otherNS      = "0199a221-0000-7000-8000-000000000002"
	outcomeID    = "0199a221-0000-7000-8000-000000000003"
	workID       = "0199a221-0000-7000-8000-000000000004"
	credentialID = "0199a221-0000-7000-8000-000000000005"
	contractID   = "0199a221-0000-7000-8000-000000000006"
	issuerID     = "0199a221-0000-7000-8000-000000000007"
	serverID     = "0199a221-0000-7000-8000-000000000008"
	activeToken  = "migration-test-active-token-at-least-32"
)

type fixture struct {
	path     string
	identity d.ServerIdentity
	proof    string
	public   ed25519.PublicKey
}
type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now().UTC() }

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func seedFixture(t *testing.T) fixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "installation.db")
	store, err := wossqlite.Open(path, wossqlite.Options{MigrateOnOpen: true})
	must(t, err)
	must(t, store.Close())
	db, err := sql.Open("sqlite3", "file:"+path+"?_pragma=foreign_keys(1)")
	must(t, err)
	defer db.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)
	public, secret, err := ed25519.GenerateKey(rand.Reader)
	must(t, err)
	fp, err := signing.Fingerprint(public)
	must(t, err)
	id := d.ServerIdentity{ID: d.ID(serverID), IssuerKeyID: d.ID(issuerID), PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fp, CreatedAt: now}
	must(t, id.Validate())
	envelope, err := signing.Sign(signing.ContractAuthority, map[string]any{"protocol_version": 2, "signer_key_id": issuerID, "server_id": serverID, "contract_id": contractID, "note": "exact bytes < & Ω\n"}, issuerID, secret)
	must(t, err)
	rawProof, err := json.Marshal(envelope)
	must(t, err)
	rawID, err := json.Marshal(id)
	must(t, err)
	actor := `{"kind":"agent","provider":"test","id":"operator"}`
	exec := func(query string, args ...any) { t.Helper(); _, err := db.Exec(query, args...); must(t, err) }
	exec("INSERT INTO namespaces VALUES(?,?,'active',1,?,?)", nsID, "primary", now.UnixMicro(), now.UnixMicro())
	exec("INSERT INTO namespaces VALUES(?,?,'active',1,?,?)", otherNS, "secondary", now.UnixMicro(), now.UnixMicro())
	exec("INSERT INTO delivery_namespace_schedule VALUES(?,3)", nsID)
	exec("INSERT INTO delivery_namespace_schedule VALUES(?,4)", otherNS)
	exec("INSERT INTO principals VALUES('operator','operator','service','active',?)", now.UnixMicro())
	exec("INSERT INTO credentials(id,namespace_id,principal_id,actor_json,digest,expires_at,revoked) VALUES(?,?,'operator',?,?,?,0)", credentialID, nsID, actor, application.TokenDigest(activeToken), now.Add(time.Hour).UnixMicro())
	exec("INSERT INTO namespace_grants VALUES(?,'operator','[\"state:read\"]')", nsID)
	exec("INSERT INTO namespace_grants VALUES(?,'operator','[\"state:read\"]')", otherNS)
	exec("INSERT INTO outcome_coordination VALUES(?,?,7)", nsID, outcomeID)
	exec("INSERT INTO entity_refs VALUES(?,?,?,'outcome')", outcomeID, nsID, outcomeID)
	exec("INSERT INTO outcomes(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,desired_state,lifecycle) VALUES(?,?,?,1,?,?,'A < B Ω','preserve text','normal','reviewed','active')", outcomeID, nsID, outcomeID, now.UnixMicro(), now.UnixMicro())
	exec("INSERT INTO entity_refs VALUES(?,?,?,'work_item')", workID, nsID, outcomeID)
	exec("INSERT INTO work_items(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,lifecycle) VALUES(?,?,?,1,?,?,'Task','description','normal','todo')", workID, nsID, outcomeID, now.UnixMicro(), now.UnixMicro())
	exec("INSERT INTO server_protocol_identity VALUES(1,?)", string(rawID))
	exec("INSERT INTO server_issuer_public_keys VALUES(?,?,?,?)", issuerID, serverID, fp, string(rawID))
	exec("INSERT INTO signed_work_contracts(id,namespace_id,outcome_id,work_item_id,credential_id,holder_principal_id,status,version,lease_version,fencing_token,expires_at,state_json,spec_json,spec_digest) VALUES(?,?,?,?,?,'operator','active','18446744073709551615','2','18446744073709551615',?,?,?,?)", contractID, nsID, outcomeID, workID, credentialID, now.Add(time.Hour).UnixMicro(), string(rawProof), "{\"exact\": true}", "spec-hash")
	exec("INSERT INTO work_activities VALUES('activity',?,?,?,?,?,?)", nsID, outcomeID, contractID, workID, "00000000000000000002", string(rawProof))
	exec("INSERT INTO signed_operation_results VALUES(?,?,'operator',?,'original-intent-0001','AcquireSignedWorkContract',?,'0199a221-0000-7000-8000-000000000009','7',?,?)", nsID, outcomeID, credentialID, strings.Repeat("a", 64), string(rawProof), now.UnixMicro())
	exec("INSERT INTO webhook_endpoints VALUES('endpoint',?,'https://example.invalid/webhook','operator-secret-ref','key')", nsID)
	exec("INSERT INTO triggers VALUES('trigger',?,?,1,1,'{}')", nsID, outcomeID)
	exec("INSERT INTO integration_events VALUES('integration',?,?,'{}')", nsID, outcomeID)
	exec("INSERT INTO trigger_firings VALUES('firing','trigger',1,'integration','{}')")
	exec("INSERT INTO integration_deliveries(id,namespace_id,outcome_id,integration_event_id,endpoint_id,body_json,url,secret_ref,key_id,status,next_attempt) VALUES('delivery',?,?,'firing','endpoint',?,'https://example.invalid/webhook','operator-secret-ref','key','pending',?)", nsID, outcomeID, string(rawProof), now.UnixMicro())
	exec("INSERT INTO delivery_attempts VALUES('delivery',1,'old-lease',?,?,'timed_out')", now.Add(-time.Minute).UnixMicro(), now.UnixMicro())
	return fixture{path: path, identity: id, proof: string(rawProof), public: public}
}
func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	must(t, err)
	for {
		if _, err = os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("repository root unavailable")
		}
		root = parent
	}
}
func backupFixture(t *testing.T, f fixture) migrateOptions {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "backup")
	cmd := exec.Command("python3", filepath.Join(repositoryRoot(t), "tools", "operations", "backup_restore.py"), "backup", "--backend", "sqlite", "--sqlite", f.path, "--output", directory, "--source-commit", strings.Repeat("a", 40), "--product-version", "mobility-test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("backup: %v %s", err, out)
	}
	var result struct {
		SHA string `json:"manifest_sha256"`
	}
	must(t, json.Unmarshal(out, &result))
	return migrateOptions{Backup: directory, ManifestSHA256: result.SHA, WritersStopped: true, ExpectedServerID: f.identity.ID.String(), IssuerFingerprint: f.identity.Fingerprint}
}
func freshPostgres(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL required: set WOS_TEST_POSTGRES_DSN")
	}
	admin, err := openPostgres(dsn)
	must(t, err)
	u, err := url.Parse(dsn)
	must(t, err)
	name := fmt.Sprintf("wos_mobility_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE " + quoted(name))
	must(t, err)
	t.Cleanup(func() {
		_, e := admin.Exec("DROP DATABASE " + quoted(name) + " WITH (FORCE)")
		if e != nil {
			t.Error(e)
		}
		admin.Close()
	})
	u.Path = "/" + name
	return u.String()
}
func TestSourceAndNamespacePreflight(t *testing.T) {
	f := seedFixture(t)
	db, err := openSQLite(f.path)
	must(t, err)
	defer db.Close()
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	must(t, err)
	defer tx.Rollback()
	tables, err := validateSource(context.Background(), tx)
	must(t, err)
	if len(tables) < 40 {
		t.Fatal("expected complete schema inventory")
	}
	result, err := preflight(context.Background(), tx, "sqlite", nsID)
	must(t, err)
	if result["ready_for_transfer"] != false || result["read_only"] != true || result["issuer_seed_collected"] != false {
		t.Fatal(result)
	}
	counts := result["scoped_table_rows"].(map[string]int64)
	if counts["signed_operation_results"] != 1 || counts["integration_deliveries"] != 1 || counts["namespace_grants"] != 1 {
		t.Fatal(counts)
	}
	indirect := fmt.Sprint(result["tables_requiring_global_or_indirect_reference_closure"])
	for _, name := range []string{"principals", "server_protocol_identity", "delivery_attempts", "trigger_firings"} {
		if !strings.Contains(indirect, name) {
			t.Fatal(indirect)
		}
	}
	inspection := result["inspection"].(map[string]any)
	if inspection["principal_inventory"].(map[string]int64)["principals_also_referenced_by_other_namespaces"] != 1 || inspection["unexpired_authority"].(map[string]int64)["signed_work_contracts_unexpired"] != 1 || inspection["delivery_inventory"].(map[string]int64)["pending"] != 1 || inspection["delivery_inventory"].(map[string]int64)["indirect_attempts"] != 1 {
		t.Fatal(inspection)
	}
	if inspection["missing_scoped_references"] != int64(0) || inspection["cross_namespace_references"] != int64(0) {
		t.Fatal(inspection)
	}
	_, err = preflight(context.Background(), tx, "sqlite", "missing")
	if err == nil {
		t.Fatal("missing Namespace accepted")
	}
}
func TestPreflightFindsBrokenCredentialAndReferenceClosure(t *testing.T) {
	f := seedFixture(t)
	write, err := sql.Open("sqlite3", "file:"+f.path)
	must(t, err)
	write.SetMaxOpenConns(1)
	_, err = write.Exec("PRAGMA foreign_keys=OFF")
	must(t, err)
	_, err = write.Exec("UPDATE credentials SET principal_id='missing',parent_digest='unknown-parent'")
	must(t, err)
	must(t, write.Close())
	db, err := openSQLite(f.path)
	must(t, err)
	defer db.Close()
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	must(t, err)
	defer tx.Rollback()
	result, err := preflight(context.Background(), tx, "sqlite", nsID)
	must(t, err)
	inspection := result["inspection"].(map[string]any)
	if inspection["missing_scoped_references"].(int64) < 1 || inspection["credential_inventory"].(map[string]int64)["missing_or_cross_namespace_session_parents"] != 1 {
		t.Fatal(inspection)
	}
	if inspection["invalid_credential_reference_counts"].(map[string]int64)["signed_work_contracts.credential_id"] != 1 {
		t.Fatal(inspection)
	}
	if _, err = validateSource(context.Background(), tx); err == nil {
		t.Fatal("corrupt reference closure accepted")
	}
}
func TestSourceMigrationChainCannotBeRelabeled(t *testing.T) {
	f := seedFixture(t)
	db, err := sql.Open("sqlite3", "file:"+f.path)
	must(t, err)
	_, err = db.Exec("UPDATE schema_migrations SET checksum='unreviewed' WHERE version=(SELECT MAX(version) FROM schema_migrations)")
	must(t, err)
	must(t, db.Close())
	opts := backupFixture(t, f)
	_, err = migrate(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "migration chain") {
		t.Fatalf("altered source migration chain accepted: %v", err)
	}
}
func TestApprovedBackupIsStagedWithoutUnapprovedWALOrPathReplacement(t *testing.T) {
	f := seedFixture(t)
	opts := backupFixture(t, f)
	path := filepath.Join(opts.Backup, "database.sqlite3")
	writer, err := sql.Open("sqlite3", "file:"+path)
	must(t, err)
	writer.SetMaxOpenConns(1)
	defer writer.Close()
	var mode string
	must(t, writer.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode))
	if mode != "wal" {
		t.Fatal("fixture needs WAL mode")
	}
	_, err = writer.Exec("PRAGMA wal_autocheckpoint=0")
	must(t, err)
	var busy, log, checkpointed int
	must(t, writer.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &log, &checkpointed))
	// Approve the checkpointed main file. Later WAL commits are deliberately
	// outside the independently approved database checksum.
	dataSHA, err := hashFile(path)
	must(t, err)
	manifestPath := filepath.Join(opts.Backup, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	must(t, err)
	var manifest map[string]any
	must(t, json.Unmarshal(raw, &manifest))
	manifest["data_sha256"] = dataSHA
	raw, err = json.Marshal(manifest)
	must(t, err)
	must(t, os.WriteFile(manifestPath, raw, 0o600))
	opts.ManifestSHA256, err = hashFile(manifestPath)
	must(t, err)
	_, err = writer.Exec("UPDATE outcomes SET title='unapproved-WAL-data'")
	must(t, err)
	after, err := hashFile(path)
	must(t, err)
	if after != dataSHA {
		t.Fatal("fixture unexpectedly checkpointed unapproved WAL")
	}
	var observed string
	must(t, writer.QueryRow("SELECT title FROM outcomes").Scan(&observed))
	if observed != "unapproved-WAL-data" {
		t.Fatal("fixture did not commit adjacent WAL")
	}
	staged, _, cleanup, err := readBackup(context.Background(), opts.Backup, opts.ManifestSHA256)
	must(t, err)
	defer cleanup()
	info, err := os.Stat(filepath.Dir(staged))
	must(t, err)
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatal("staging directory is not private")
	}
	info, err = os.Stat(staged)
	must(t, err)
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatal("staged database is not private")
	}
	// Closing can checkpoint the unapproved commit, and replacing the source
	// path afterwards must still have no influence on the approved snapshot.
	must(t, writer.Close())
	must(t, os.WriteFile(path, []byte("replacement after approval"), 0o600))
	approved, err := openSQLiteMode(staged, true)
	must(t, err)
	must(t, approved.QueryRow("SELECT title FROM outcomes").Scan(&observed))
	must(t, approved.Close())
	if observed != "A < B Ω" {
		t.Fatal("unapproved adjacent WAL or replaced path was imported", observed)
	}
	cleanup()
	if _, err = os.Stat(staged); !os.IsNotExist(err) {
		t.Fatal("staged approved data was not removed")
	}
}

func TestUnsupportedHostChecksFollowHelpAndPrecedeStorageAccess(t *testing.T) {
	unsupported := refuse("storage operator commands are supported only on Linux")
	for _, args := range [][]string{{"migrate-sqlite", "--backup", "absent"}, {"migrate-sqlite", "--backup", "--help", "--writers-stopped"}, {"namespace-preflight", "--sqlite", "absent", "--namespace", nsID}} {
		var output bytes.Buffer
		err := runCommand(context.Background(), args, &output, func() error { return unsupported })
		if err != unsupported || output.Len() != 0 {
			t.Fatal("unsupported host reached storage access or accepted help flag value", args, err, output.String())
		}
	}
	for _, args := range [][]string{{"--help"}, {"migrate-sqlite", "--help"}, {"namespace-preflight", "--help"}} {
		var output bytes.Buffer
		err := runCommand(context.Background(), args, &output, func() error { t.Fatal("local help checked storage host"); return unsupported })
		if err != nil || output.Len() == 0 {
			t.Fatal("help unavailable on unsupported host", args, err)
		}
	}
}
func TestApprovedBackupAndCutoverRequiredBeforeDestination(t *testing.T) {
	f := seedFixture(t)
	opts := backupFixture(t, f)
	for _, change := range []func(*migrateOptions){func(o *migrateOptions) { o.WritersStopped = false }, func(o *migrateOptions) { o.ManifestSHA256 = strings.Repeat("f", 64) }, func(o *migrateOptions) { o.ExpectedServerID = "other" }, func(o *migrateOptions) { o.IssuerFingerprint = "other" }} {
		probe := opts
		change(&probe)
		if _, err := migrate(context.Background(), probe); err == nil {
			t.Fatal("unsafe migration accepted")
		}
	}
	path := filepath.Join(opts.Backup, "database.sqlite3")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	must(t, err)
	_, err = file.Write([]byte("tamper"))
	must(t, err)
	must(t, file.Close())
	if _, err := migrate(context.Background(), opts); err == nil {
		t.Fatal("tampered backup accepted")
	}
}
func TestSQLiteToPostgresPreservesAuthorityReceiptsOutboxAndProof(t *testing.T) {
	f := seedFixture(t)
	opts := backupFixture(t, f)
	opts.DSN = freshPostgres(t)
	before, err := hashFile(f.path)
	must(t, err)
	result, err := migrate(context.Background(), opts)
	must(t, err)
	if result["verified"] != true || result["scope"] != "whole_installation" || result["issuer_seed_collected"] != false {
		t.Fatal(result)
	}
	after, err := hashFile(f.path)
	must(t, err)
	if before != after {
		t.Fatal("source changed")
	}
	target, err := openPostgres(opts.DSN)
	must(t, err)
	defer target.Close()
	var proof, version, sequence string
	must(t, target.QueryRow("SELECT state_json,version::text FROM signed_work_contracts WHERE id=$1", contractID).Scan(&proof, &version))
	if proof != f.proof || version != "18446744073709551615" {
		t.Fatal("proof or uint64 changed")
	}
	var envelope signing.Envelope
	must(t, json.Unmarshal([]byte(proof), &envelope))
	_, err = signing.Verify(envelope, signing.ContractAuthority, issuerID, f.public)
	must(t, err)
	must(t, target.QueryRow("SELECT sequence::text FROM work_activities").Scan(&sequence))
	if sequence != "2" {
		t.Fatal(sequence)
	}
	var pending, attempts, receipts, noncanonical int
	must(t, target.QueryRow("SELECT COUNT(*) FROM integration_deliveries WHERE status='pending'").Scan(&pending))
	must(t, target.QueryRow("SELECT COUNT(*) FROM delivery_attempts").Scan(&attempts))
	must(t, target.QueryRow("SELECT COUNT(*) FROM signed_operation_results WHERE response_json=$1", f.proof).Scan(&receipts))
	must(t, target.QueryRow("SELECT COUNT(*) FROM pg_constraint WHERE connamespace='public'::regnamespace AND contype='f' AND condeferrable").Scan(&noncanonical))
	if pending != 1 || attempts != 1 || receipts != 1 || noncanonical != 0 {
		t.Fatal("closure, receipts or original constraints changed", pending, attempts, receipts, noncanonical)
	}
	var turn int64
	must(t, target.QueryRow("SELECT last_turn FROM delivery_namespace_schedule WHERE namespace_id=$1", nsID).Scan(&turn))
	if turn != 3 {
		t.Fatal("delivery fairness schedule changed")
	}
	preflightTx, err := target.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	must(t, err)
	checked, err := preflight(context.Background(), preflightTx, "postgres", nsID)
	must(t, err)
	must(t, preflightTx.Rollback())
	if checked["inspection"].(map[string]any)["principal_inventory"].(map[string]int64)["principals_also_referenced_by_other_namespaces"] != 1 {
		t.Fatal(checked)
	}
	store, err := postgres.Open(opts.DSN, postgres.Options{MigrateOnOpen: true})
	must(t, err)
	defer store.Close()
	credential, err := store.GetCredential(context.Background(), application.TokenDigest(activeToken))
	must(t, err)
	if credential.PrincipalID != "operator" || credential.Revoked {
		t.Fatal("credential did not survive")
	}
	security := application.SecurityService{Store: store, Clock: wallClock{}}
	identity, err := security.Authenticate(context.Background(), activeToken)
	must(t, err)
	must(t, security.Require(application.WithIdentity(context.Background(), identity), d.ID(nsID), ports.PermissionStateRead))
	unit, err := store.Begin(context.Background())
	must(t, err)
	defer unit.Rollback()
	storedIdentity, err := unit.(ports.ServerIdentityUnitOfWork).ServerIdentity().Server(context.Background())
	must(t, err)
	if storedIdentity != f.identity {
		t.Fatal("installation trust changed")
	}
	receipt, err := unit.(ports.SignedOperationUnitOfWork).SignedOperations().Get(context.Background(), d.ID(nsID), "operator", "original-intent-0001")
	must(t, err)
	if string(receipt.Result.ResponseJSON) != f.proof || receipt.Result.OutcomeRevision != 7 {
		t.Fatal("exact-intent receipt changed")
	}
	if _, err = migrate(context.Background(), opts); err == nil {
		t.Fatal("existing destination overwritten")
	}
}
func TestMigrationFailureLeavesNoPartialDomainState(t *testing.T) {
	f := seedFixture(t)
	// This is legal in SQLite's legacy unsigned decimal field but is not a
	// canonical uint64. Refuse it rather than silently normalize authority.
	db, err := sql.Open("sqlite3", "file:"+f.path)
	must(t, err)
	_, err = db.Exec("UPDATE signed_work_contracts SET lease_version='02'")
	must(t, err)
	must(t, db.Close())
	opts := backupFixture(t, f)
	opts.DSN = freshPostgres(t)
	if _, err = migrate(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "noncanonical") {
		t.Fatalf("expected canonicality refusal: %v", err)
	}
	target, err := openPostgres(opts.DSN)
	must(t, err)
	defer target.Close()
	var count, changed int
	must(t, target.QueryRow("SELECT COUNT(*) FROM namespaces").Scan(&count))
	must(t, target.QueryRow("SELECT COUNT(*) FROM pg_constraint WHERE connamespace='public'::regnamespace AND contype='f' AND condeferrable").Scan(&changed))
	if count != 0 || changed != 0 {
		t.Fatal("partial domain state or relaxed constraints committed")
	}
}
func TestCommandPreflightIsMachineReadableAndNeverReportsReady(t *testing.T) {
	f := seedFixture(t)
	var out bytes.Buffer
	must(t, run(context.Background(), []string{"namespace-preflight", "--sqlite", f.path, "--namespace", nsID}, &out))
	var result map[string]any
	must(t, json.Unmarshal(out.Bytes(), &result))
	if result["ready_for_transfer"] != false {
		t.Fatal(result)
	}
	if strings.Contains(out.String(), activeToken) || strings.Contains(out.String(), f.proof) {
		t.Fatal("preflight disclosed private state")
	}
}
