package sqlite

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestActualV010DatasetMigrationsPreserveProofAndCommandFingerprints(t *testing.T) {
	synctest.Test(t, testActualV010DatasetMigrations)
}

func testActualV010DatasetMigrations(t *testing.T) {
	var fixture struct {
		NamespaceID  d.ID      `json:"namespace_id"`
		OutcomeID    d.ID      `json:"outcome_id"`
		WorkItemID   d.ID      `json:"work_item_id"`
		WorkVersion  d.Version `json:"work_version"`
		ConclusionID d.ID      `json:"conclusion_id"`
		CriterionID  d.ID      `json:"criterion_id"`
		AssessmentID d.ID      `json:"assessment_id"`
		Checksum     string    `json:"sha256_uncompressed"`
	}
	metadata, err := os.ReadFile("testdata/v010-contract-migration.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(metadata, &fixture); err != nil {
		t.Fatal(err)
	}
	compressed, err := os.ReadFile("testdata/v010-contract-migration.db.gz")
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(io.LimitReader(gz, 8<<20))
	gz.Close()
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != fixture.Checksum {
		t.Fatal("legacy dataset checksum")
	}
	path := filepath.Join(t.TempDir(), "legacy.db")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, Options{MigrateOnOpen: false})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range migrations {
		if item.version > 34 {
			break
		}
		if err := store.applyMigration(ctx, item); err != nil {
			t.Fatalf("apply historical migration %s: %v", item.name, err)
		}
	}
	if version, err := store.SchemaVersion(ctx); err != nil || version != 34 {
		t.Fatalf("legacy fixture schema = %d, err=%v; want 34 before the ownership contract", version, err)
	}
	scope := d.Scope{NamespaceID: fixture.NamespaceID, OutcomeID: fixture.OutcomeID}
	// Replay at the backup's last command, while its original receipts are retained.
	// The bubble also controls SQLite's internal retention clock. Application Clock
	// injection alone cannot isolate that wall-clock check from the current date.
	var lastCommandAt, firstExpiry int64
	if err = store.db.QueryRowContext(ctx, `
SELECT MAX(created_at), MIN(expires_at) FROM idempotency_records
WHERE namespace_id = ?`, fixture.NamespaceID.String()).Scan(&lastCommandAt, &firstExpiry); err != nil {
		t.Fatal(err)
	}
	if lastCommandAt >= firstExpiry {
		t.Fatal("historical receipts have no shared retention window")
	}
	// synctest advances this virtual interval immediately, without a real sleep.
	time.Sleep(time.UnixMicro(lastCommandAt).Sub(time.Now()))
	service, err := a.NewService(store, sqliteFixedClock{time.UnixMicro(lastCommandAt).UTC()}, &sqliteSequenceIDs{prefix: "0199ff91", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199ff91-0000-7000-8000-000000000001", "legacy-fixture-claim_work_item")
	cc.PrincipalID = "legacy-fixture"
	cc.Actor.ID = "legacy-fixture"
	claim := a.ClaimWorkItemCommand{Scope: scope, WorkItemID: fixture.WorkItemID, ExpectedVersion: 1, TTL: 300 * time.Second}
	replay, err := service.ClaimWorkItem(ctx, cc, claim)
	if err != nil || !replay.IdempotentReplay {
		t.Fatal("original 0.1 claim fingerprint changed", err)
	}
	cc.IdempotencyKey = "legacy-fixture-attest_criterion"
	attested, err := service.AttestCriterion(ctx, cc, a.AttestCriterionCommand{Owner: d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}, ExpectedVersion: 3, CriterionID: fixture.CriterionID, CriterionRevision: 1, Result: d.AssessmentResultMet, Rationale: "Historic attestation from the original writer"})
	if err != nil || !attested.IdempotentReplay || attested.Value.ID != fixture.AssessmentID {
		t.Fatal("new optional submission field changed 0.1 receipt", err)
	}
	cc.IdempotencyKey = "legacy-upgrade-draining"
	if _, err = service.SetNamespaceWorkProtocol(ctx, cc, a.SetNamespaceWorkProtocolCommand{Scope: scope, ExpectedProtocolVersion: 1, Phase: d.WorkProtocolDraining, Reason: "Stop old 0.1 writers"}); err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = "legacy-upgrade-contracts"
	if _, err = service.SetNamespaceWorkProtocol(ctx, cc, a.SetNamespaceWorkProtocolCommand{Scope: scope, ExpectedProtocolVersion: 2, Phase: d.WorkProtocolContracts, WritersDrained: true, Reason: "Validated historical backup"}); err != nil {
		t.Fatal(err)
	}
	tx, _ := store.Begin(ctx)
	work, err := tx.WorkItems().Get(ctx, scope, fixture.WorkItemID)
	tx.Rollback()
	if err != nil || work.ID != fixture.WorkItemID || work.Version != fixture.WorkVersion || work.Lifecycle != d.WorkItemLifecycleDone || work.CurrentConclusion == nil || work.CurrentConclusion.ID != fixture.ConclusionID || !work.ContractsEnabled || work.CurrentContractID != nil {
		t.Fatal("historical completion was rewritten", err, work)
	}
	cc.IdempotencyKey = "legacy-fixture-claim_work_item"
	after, err := service.ClaimWorkItem(ctx, cc, claim)
	if err != nil || !after.IdempotentReplay || after.CommandID != replay.CommandID || after.OutcomeRevision != replay.OutcomeRevision {
		t.Fatal("old receipt not replayable after cutover", err)
	}
	// The historical dataset contains no Objective at all, so it has no
	// reviewed ownership mapping. The final contract migration must stop rather
	// than invent a parent and leave the legacy state available for remediation.
	if err := store.Migrate(ctx); err == nil || !strings.Contains(err.Error(), "requires zero Tasks without an Objective") {
		t.Fatalf("unmapped legacy Task did not block schema 35: %v", err)
	}
	if version, err := store.SchemaVersion(ctx); err != nil || version != 34 {
		t.Fatalf("blocked migration changed schema version: version=%d err=%v", version, err)
	}
	var objectiveID sql.NullString
	if err := store.db.QueryRowContext(ctx, `SELECT objective_id FROM work_items WHERE id = ?`, fixture.WorkItemID.String()).Scan(&objectiveID); err != nil || objectiveID.Valid {
		t.Fatalf("blocked migration changed legacy ownership: objective=%v err=%v", objectiveID, err)
	}
}
