package application

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestHistoricalUnsignedFingerprintLeavesConsumerContextOpaque(t *testing.T) {
	id := domain.MustParseID("0199f490-0000-7000-8000-000000000021")
	type command struct {
		TenantID   domain.ID
		Owner      domain.EntityRef
		References []domain.EntityRef
		Metadata   map[string]any
		Opaque     json.RawMessage
		Text       string
		Version    uint64
	}
	ref := domain.EntityRef{Scope: domain.Scope{TenantID: id, OutcomeID: id}, Kind: domain.EntityKindOutcome, ID: id}
	value := command{id, ref, []domain.EntityRef{ref}, map[string]any{"tenant_id": "customer-defined", "TenantID": "consumer-value", "nested": map[string]string{"tenant_id": "leave alone"}}, json.RawMessage(`{"tenant_id":"signed-content","NamespaceID":"opaque"}`), `"tenant_id":"user text"`, ^uint64(0)}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	old, changed, err := historicalUnsignedPayload(raw, reflect.TypeOf(value))
	if err != nil || !changed {
		t.Fatal("typed historical projection failed", err)
	}
	if bytes.Count(old, []byte(`"namespace_id":`)) != 2 || bytes.Count(old, []byte(`"NamespaceID":`)) != 2 {
		t.Fatalf("typed references were not projected precisely: %s", old)
	}
	for _, preserved := range []string{`"tenant_id":"customer-defined"`, `"TenantID":"consumer-value"`, `"tenant_id":"leave alone"`, `"tenant_id":"signed-content"`, `18446744073709551615`, `\"tenant_id\":\"user text\"`} {
		if !bytes.Contains(old, []byte(preserved)) {
			t.Fatalf("historical projection modified consumer data or precision: %s", preserved)
		}
	}
	// Field ordering outside the renamed typed keys remains byte-for-byte.
	expected := bytes.Replace(raw, []byte(`"TenantID":"`+id.String()+`"`), []byte(`"NamespaceID":"`+id.String()+`"`), 1)
	expected = bytes.ReplaceAll(expected, []byte(`"tenant_id":"`+id.String()+`"`), []byte(`"namespace_id":"`+id.String()+`"`))
	if !bytes.Equal(expected, old) {
		t.Fatalf("historical projection changed field ordering:\n%s\n%s", expected, old)
	}
}

type retainedUnsignedReceipt struct {
	ports.IdempotencyStore
	fingerprint string
	result      domain.StoredCommandResult
	calls       []string
	disappears  bool
}

func (s *retainedUnsignedReceipt) Reserve(_ context.Context, identity domain.IdempotencyIdentity, fingerprint string) (domain.IdempotencyReservation, error) {
	s.calls = append(s.calls, fingerprint)
	if len(s.calls) == 2 && s.disappears {
		return domain.IdempotencyReservation{Identity: identity, Fingerprint: fingerprint}, nil
	}
	if fingerprint != s.fingerprint {
		return domain.IdempotencyReservation{}, domain.NewError(domain.ErrorCodeIdempotencyConflict, "different frozen intention")
	}
	return domain.IdempotencyReservation{Identity: identity, Fingerprint: fingerprint, Replay: &s.result}, nil
}

func TestUnsignedHistoricalReplayRequiresExactCompletedReceipt(t *testing.T) {
	id := domain.MustParseID("0199f490-0000-7000-8000-000000000022")
	actor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "original"}
	command := ActivateOutcomeCommand{Scope: domain.Scope{TenantID: id, OutcomeID: id}, ExpectedVersion: 1}
	meta, err := buildCommandMetadata(domain.CommandContext{Actor: actor}, command)
	if err != nil {
		t.Fatal(err)
	}
	old := json.RawMessage(`{"Scope":{"namespace_id":"` + id.String() + `","outcome_id":"` + id.String() + `"},"ExpectedVersion":1}`)
	oldHash, err := fingerprintCommand(meta.Name, actor, old)
	if err != nil {
		t.Fatal(err)
	}
	identity := domain.IdempotencyIdentity{TenantID: id, PrincipalID: "original", CommandName: meta.Name, IdempotencyKey: "historical-replay-0001"}
	result := domain.StoredCommandResult{CommandID: id, OutcomeRevision: 7, ResponseJSON: json.RawMessage(`{"value":{"namespace_id":"` + id.String() + `"},"outcome_revision":7}`)}
	store := &retainedUnsignedReceipt{fingerprint: oldHash, result: result}
	replay, err := reserveCommandIntent(context.Background(), store, identity, meta, actor, command)
	if err != nil || !replay.IsReplay() || len(store.calls) != 2 || store.calls[0] != meta.Fingerprint || store.calls[1] != oldHash || !bytes.Equal(replay.Replay.ResponseJSON, result.ResponseJSON) {
		t.Fatal("exact retained receipt was not recovered unchanged", err)
	}
	for _, change := range []string{"version", "actor", "signed", "expired"} {
		t.Run(change, func(t *testing.T) {
			changedCommand, changedActor := command, actor
			if change == "version" {
				changedCommand.ExpectedVersion++
			}
			if change == "actor" {
				changedActor.ID = "other"
			}
			changedMeta, err := buildCommandMetadata(domain.CommandContext{Actor: changedActor}, changedCommand)
			if err != nil {
				t.Fatal(err)
			}
			if change == "signed" {
				changedMeta.Name = "AcquireSignedWorkContract"
			}
			store := &retainedUnsignedReceipt{fingerprint: oldHash, result: result, disappears: change == "expired"}
			reservation, err := reserveCommandIntent(context.Background(), store, identity, changedMeta, changedActor, changedCommand)
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeIdempotencyConflict || reservation.IsReplay() {
				t.Fatalf("historical compatibility accepted %s intention: %v", change, err)
			}
			if change == "signed" && len(store.calls) != 1 {
				t.Fatal("signed mutation consulted a historical fingerprint")
			}
		})
	}
	if strings.Contains(string(meta.Payload), "namespace_id") {
		t.Fatal("historical projection altered the current command metadata")
	}
}
