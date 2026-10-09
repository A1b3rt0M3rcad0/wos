package server

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"os"
	"regexp"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

func DecodeIssuerRecoveryIntent(reader io.Reader) (a.IssuerRecoveryIntent, error) {
	var intent a.IssuerRecoveryIntent
	raw, e := io.ReadAll(io.LimitReader(reader, (32<<10)+1))
	if e != nil {
		return intent, e
	}
	if e = signing.DecodeStrict(raw, &intent, 32<<10); e != nil {
		return intent, e
	}
	return intent, nil
}

// RecoverIssuer opens storage without serving, bootstrap or silent signer
// provisioning. The host explicitly supplies the public predecessor and new
// pin; the protected seed environment reference supplies possession only.
func RecoverIssuer(ctx context.Context, config Config, intent a.IssuerRecoveryIntent) (d.IssuerRecovery, bool, error) {
	var zero d.IssuerRecovery
	if e := config.Validate(); e != nil {
		return zero, false, e
	}
	if config.Auth.Mode != AuthModeAPIToken || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(config.Signing.SeedEnv) {
		return zero, false, d.NewError(d.ErrorCodeInvalidConfig, "host recovery requires configured API mode and protected seed reference")
	}
	encoded := os.Getenv(config.Signing.SeedEnv)
	seed, e := base64.StdEncoding.Strict().DecodeString(encoded)
	if e != nil || len(seed) != ed25519.SeedSize || base64.StdEncoding.EncodeToString(seed) != encoded {
		clear(seed)
		return zero, false, d.NewError(d.ErrorCodeInvalidConfig, "new issuer seed is missing or not canonical base64 Ed25519 seed")
	}
	private := ed25519.NewKeyFromSeed(seed)
	clear(seed)
	defer clear(private)
	public := private.Public().(ed25519.PublicKey)
	fp, e := signing.Fingerprint(public)
	if e != nil {
		return zero, false, e
	}
	if fp != intent.NewFingerprint {
		return zero, false, d.NewError(d.ErrorCodeInvalidConfig, "replacement seed differs from explicitly approved fingerprint")
	}
	var store runtimeStore
	if config.Storage.Driver == StorageDriverPostgres {
		store, e = postgres.Open(config.Storage.PostgresDSN, postgres.Options{MigrateOnOpen: config.Storage.MigrateOnStart})
	} else {
		store, e = sqlite.Open(config.Storage.SQLitePath, sqlite.Options{MigrateOnOpen: config.Storage.MigrateOnStart})
	}
	if e != nil {
		return zero, false, e
	}
	defer store.Close()
	version, schemaErr := store.SchemaVersion(ctx)
	supported, latestErr := sqlite.LatestSchemaVersion()
	if config.Storage.Driver == StorageDriverPostgres {
		supported, latestErr = postgres.LatestSchemaVersion()
	}
	if schemaErr != nil || latestErr != nil || version != supported {
		return zero, false, d.NewError(d.ErrorCodeInvalidConfig, "database schema is incompatible with host issuer recovery")
	}
	u, e := store.Begin(ctx)
	if e != nil {
		return zero, false, e
	}
	repo, ok := u.(ports.ServerIdentityUnitOfWork)
	if !ok {
		u.Rollback()
		return zero, false, d.NewError(d.ErrorCodeInvalidConfig, "persistent issuer unavailable")
	}
	pinned, e := repo.ServerIdentity().Server(ctx)
	u.Rollback()
	if e != nil {
		return zero, false, e
	}
	replacement := d.ServerIdentity{ID: intent.ServerID, IssuerKeyID: intent.NewIssuerID, PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fp, CreatedAt: pinned.CreatedAt}
	return a.RecoverServerIssuer(ctx, store, systemClock{}, intent, &localIssuer{identity: replacement, private: private})
}
