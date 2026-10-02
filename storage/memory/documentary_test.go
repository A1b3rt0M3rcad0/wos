package memory_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func documentaryUOW(t *testing.T, store *memory.Store) ports.DocumentaryUnitOfWork {
	t.Helper()
	uow, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	documentary, ok := uow.(ports.DocumentaryUnitOfWork)
	if !ok {
		t.Fatal("memory transaction does not implement DocumentaryUnitOfWork")
	}
	return documentary
}

func TestMemoryDocumentaryRepositoriesEnforceImmutableContent(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f400-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199f400-0000-7000-8000-000000000002"),
	}
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	actor := domain.ActorRef{Kind: domain.ActorKindService, Provider: "test", ID: "producer"}

	tx := documentaryUOW(t, store)
	artifact, err := domain.NewArtifact(
		domain.MustParseID("0199f400-0000-7000-8000-000000000010"),
		scope, "report", "report", "https://example.test/report", "application/json",
		"sha256:a", "v1", actor, nil, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Artifacts().Insert(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	evidence, err := domain.NewEvidence(
		domain.MustParseID("0199f400-0000-7000-8000-000000000020"),
		scope,
		domain.EvidenceTypeMeasurement,
		"p95 was 180ms",
		domain.ExternalReference{Provider: "ci", ID: "run-1"},
		actor,
		now,
		&artifact.ID,
		&domain.Measurement{Value: json.RawMessage("180"), Unit: "ms"},
		"scenario-1",
		"sha256:e",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Evidence().Insert(ctx, evidence); err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NewDecision(
		domain.MustParseID("0199f400-0000-7000-8000-000000000030"),
		scope,
		"Store",
		"Choose store",
		"",
		[]string{"SQLite", "PostgreSQL"},
		nil,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := decision.Accept("SQLite", "standalone-first", actor, now); err != nil {
		t.Fatal(err)
	}
	if err := tx.Decisions().Insert(ctx, decision); err != nil {
		t.Fatal(err)
	}
	link, err := domain.NewEvidenceLink(
		domain.MustParseID("0199f400-0000-7000-8000-000000000040"),
		scope,
		evidence.ID,
		decision.Ref(),
		nil,
		domain.EvidenceStanceSupports,
		"supports the choice",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.EvidenceLinks().Insert(ctx, link); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	assertImmutableSaveRejected(t, store, func(uow ports.DocumentaryUnitOfWork) error {
		value, err := uow.Artifacts().Get(ctx, scope, artifact.ID)
		if err != nil {
			return err
		}
		expected := value.Version
		value.URI = "https://example.test/rewritten"
		value.Version++
		value.UpdatedAt = now.Add(time.Minute)
		return uow.Artifacts().Save(ctx, value, expected)
	})
	assertImmutableSaveRejected(t, store, func(uow ports.DocumentaryUnitOfWork) error {
		value, err := uow.Evidence().Get(ctx, scope, evidence.ID)
		if err != nil {
			return err
		}
		expected := value.Version
		value.Description = "rewritten observation"
		value.Version++
		value.UpdatedAt = now.Add(time.Minute)
		return uow.Evidence().Save(ctx, value, expected)
	})
	assertImmutableSaveRejected(t, store, func(uow ports.DocumentaryUnitOfWork) error {
		value, err := uow.Decisions().Get(ctx, scope, decision.ID)
		if err != nil {
			return err
		}
		expected := value.Version
		value.Title = "rewritten accepted decision"
		value.Version++
		value.UpdatedAt = now.Add(time.Minute)
		return uow.Decisions().Save(ctx, value, expected)
	})
	assertImmutableSaveRejected(t, store, func(uow ports.DocumentaryUnitOfWork) error {
		value, err := uow.EvidenceLinks().Get(ctx, scope, link.ID)
		if err != nil {
			return err
		}
		expected := value.Version
		value.Rationale = "rewritten stance rationale"
		value.Version++
		value.UpdatedAt = now.Add(time.Minute)
		return uow.EvidenceLinks().Save(ctx, value, expected)
	})
}

func assertImmutableSaveRejected(
	t *testing.T,
	store *memory.Store,
	mutate func(ports.DocumentaryUnitOfWork) error,
) {
	t.Helper()
	uow := documentaryUOW(t, store)
	defer uow.Rollback()
	if err := mutate(uow); err == nil {
		t.Fatal("documentary repository accepted immutable content rewrite")
	}
}
