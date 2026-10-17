package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

// InstallationOperatorIntent is an offline, host-authorized upgrade intent. It
// never issues a secret, restores revoked credentials or starts the HTTP runtime.
type InstallationOperatorIntent struct {
	TenantID        d.ID      `json:"tenant_id"`
	PrincipalID     string    `json:"principal_id"`
	ExpectedVersion d.Version `json:"expected_tenant_version"`
	IdempotencyKey  string    `json:"idempotency_key"`
	Reason          string    `json:"reason"`
	DrainedWriters  bool      `json:"drained_writers"`
}

func DecodeInstallationOperatorIntent(reader io.Reader) (InstallationOperatorIntent, error) {
	var intent InstallationOperatorIntent
	raw, err := io.ReadAll(io.LimitReader(reader, (16<<10)+1))
	if err != nil {
		return intent, err
	}
	err = signing.DecodeStrict(raw, &intent, 16<<10)
	return intent, err
}

// AuthorizeInstallationOperator requires possession of the installation's storage
// configuration and the host's explicit assertion that old writers are drained.
// Existing grants are never promoted by a schema migration or normal restart.
func AuthorizeInstallationOperator(ctx context.Context, config Config, intent InstallationOperatorIntent) (ports.SecurityAdminResult, error) {
	var zero ports.SecurityAdminResult
	config = config.withTenantBindingDefaults()
	if err := config.Validate(); err != nil {
		return zero, err
	}
	if !intent.DrainedWriters || intent.TenantID.Validate() != nil || intent.ExpectedVersion < 1 || strings.TrimSpace(intent.PrincipalID) == "" || len(intent.PrincipalID) > 256 || len(strings.TrimSpace(intent.Reason)) < 8 || len(intent.Reason) > 1024 {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "explicit Tenant, existing administrator, expected version, reason and drained_writers=true are required")
	}
	if err := d.ValidateIdempotencyKey(intent.IdempotencyKey); err != nil {
		return zero, err
	}
	var store runtimeStore
	var err error
	var supported int64
	var schemaErr error
	if config.Storage.Driver == StorageDriverPostgres {
		binding, openErr := openTenantStorage(ctx, config, false, nil)
		if openErr != nil {
			return zero, openErr
		}
		defer binding.Close()
		store = binding.Store
		supported, schemaErr = postgres.LatestSchemaVersion()
	} else {
		store, err = sqlite.Open(config.Storage.SQLitePath, sqlite.Options{TenantID: d.MustParseID(config.Storage.TenantID), TenantName: config.Auth.BootstrapTenantName})
		supported, schemaErr = sqlite.LatestSchemaVersion()
	}
	if err != nil {
		return zero, err
	}
	defer store.Close()
	version, err := store.SchemaVersion(ctx)
	if err != nil || schemaErr != nil || version != supported {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "migrate the drained installation before authorizing an operator")
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return zero, err
	}
	fingerprint := sha256.Sum256(raw)
	admin, ok := store.(ports.SecurityAdministrationStore)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "transactional operator authorization unavailable")
	}
	return admin.ApplySecurityAdmin(ctx, ports.SecurityAdminRequest{
		Command:     ports.SecurityAdminCommand{TenantID: intent.TenantID, ExpectedVersion: intent.ExpectedVersion, Operation: "authorize_installation_operator"},
		PrincipalID: intent.PrincipalID, Actor: d.ActorRef{Kind: d.ActorKindService, Provider: "wos", ID: "host-operator"},
		IdempotencyKey: intent.IdempotencyKey, Fingerprint: hex.EncodeToString(fingerprint[:]), Now: systemClock{}.Now().UTC(),
		HostOperator: true, OperatorReason: intent.Reason,
	})
}

func RunInstallationOperatorCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 || args[0] != "authorize" || args[1] != "--file" {
		fmt.Fprintln(stderr, "usage: wos installation authorize --file FILE (offline; stop all old writers first)")
		return 2
	}
	file, err := os.Open(args[2])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	intent, err := DecodeInstallationOperatorIntent(file)
	file.Close()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := AuthorizeInstallationOperator(ctx, ConfigFromEnv(), intent)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err = json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
