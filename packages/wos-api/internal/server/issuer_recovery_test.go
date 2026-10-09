package server

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func TestExplicitHostIssuerRecoveryRestartsAndExposesBoundedPublicTrust(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		ctx := context.Background()
		must := func(e error) {
			t.Helper()
			if e != nil {
				t.Fatal(e)
			}
		}
		oldSeed := make([]byte, ed25519.SeedSize)
		oldSeed[0] = 41
		newSeed := make([]byte, ed25519.SeedSize)
		newSeed[0] = 42
		t.Setenv("WOS_TEST_RECOVERY_SEED", base64.StdEncoding.EncodeToString(oldSeed))
		cfg.Signing.SeedEnv = "WOS_TEST_RECOVERY_SEED"
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("r", 40)
		cfg.Auth.BootstrapNamespaceID = "0199a777-0000-7000-8000-000000000001"
		cfg.Auth.BootstrapNamespaceName = "host recovery"
		initial, e := OpenRuntime(cfg)
		must(e)
		u, e := initial.store.Begin(ctx)
		must(e)
		old, e := u.(ports.ServerIdentityUnitOfWork).ServerIdentity().Server(ctx)
		must(e)
		must(u.Rollback())
		must(initial.Close())
		newPublic := ed25519.NewKeyFromSeed(newSeed).Public().(ed25519.PublicKey)
		fp, e := signing.Fingerprint(newPublic)
		must(e)
		newID, e := (uuidV7Generator{}).NewID()
		must(e)
		intent := a.IssuerRecoveryIntent{ServerID: old.ID, PreviousIssuerID: old.IssuerKeyID, PreviousFingerprint: old.Fingerprint, NewIssuerID: newID, NewFingerprint: fp, Reason: "explicit operator recovery with approved replacement public fingerprint"}
		t.Setenv("WOS_TEST_RECOVERY_SEED", base64.StdEncoding.EncodeToString(newSeed))
		if silently, e := OpenRuntime(cfg); e == nil {
			silently.Close()
			t.Fatal("runtime accepted replacement without host intention")
		}
		wrong := intent
		wrong.NewFingerprint = old.Fingerprint
		if _, _, e = RecoverIssuer(ctx, cfg, wrong); e == nil {
			t.Fatal("host recovery ignored declared replacement pin")
		}
		record, replay, e := RecoverIssuer(ctx, cfg, intent)
		must(e)
		if replay || record.Replacement.ID != old.ID || record.Replacement.CreatedAt != old.CreatedAt {
			t.Fatal("host recovery replaced persistent instance identity")
		}
		clear(oldSeed)
		again, replay, e := RecoverIssuer(ctx, cfg, intent)
		must(e)
		if !replay || again != record {
			t.Fatal("host command lost frozen recovery receipt")
		}
		raw, e := json.Marshal(record)
		must(e)
		if strings.Contains(string(raw), base64.StdEncoding.EncodeToString(newSeed)) {
			t.Fatal("recovery receipt exposed private seed")
		}
		current, e := OpenRuntime(cfg)
		must(e)
		defer current.Close()
		endpoint := httptest.NewServer(current.Handler())
		defer endpoint.Close()
		client, e := sdk.New(endpoint.URL, cfg.Auth.BootstrapToken, endpoint.Client())
		must(e)
		query := a.SignedStateQuery{Scope: d.Scope{NamespaceID: d.MustParseID(cfg.Auth.BootstrapNamespaceID), OutcomeID: d.MustParseID("0199a777-0000-7000-8000-000000000002")}, Resource: "trust", Limit: 1}
		first, e := client.ReadSignedState(ctx, query)
		must(e)
		if first.Server == nil || first.Server.ID != old.ID || first.Server.IssuerKeyID != newID || len(first.IssuerHistory) != 1 || first.NextCursor == "" || first.SearchComplete {
			t.Fatal("public trust pagination lost current issuer or history")
		}
		query.Cursor = first.NextCursor
		second, e := client.ReadSignedState(ctx, query)
		must(e)
		if len(second.IssuerHistory) != 1 || !second.SearchComplete || second.NextCursor != "" {
			t.Fatal("issuer history did not finish bounded page")
		}
	})
}

func TestHostIssuerRecoveryIntentStrictlyRejectsAmbiguityAndOversizedInput(t *testing.T) {
	for _, input := range []string{`{"server_id":"first","server_id":"replacement"}`, `{"ServerID":"alternate spelling"}`, `{"private_seed":"must never be in the public recovery manifest"}`, strings.Repeat(" ", 33<<10)} {
		if _, e := DecodeIssuerRecoveryIntent(strings.NewReader(input)); e == nil {
			t.Fatal("invalid recovery input accepted")
		}
	}
}

func TestHostIssuerRecoveryRefusesIncompatibleSchemaWithoutMigrating(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		ctx := context.Background()
		seed := make([]byte, ed25519.SeedSize)
		seed[0] = 63
		t.Setenv("WOS_TEST_RECOVERY_SCHEMA_SEED", base64.StdEncoding.EncodeToString(seed))
		cfg.Signing.SeedEnv = "WOS_TEST_RECOVERY_SCHEMA_SEED"
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("s", 40)
		cfg.Auth.BootstrapNamespaceID = "0199a778-0000-7000-8000-000000000001"
		cfg.Auth.BootstrapNamespaceName = "host schema recovery"
		runtime, e := OpenRuntime(cfg)
		if e != nil {
			t.Fatal(e)
		}
		latest, e := runtime.store.SchemaVersion(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if e = runtime.Close(); e != nil {
			t.Fatal(e)
		}
		driver, dsn := "sqlite3", cfg.Storage.SQLitePath
		if cfg.Storage.Driver == StorageDriverPostgres {
			driver, dsn = "pgx", cfg.Storage.PostgresDSN
		}
		db, e := sql.Open(driver, dsn)
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		_, e = db.ExecContext(ctx, "INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES ($1, 'future.sql', 'future-recovery-sentinel', 1)", latest+1)
		if e != nil {
			t.Fatal(e)
		}
		public := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
		fingerprint, e := signing.Fingerprint(public)
		if e != nil {
			t.Fatal(e)
		}
		cfg.Storage.MigrateOnStart = false
		_, _, e = RecoverIssuer(ctx, cfg, a.IssuerRecoveryIntent{NewFingerprint: fingerprint})
		if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeInvalidConfig || !strings.Contains(e.Error(), "database schema") {
			t.Fatalf("host recovery did not reject incompatible schema: %v", e)
		}
		var checksum string
		if e = db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version=$1", latest+1).Scan(&checksum); e != nil || checksum != "future-recovery-sentinel" {
			t.Fatal("host recovery altered incompatible history", checksum, e)
		}
	})
}
