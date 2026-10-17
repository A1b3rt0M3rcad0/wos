package acceptance_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ht "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/http"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// Qualify a generic external source projection against real authenticated HTTP
// and durable storage. This is not a fixture of an application-specific rule.
func TestClientEvidenceCommitLossRotationAndExplicitAchievement(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			var store interface {
				ports.TransactionManager
				ports.SecurityStore
			}
			switch backend {
			case "memory":
				store = memory.New()
			case "sqlite":
				value, err := sqlite.Open(filepath.Join(t.TempDir(), "client.db"), sqlite.Options{MigrateOnOpen: true})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { value.Close() })
				store = value
			case "postgres":
				if os.Getenv("WOS_TEST_POSTGRES_DSN") == "" {
					t.Skip("real PostgreSQL required")
				}
				value, err := postgres.Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), postgres.Options{MigrateOnOpen: true, Schema: fmt.Sprintf("client_evidence_%d", time.Now().UnixNano())})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { value.Close() })
				store = value
			}
			qualifyClientEvidence(t, store)
		})
	}
}

func qualifyClientEvidence(t *testing.T, store interface {
	ports.TransactionManager
	ports.SecurityStore
}) {
	t.Helper()
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	generator := &ids{}
	tenant := d.MustParseID("0199ff41-0000-7000-8000-000000000001")
	security := &a.SecurityService{Store: store, Clock: clock{}, IDs: generator}
	const adminToken = "client-qualification-admin-token-at-least-32"
	must(security.Bootstrap(ctx, ports.Tenant{ID: tenant, Name: "generic client"}, "operator", adminToken))
	admin, err := security.Authenticate(ctx, adminToken)
	must(err)
	adminCtx := a.WithIdentity(ctx, admin)
	must(security.SetGrant(adminCtx, ports.TenantGrant{TenantID: tenant, PrincipalID: "source-client", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionRecordsWrite}}))
	actor := d.ActorRef{Kind: d.ActorKindService, Provider: "generic-client", ID: "source-projector"}
	oldCredential, oldToken, err := security.IssueCredential(adminCtx, tenant, "source-client", actor, time.Now().Add(time.Hour))
	must(err)
	_, newToken, err := security.IssueCredential(adminCtx, tenant, "source-client", actor, time.Now().Add(time.Hour))
	must(err)
	service, err := a.NewAuthorizedService(store, clock{}, generator, security)
	must(err)
	handler, err := ht.New(ht.Options{Prefix: "/api/v1", Service: service, IDs: generator, Security: security, RequestTimeout: 10 * time.Second, ResolveIdentity: func(r *http.Request) (a.Identity, error) {
		return security.Authenticate(r.Context(), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	}})
	must(err)
	var drop atomic.Bool
	var registrationCalls atomic.Int64
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/register_evidence") {
			registrationCalls.Add(1)
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/register_evidence") && drop.CompareAndSwap(true, false) {
			// Execute and commit the real handler, then lose the entire response.
			committed := httptest.NewRecorder()
			handler.ServeHTTP(committed, r)
			if committed.Code != http.StatusOK {
				t.Errorf("fixture failed before commit: %d %s", committed.Code, committed.Body.String())
			}
			connection, _, hijackErr := w.(http.Hijacker).Hijack()
			if hijackErr != nil {
				t.Error(hijackErr)
				return
			}
			connection.Close()
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer endpoint.Close()
	operator, err := sdk.New(endpoint.URL, adminToken, endpoint.Client())
	must(err)
	oldClient, err := sdk.New(endpoint.URL, oldToken, endpoint.Client())
	must(err)
	rotated, err := sdk.New(endpoint.URL, newToken, endpoint.Client())
	must(err)
	outcome, err := operator.CreateOutcome(ctx, "client-outcome-0001", a.CreateOutcomeCommand{TenantID: tenant, Title: "Verifiable client result", DesiredState: "Explicitly evaluated result", Priority: d.PriorityNormal})
	must(err)
	scope := outcome.Value.Scope()
	criterion, err := operator.AddCriterion(ctx, "client-criterion-0001", a.AddCriterionCommand{Owner: outcome.Value.Ref(), ExpectedVersion: outcome.Value.Version, Title: "Independent check", Required: true, VerificationMode: d.VerificationModeEvidenceReview})
	must(err)
	getOutcome := func() d.Outcome {
		t.Helper()
		raw, readErr := operator.QueryJSON(ctx, "/tenants/"+tenant.String()+"/outcomes/"+scope.OutcomeID.String())
		must(readErr)
		var result a.ReadResult[d.Outcome]
		must(json.Unmarshal(raw, &result))
		return result.Value
	}
	current := getOutcome()
	_, err = operator.ActivateOutcome(ctx, "client-activate-0001", a.ActivateOutcomeCommand{Scope: scope, ExpectedVersion: current.Version})
	must(err)
	identityBytes, err := json.Marshal([]string{tenant.String(), scope.OutcomeID.String(), "generic-client", "event-0001", "register_evidence"})
	must(err)
	digest := sha256.Sum256(identityBytes)
	key := "source-event-v1-" + hex.EncodeToString(digest[:])
	command := a.RegisterEvidenceCommand{Scope: scope, EvidenceType: d.EvidenceTypeSource, Description: "Observed external source event", SourceRef: d.SourceReference{Provider: "generic-client", ID: "event-0001"}, CapturedAt: time.Now().UTC()}
	drop.Store(true)
	if _, err = oldClient.RegisterEvidence(ctx, key, command); err == nil {
		t.Fatal("lost response was falsely reported as confirmed")
	}
	if registrationCalls.Load() != 1 {
		t.Fatal("SDK transport silently retried the lost mutation")
	}
	must(security.RevokeCredential(adminCtx, tenant, oldCredential.ID))
	if _, err = oldClient.RegisterEvidence(ctx, key, command); err == nil {
		t.Fatal("revoked token recovered protected evidence")
	}
	replayed, err := rotated.RegisterEvidence(ctx, key, command)
	must(err)
	if !replayed.IdempotentReplay || replayed.CommandID.IsZero() || replayed.Value.SourceRef.ID != "event-0001" {
		t.Fatal("rotation did not recover the original receipt/evidence")
	}
	repeated, err := rotated.RegisterEvidence(ctx, key, command)
	must(err)
	if repeated.Value.ID != replayed.Value.ID || repeated.CommandID != replayed.CommandID || repeated.OutcomeRevision != replayed.OutcomeRevision {
		t.Fatal("redelivery created another evidence or commit")
	}
	receipt, err := rotated.GetCommandReceiptObservation(ctx, tenant, replayed.CommandID)
	must(err)
	var accepted sdk.CommandResult[d.Evidence]
	must(json.Unmarshal(receipt.ResponseJSON, &accepted))
	if receipt.ResultOmitted || accepted.Value.ID != replayed.Value.ID || accepted.CommandID != replayed.CommandID {
		t.Fatal("durable result does not bind the accepted evidence")
	}
	changed := command
	changed.Description = "Different semantic intention"
	_, err = rotated.RegisterEvidence(ctx, key, changed)
	var remote *sdk.Error
	if !errors.As(err, &remote) || remote.Code != "idempotency_conflict" {
		t.Fatal("changed intention reused the original key", err)
	}
	_, err = rotated.CreateEvidenceLink(ctx, "client-evidence-link-0001", a.CreateEvidenceLinkCommand{Scope: scope, EvidenceID: replayed.Value.ID, TargetRef: outcome.Value.Ref(), CriterionID: &criterion.Value.ID, Stance: d.EvidenceStanceSupports, Rationale: "External observation is input to an explicit evaluation"})
	must(err)
	current = getOutcome()
	if current.Lifecycle != d.OutcomeLifecycleActive || len(current.Criteria.CurrentAssessments) != 0 {
		t.Fatal("Evidence/EvidenceLink evaluated or achieved the Outcome")
	}
	_, err = operator.AchieveOutcome(ctx, "client-no-assessment-0001", a.AchieveOutcomeCommand{Scope: scope, ExpectedVersion: current.Version, Reason: "link is not an assessment"})
	if err == nil {
		t.Fatal("unassessed evidence concluded the Outcome")
	}
	_, err = operator.RecordCriterionAssessment(ctx, "client-explicit-assessment-0001", a.RecordCriterionAssessmentCommand{Owner: outcome.Value.Ref(), ExpectedVersion: current.Version, CriterionID: criterion.Value.ID, CriterionRevision: criterion.Value.Revision, Result: d.AssessmentResultMet, Rationale: "Authorized explicit test-fixture evaluation", EvidenceIDs: []d.ID{replayed.Value.ID}})
	must(err)
	current = getOutcome()
	if current.Lifecycle != d.OutcomeLifecycleActive {
		t.Fatal("assessment automatically achieved the Outcome")
	}
	_, err = operator.AchieveOutcome(ctx, "client-explicit-achievement-0001", a.AchieveOutcomeCommand{Scope: scope, ExpectedVersion: current.Version, Reason: "Authorized explicit test-fixture conclusion"})
	must(err)
	if getOutcome().Lifecycle != d.OutcomeLifecycleAchieved {
		t.Fatal("explicit authorized conclusion failed")
	}
	unit, err := store.Begin(ctx)
	must(err)
	defer unit.Rollback()
	evidence, err := unit.(ports.DocumentaryUnitOfWork).Evidence().ListByOutcome(ctx, scope)
	must(err)
	if len(evidence) != 1 || evidence[0].ID != replayed.Value.ID {
		t.Fatal("timeout/redelivery/credential rotation duplicated the evidence")
	}
}
