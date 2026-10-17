// Package tenants exposes offline installation administration for PostgreSQL
// Tenant placements. Database credentials are operator authority; this command
// does not use browser tokens or infer a destination from a client profile.
package tenants

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
)

// Intent is an explicit operator request. ExpectedVersion is mandatory for any
// existing placement; provision creates a new, never previously allocated ID.
type Intent struct {
	TenantID        domain.ID      `json:"tenant_id"`
	ExpectedVersion domain.Version `json:"expected_version,omitempty"`
	Name            string         `json:"name,omitempty"`
	Reason          string         `json:"reason"`
}

func decodeOperatorRequest(reader io.Reader, value any) error {
	raw, err := io.ReadAll(io.LimitReader(reader, (16<<10)+1))
	if err != nil || len(raw) > 16<<10 {
		return fmt.Errorf("operator request exceeds the 16 KiB limit or cannot be read")
	}
	if signing.DecodeStrict(raw, value, 16<<10) != nil {
		return fmt.Errorf("invalid operator request JSON: use one object with exact fields and no duplicate keys")
	}
	return nil
}
func decodeIntent(reader io.Reader) (Intent, error) {
	var value Intent
	if err := decodeOperatorRequest(reader, &value); err != nil {
		return value, err
	}
	if value.TenantID.Validate() != nil || strings.TrimSpace(value.Reason) == "" || len(value.Reason) > 4096 || len(value.Name) > 200 {
		return value, fmt.Errorf("Tenant request requires a valid tenant_id and bounded nonempty reason")
	}
	return value, nil
}

func usage(out io.Writer) {
	fmt.Fprintln(out, `Usage: wos storage tenant <list|show|provision|retry-provision|suspend|migrate|resume|delete|import-shared> [options]
  --dsn-env NAME       operator PostgreSQL DSN environment variable (default WOS_POSTGRES_DSN)
  --catalog-schema NAME  overrides WOS_POSTGRES_CATALOG_SCHEMA (default wos_control)
  list [--after UUID] [--limit 100]
  show --tenant UUID
  provision (--file REQUEST.json | --from-env) [--if-absent]
  import-shared --file REQUEST.json --writers-stopped
Import request: {"source_schema":"public","reason":"approved offline repartition"}
  retry-provision|suspend|migrate|resume|delete --file REQUEST.json --writers-stopped
Request: {"tenant_id":"UUID","expected_version":1,"reason":"operator reason"}
Provision also requires "name" and omits expected_version. --file - reads stdin.
--from-env reads WOS_TENANT_ID and WOS_BOOTSTRAP_TENANT_NAME (defaults to the ID).
--if-absent accepts only an existing active Tenant with the same name and ID.
First API startup
initializes its own configured credentials and issuer; this command prints none.
Migrate requires a suspended Tenant and leaves it suspended with a new version;
verify the migration before explicitly resuming it.
Stop all Tenant API/worker processes before lifecycle operations. Live runtime
leases prevent the change. Delete requires a suspended Tenant or interrupted
provisioning and removes its schema permanently; catalog tombstones remain.
External files and backups require separate retention handling.`)
}

// Run executes the native operator command. Failure messages deliberately do not
// expose raw driver diagnostics, because PostgreSQL errors can contain a DSN.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return RunWithInput(ctx, args, os.Stdin, stdout, stderr)
}

// RunWithInput supports an explicit stdin request stream without changing process state.
func RunWithInput(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		usage(stdout)
		return 0
	}
	action := args[0]
	switch action {
	case "list", "show", "provision", "retry-provision", "suspend", "migrate", "resume", "delete", "import-shared":
	default:
		usage(stderr)
		return 2
	}
	flags := flag.NewFlagSet("storage tenant "+action, flag.ContinueOnError)
	flags.SetOutput(stderr)
	dsnEnv := flags.String("dsn-env", "WOS_POSTGRES_DSN", "operator DSN environment variable")
	defaultCatalog := os.Getenv("WOS_POSTGRES_CATALOG_SCHEMA")
	if defaultCatalog == "" {
		defaultCatalog = "wos_control"
	}
	catalogSchema := flags.String("catalog-schema", defaultCatalog, "catalog schema (defaults to WOS_POSTGRES_CATALOG_SCHEMA or wos_control)")
	file := flags.String("file", "", "operator request JSON file; - reads stdin")
	fromEnv := flags.Bool("from-env", false, "provision WOS_TENANT_ID and WOS_BOOTSTRAP_TENANT_NAME")
	ifAbsent := flags.Bool("if-absent", false, "provision only when absent; require exact active existing name and ID")
	tenant := flags.String("tenant", "", "Tenant ID for show")
	after := flags.String("after", "", "last Tenant ID from previous page")
	limit := flags.Int("limit", 100, "maximum list results (1..1000)")
	stopped := flags.Bool("writers-stopped", false, "all Tenant API and worker processes have been stopped")
	flags.Usage = func() { usage(stdout) }
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	fail := func(message string) int { fmt.Fprintln(stderr, message); return 1 }
	if flags.NArg() != 0 {
		return fail("unexpected Tenant command arguments")
	}
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(*dsnEnv) {
		return fail("invalid DSN environment variable name")
	}
	if *fromEnv && (action != "provision" || *file != "" || *tenant != "" || *after != "") {
		return fail("--from-env is exclusive to provision and cannot be combined with --file or target flags")
	}
	if *ifAbsent && action != "provision" {
		return fail("--if-absent is available only for provision")
	}
	requestReader := func() (io.Reader, func(), error) {
		if *file == "-" {
			if stdin == nil {
				return nil, func() {}, fmt.Errorf("request stdin unavailable")
			}
			return stdin, func() {}, nil
		}
		f, err := os.Open(*file)
		if err != nil {
			return nil, func() {}, err
		}
		return f, func() { _ = f.Close() }, nil
	}
	var intent Intent
	var importRequest postgres.TenantPlacementImport
	if action == "import-shared" {
		if *file == "" || *tenant != "" || *after != "" || !*stopped {
			return fail("import-shared requires --file REQUEST.json and --writers-stopped after draining all source processes")
		}
		f, closeRequest, err := requestReader()
		if err != nil {
			return fail("cannot open import request file")
		}
		var request struct {
			SourceSchema string `json:"source_schema"`
			Reason       string `json:"reason"`
		}
		err = decodeOperatorRequest(f, &request)
		closeRequest()
		if err != nil || !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(request.SourceSchema) || strings.TrimSpace(request.Reason) == "" || len(request.Reason) > 4096 {
			return fail("import requires a valid source_schema and nonempty reason")
		}
		importRequest = postgres.TenantPlacementImport{SourceSchema: request.SourceSchema, Reason: request.Reason, WritersStopped: true}
	} else if action != "list" && action != "show" {
		if *fromEnv {
			id, parseErr := domain.ParseID(os.Getenv("WOS_TENANT_ID"))
			if parseErr != nil {
				return fail("--from-env requires a valid WOS_TENANT_ID")
			}
			name := os.Getenv("WOS_BOOTSTRAP_TENANT_NAME")
			if name == "" {
				name = id.String()
			}
			if strings.TrimSpace(name) == "" || len(name) > 200 {
				return fail("WOS_BOOTSTRAP_TENANT_NAME must be nonempty and at most 200 bytes")
			}
			intent = Intent{TenantID: id, Name: name, Reason: "Provision Tenant from explicit deployment configuration"}
		} else {
			if *file == "" || *tenant != "" || *after != "" {
				return fail("Tenant mutation requires only --file REQUEST.json for its target")
			}
			f, closeRequest, err := requestReader()
			if err != nil {
				return fail("cannot open Tenant request file")
			}
			intent, err = decodeIntent(f)
			closeRequest()
			if err != nil {
				return fail(err.Error())
			}
		}
		if action == "provision" {
			if intent.ExpectedVersion != 0 || strings.TrimSpace(intent.Name) == "" {
				return fail("provision requires name and no expected_version")
			}
		} else {
			if intent.ExpectedVersion == 0 || intent.Name != "" {
				return fail("existing Tenant mutation requires expected_version and no name")
			}
			if !*stopped {
				return fail("stop all Tenant processes and provide --writers-stopped")
			}
		}
	} else if *file != "" || *stopped {
		return fail("read-only Tenant command does not accept a mutation file or writers acknowledgement")
	}
	var id, afterID domain.ID
	if action == "show" {
		var err error
		id, err = domain.ParseID(*tenant)
		if err != nil || *after != "" {
			return fail("show requires a valid --tenant UUID")
		}
	}
	if action == "list" {
		if *tenant != "" || *limit < 1 || *limit > 1000 {
			return fail("list requires a limit from 1 to 1000")
		}
		if *after != "" {
			var err error
			afterID, err = domain.ParseID(*after)
			if err != nil {
				return fail("invalid --after Tenant ID")
			}
		}
	}
	dsn := os.Getenv(*dsnEnv)
	if dsn == "" {
		return fail("operator PostgreSQL DSN environment variable is not set")
	}
	catalog, err := postgres.OpenTenantCatalog(ctx, dsn, postgres.CatalogOptions{Schema: *catalogSchema, MaxOpenConnections: 2, MigrateOnOpen: action == "provision" || action == "import-shared"})
	if err != nil {
		return fail("cannot open Tenant catalog; check operator database access and catalog schema")
	}
	defer catalog.Close()
	initialize := func(ctx context.Context, placement postgres.TenantPlacement) error {
		options := placement.InitializerOptions()
		options.MaxOpenConnections = 2
		store, err := postgres.Open(dsn, options)
		if err != nil {
			return err
		}
		return store.Close()
	}
	var result any
	switch action {
	case "import-shared":
		if checkErr := catalog.CheckTenantPlacementSource(ctx, importRequest.SourceSchema); checkErr != nil {
			return fail("source schema is absent, reserved, incompatible or already fenced; no migration was attempted")
		}
		source, openErr := postgres.Open(dsn, postgres.Options{Schema: importRequest.SourceSchema, MigrateOnOpen: true, MaxOpenConnections: 2})
		if openErr != nil {
			return fail("cannot prepare source schema; active legacy work must be reconciled before import")
		}
		if closeErr := source.Close(); closeErr != nil {
			return fail("cannot close source schema before import")
		}
		result, err = catalog.ApplyTenantPlacement(ctx, importRequest)
	case "list":
		result, err = catalog.ListTenants(ctx, afterID, *limit)
	case "show":
		result, err = catalog.GetTenant(ctx, id)
	case "provision":
		if *ifAbsent {
			existing, lookupErr := catalog.GetTenant(ctx, intent.TenantID)
			if lookupErr == nil {
				if existing.State != postgres.TenantActive || existing.Name != intent.Name {
					return fail("existing Tenant must be active with the exact requested name; no provisioning change was made")
				}
				result = existing
			} else if code, ok := domain.ErrorCodeOf(lookupErr); ok && code == domain.ErrorCodeNotFound {
				result, err = catalog.ProvisionTenant(ctx, postgres.TenantProvision{TenantID: intent.TenantID, Name: intent.Name, Reason: intent.Reason}, initialize)
			} else {
				err = lookupErr
			}
		} else {
			result, err = catalog.ProvisionTenant(ctx, postgres.TenantProvision{TenantID: intent.TenantID, Name: intent.Name, Reason: intent.Reason}, initialize)
		}
	case "retry-provision":
		result, err = catalog.RetryProvisionTenant(ctx, intent.TenantID, intent.ExpectedVersion, intent.Reason, initialize)
	case "migrate":
		result, err = catalog.MigrateTenant(ctx, intent.TenantID, intent.ExpectedVersion, intent.Reason, initialize)
	case "suspend":
		result, err = catalog.ChangeTenantState(ctx, intent.TenantID, intent.ExpectedVersion, postgres.TenantSuspended, intent.Reason)
	case "resume":
		result, err = catalog.ChangeTenantState(ctx, intent.TenantID, intent.ExpectedVersion, postgres.TenantActive, intent.Reason)
	case "delete":
		result, err = catalog.DeleteTenant(ctx, intent.TenantID, intent.ExpectedVersion, intent.Reason)
	}
	if err != nil {
		code, ok := domain.ErrorCodeOf(err)
		if ok {
			return fail("Tenant operation refused: " + string(code) + "; inspect the placement version, state and runtime leases")
		}
		return fail("Tenant operation failed; inspect database availability and operator configuration; preserve the request")
	}
	if err = json.NewEncoder(stdout).Encode(map[string]any{"operation": "tenant_" + strings.ReplaceAll(action, "-", "_"), "data": result}); err != nil {
		return fail("cannot write Tenant operation result")
	}
	return 0
}
