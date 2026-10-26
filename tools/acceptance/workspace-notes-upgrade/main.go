// Exercise additive note persistence against a separately produced historical DB.
// This helper consumes only opt-in, isolated fixtures, never a running service DB.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	"os"
	"time"
)

// BuildCommit is set by the qualification build, independently of the fixture.
var BuildCommit = "unqualified"

type clock struct{}

func (clock) Now() time.Time { return time.Now().UTC() }

type ids struct{}

func (ids) NewID() (d.ID, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = value[6]&15 | 0x70
	value[8] = value[8]&63 | 0x80
	raw := hex.EncodeToString(value[:])
	return d.ParseID(raw[:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20] + "-" + raw[20:])
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	backend := flag.String("backend", "sqlite", "isolated fixture backend")
	path := flag.String("sqlite", "", "isolated SQLite file")
	schema := flag.String("schema", "", "isolated PostgreSQL schema")
	mode := flag.String("mode", "migrate", "migrate or note")
	flag.Parse()
	if *mode == "identity" {
		sqliteVersion, err := sqlite.LatestSchemaVersion()
		if err != nil {
			return err
		}
		postgresVersion, err := postgres.LatestSchemaVersion()
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"source": BuildCommit, "sqlite_schema": sqliteVersion, "postgres_schema": postgresVersion})
	}
	var store interface {
		ports.TransactionManager
		ports.SecurityStore
		Close() error
		SchemaVersion(context.Context) (int64, error)
	}
	var err error
	if *backend == "sqlite" {
		store, err = sqlite.Open(*path, sqlite.Options{MigrateOnOpen: true})
	} else if *backend == "postgres" {
		store, err = postgres.Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), postgres.Options{Schema: *schema, MigrateOnOpen: true})
	} else {
		return fmt.Errorf("unsupported fixture backend")
	}
	if err != nil {
		return err
	}
	defer store.Close()
	ctx := context.Background()
	version, err := store.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	if *mode == "migrate" {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": version})
	}
	if *mode != "note" {
		return fmt.Errorf("unsupported fixture mode")
	}
	security := a.SecurityService{Store: store, Clock: clock{}, IDs: ids{}}
	identity, err := security.Authenticate(ctx, os.Getenv("WOS_HISTORICAL_BOOTSTRAP_TOKEN"))
	if err != nil {
		return err
	}
	ctx = a.WithIdentity(ctx, identity)
	service, err := a.NewAuthorizedService(store, clock{}, ids{}, security)
	if err != nil {
		return err
	}
	scope := d.Scope{NamespaceID: d.MustParseID("0199a555-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199a555-0000-7000-8000-000000000002")}
	workID := d.MustParseID("0199a555-0000-7000-8000-000000000014")
	before, err := service.GetWorkItem(ctx, scope, workID)
	if err != nil {
		return err
	}
	original, err := json.Marshal(before.Value)
	if err != nil {
		return err
	}
	commandID, err := (ids{}).NewID()
	if err != nil {
		return err
	}
	cc := d.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor, CommandID: commandID, IdempotencyKey: "documentary-note-upgrade-intent"}
	cmd := a.AppendEntityNoteCommand{Scope: scope, SubjectRef: before.Value.Ref(), Type: d.EntityNoteObservation, Body: "A human observation after upgrade. Existing signed work, assessments and acceptance remain unchanged."}
	saved, err := service.AppendEntityNote(ctx, cc, cmd)
	if err != nil {
		return err
	}
	replay, err := service.AppendEntityNote(ctx, cc, cmd)
	if err != nil {
		return err
	}
	if !replay.IdempotentReplay || saved.Value.ID != replay.Value.ID {
		return fmt.Errorf("upgraded note replay changed identity")
	}
	after, err := service.GetWorkItem(ctx, scope, workID)
	if err != nil {
		return err
	}
	current, err := json.Marshal(after.Value)
	if err != nil {
		return err
	}
	if !bytes.Equal(original, current) {
		return fmt.Errorf("documentary note changed historical Task state")
	}
	ref := before.Value.Ref()
	page, err := service.GetWorkspaceTimeline(ctx, scope, a.WorkspaceTimelineQuery{SubjectRef: &ref, Limit: 20})
	if err != nil {
		return err
	}
	found := 0
	for _, entry := range page.Items {
		if entry.ID == saved.Value.ID && entry.Origin == "entity_note" {
			found++
		}
	}
	if found != 1 {
		return fmt.Errorf("upgraded note did not project exactly once")
	}
	restored, err := service.GetEntityNote(ctx, scope, saved.Value.ID)
	if err != nil {
		return err
	}
	if restored.Body != cmd.Body || restored.PrincipalID != identity.PrincipalID {
		return fmt.Errorf("upgraded note provenance changed")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": version, "note_replay": true, "note_timeline_once": true, "original_task_unchanged": true, "lifecycle": string(after.Value.Lifecycle), "recorded_principal": saved.Value.PrincipalID})
}
