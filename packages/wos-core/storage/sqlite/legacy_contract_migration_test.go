package sqlite

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActualV010DatasetMigrationsPreserveProofAndCommandFingerprints(t *testing.T) {
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
	store := openTestStore(t, path)
	ctx := context.Background()
	scope := d.Scope{NamespaceID: fixture.NamespaceID, OutcomeID: fixture.OutcomeID}
	service, err := a.NewService(store, sqliteFixedClock{time.Now().UTC()}, &sqliteSequenceIDs{prefix: "0199ff91", next: 1})
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
}
