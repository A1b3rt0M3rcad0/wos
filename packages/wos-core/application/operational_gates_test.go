package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

var operationalGateNow = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

func operationalGateID(n int) domain.ID {
	return domain.MustParseID(fmt.Sprintf("0199f100-0000-7000-8000-%012x", n))
}

type operationalGateFixture struct {
	store   *memory.Store
	scope   domain.Scope
	outcome domain.Outcome
	root    domain.Objective
	child   domain.Objective
	task    domain.WorkItem
}

func newOperationalGateFixture(t *testing.T, profile domain.OperationalProfile, rootState, childState domain.ObjectiveLifecycle) operationalGateFixture {
	t.Helper()
	ctx := context.Background()
	store := memory.New()
	var err error
	result := operationalGateFixture{store: store}
	result.outcome, err = domain.NewOutcome(operationalGateID(10), operationalGateID(1), "Execution gate fixture", "", "An operational outcome", domain.PriorityNormal, operationalGateNow)
	if err != nil {
		t.Fatal(err)
	}
	result.outcome.Lifecycle = domain.OutcomeLifecycleActive
	result.outcome.OperationalProfile = profile
	result.scope = result.outcome.Scope()
	result.root = newOperationalGateObjective(t, result.scope, 20, "Root objective", nil, rootState)
	result.child = newOperationalGateObjective(t, result.scope, 21, "Child objective", &result.root.ID, childState)
	result.task, err = domain.NewWorkItemForObjective(operationalGateID(30), result.scope, &result.child.ID, "Executable task", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, operationalGateNow)
	if err != nil {
		t.Fatal(err)
	}
	addOperationalGateCriterion(t, &result.task, operationalGateID(31))
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.Outcomes().Insert(ctx, result.outcome); err != nil {
		t.Fatal(err)
	}
	if err := uow.Objectives().Insert(ctx, result.root); err != nil {
		t.Fatal(err)
	}
	if err := uow.Objectives().Insert(ctx, result.child); err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Insert(ctx, result.task); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return result
}

func newOperationalGateObjective(t *testing.T, scope domain.Scope, id int, title string, parent *domain.ID, lifecycle domain.ObjectiveLifecycle) domain.Objective {
	t.Helper()
	value, err := domain.NewObjective(operationalGateID(id), scope, title, "", domain.PriorityNormal, false, operationalGateNow)
	if err != nil {
		t.Fatal(err)
	}
	value.ParentObjectiveID = parent
	addOperationalGateCriterion(t, &value, operationalGateID(id+100))
	if lifecycle == domain.ObjectiveLifecycleCancelled {
		conclusion := domain.Conclusion{
			PrincipalID: "planner",
			Actor:       domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "planner"},
			Reason:      "fixture terminal state",
			ConcludedAt: operationalGateNow,
		}
		if err := value.Cancel(conclusion, operationalGateNow); err != nil {
			t.Fatal(err)
		}
	} else {
		value.Lifecycle = lifecycle
	}
	return value
}

func addOperationalGateCriterion(t *testing.T, owner interface {
	Ref() domain.EntityRef
}, id domain.ID) {
	t.Helper()
	criterion, err := domain.NewSuccessCriterion(id, owner.Ref(), "A required condition", "", true, domain.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	switch value := owner.(type) {
	case *domain.Objective:
		if err := value.Criteria.Add(criterion); err != nil {
			t.Fatal(err)
		}
	case *domain.WorkItem:
		if err := value.Criteria.Add(criterion); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported criterion owner %T", owner)
	}
}

func withOperationalGateRead(t *testing.T, store *memory.Store, fn func(ports.UnitOfWork)) {
	t.Helper()
	uow, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = uow.Rollback() })
	fn(uow)
}

func hasGateReason(readiness domain.Readiness, reason domain.ReadinessReason) bool {
	for _, value := range readiness.Reasons {
		if value == reason {
			return true
		}
	}
	return false
}

func TestOperationalTaskGatesAreProfileAwareAndDoNotMutate(t *testing.T) {
	tests := []struct {
		name       string
		profile    domain.OperationalProfile
		root       domain.ObjectiveLifecycle
		child      domain.ObjectiveLifecycle
		wantReady  bool
		wantReason domain.ReadinessReason
	}{
		{name: "compatibility keeps planned owner behavior", profile: domain.OperationalProfileCompatibilityV1, root: domain.ObjectiveLifecyclePlanned, child: domain.ObjectiveLifecyclePlanned, wantReady: true},
		{name: "v2 rejects planned owner", profile: domain.OperationalProfileOperationalV2, root: domain.ObjectiveLifecycleInProgress, child: domain.ObjectiveLifecyclePlanned, wantReason: domain.ReadinessReasonObjectiveNotStarted},
		{name: "v2 rejects planned ancestor", profile: domain.OperationalProfileOperationalV2, root: domain.ObjectiveLifecyclePlanned, child: domain.ObjectiveLifecycleInProgress, wantReason: domain.ReadinessReasonAncestorObjectiveNotStarted},
		{name: "v2 rejects terminal owner", profile: domain.OperationalProfileOperationalV2, root: domain.ObjectiveLifecycleInProgress, child: domain.ObjectiveLifecycleCancelled, wantReason: domain.ReadinessReasonObjectiveTerminal},
		{name: "v2 rejects terminal ancestor", profile: domain.OperationalProfileOperationalV2, root: domain.ObjectiveLifecycleCancelled, child: domain.ObjectiveLifecycleInProgress, wantReason: domain.ReadinessReasonAncestorObjectiveTerminal},
		{name: "v2 accepts active ancestry", profile: domain.OperationalProfileOperationalV2, root: domain.ObjectiveLifecycleInProgress, child: domain.ObjectiveLifecycleInProgress, wantReady: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newOperationalGateFixture(t, test.profile, test.root, test.child)
			withOperationalGateRead(t, fixture.store, func(uow ports.UnitOfWork) {
				beforeRoot, err := uow.Objectives().Get(context.Background(), fixture.scope, fixture.root.ID)
				if err != nil {
					t.Fatal(err)
				}
				beforeChild, err := uow.Objectives().Get(context.Background(), fixture.scope, fixture.child.ID)
				if err != nil {
					t.Fatal(err)
				}
				beforeTask, err := uow.WorkItems().Get(context.Background(), fixture.scope, fixture.task.ID)
				if err != nil {
					t.Fatal(err)
				}
				readiness, err := evaluateTaskAcquisition(context.Background(), uow, beforeTask, operationalGateNow)
				if err != nil {
					t.Fatal(err)
				}
				if readiness.Ready != test.wantReady {
					t.Fatalf("ready=%v reasons=%v", readiness.Ready, readiness.Reasons)
				}
				if test.wantReason != "" && !hasGateReason(readiness, test.wantReason) {
					t.Fatalf("missing %s in reasons %v", test.wantReason, readiness.Reasons)
				}
				if test.wantReason != "" && readiness.Scope.Ready {
					t.Fatalf("scope dimension remained ready for %s", test.wantReason)
				}
				if readiness.Definition.Ready == false {
					t.Fatal("complete Task definition was rejected")
				}
				afterRoot, _ := uow.Objectives().Get(context.Background(), fixture.scope, fixture.root.ID)
				afterChild, _ := uow.Objectives().Get(context.Background(), fixture.scope, fixture.child.ID)
				afterTask, _ := uow.WorkItems().Get(context.Background(), fixture.scope, fixture.task.ID)
				if beforeRoot.Version != afterRoot.Version || beforeChild.Version != afterChild.Version || beforeTask.Version != afterTask.Version {
					t.Fatal("readiness evaluation mutated an aggregate")
				}
			})
		})
	}
}

func TestOperationalObjectiveGateAndTaskScopeReasons(t *testing.T) {
	t.Run("v2 objective start requires in-progress ancestors while legacy remains compatible", func(t *testing.T) {
		for _, test := range []struct {
			name       string
			profile    domain.OperationalProfile
			wantReady  bool
			wantReason domain.ReadinessReason
		}{
			{name: "compatibility", profile: domain.OperationalProfileCompatibilityV1, wantReady: true},
			{name: "operational v2", profile: domain.OperationalProfileOperationalV2, wantReason: domain.ReadinessReasonAncestorObjectiveNotStarted},
		} {
			t.Run(test.name, func(t *testing.T) {
				fixture := newOperationalGateFixture(t, test.profile, domain.ObjectiveLifecyclePlanned, domain.ObjectiveLifecyclePlanned)
				withOperationalGateRead(t, fixture.store, func(uow ports.UnitOfWork) {
					objective, err := uow.Objectives().Get(context.Background(), fixture.scope, fixture.child.ID)
					if err != nil {
						t.Fatal(err)
					}
					readiness, err := evaluateObjectiveExecutionGate(context.Background(), uow, objective, operationalGateNow)
					if err != nil {
						t.Fatal(err)
					}
					if readiness.Ready != test.wantReady {
						t.Fatalf("ready=%v reasons=%v", readiness.Ready, readiness.Reasons)
					}
					if test.wantReason != "" && !hasGateReason(readiness, test.wantReason) {
						t.Fatalf("missing %s in reasons %v", test.wantReason, readiness.Reasons)
					}
				})
			})
		}
	})

	t.Run("missing Task Objective differs from a dangling Objective reference", func(t *testing.T) {
		for _, missingOwner := range []bool{false, true} {
			name := "nil objective"
			if missingOwner {
				name = "missing objective record"
			}
			t.Run(name, func(t *testing.T) {
				fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
				if missingOwner {
					missingID := operationalGateID(998)
					fixture.task.ObjectiveID = &missingID
				} else {
					fixture.task.ObjectiveID = nil
				}
				ctx := context.Background()
				uow, err := fixture.store.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				item, err := uow.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
				if err != nil {
					t.Fatal(err)
				}
				item.ObjectiveID = fixture.task.ObjectiveID
				item.Version++
				if err := uow.WorkItems().Save(ctx, item, fixture.task.Version); err != nil {
					t.Fatal(err)
				}
				if err := uow.Commit(); err != nil {
					t.Fatal(err)
				}
				withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
					item, err := read.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
					if err != nil {
						t.Fatal(err)
					}
					readiness, err := evaluateTaskAcquisition(ctx, read, item, operationalGateNow)
					if err != nil {
						t.Fatal(err)
					}
					expected := domain.ReadinessReasonTaskObjectiveRequired
					if missingOwner {
						expected = domain.ReadinessReasonObjectiveAncestryMissing
					}
					if !hasGateReason(readiness, expected) {
						t.Fatalf("expected %s, got %v", expected, readiness.Reasons)
					}
				})
			})
		}
	})
}

func TestOperationalTaskGateRequiresActiveNonArchivedOutcome(t *testing.T) {
	for _, test := range []struct {
		name       string
		archive    bool
		wantReason domain.ReadinessReason
	}{
		{name: "inactive outcome", wantReason: domain.ReadinessReasonOutcomeNotActive},
		{name: "archived outcome", archive: true, wantReason: domain.ReadinessReasonOutcomeArchived},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
			ctx := context.Background()
			uow, err := fixture.store.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := uow.Outcomes().Get(ctx, fixture.scope.NamespaceID, fixture.scope.OutcomeID)
			if err != nil {
				t.Fatal(err)
			}
			if test.archive {
				if err := outcome.Archive(operationalGateNow.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
			} else {
				outcome.Lifecycle = domain.OutcomeLifecycleDraft
				outcome.Version++
				outcome.UpdatedAt = operationalGateNow.Add(time.Minute)
			}
			if err := uow.Outcomes().Save(ctx, outcome, fixture.outcome.Version); err != nil {
				t.Fatal(err)
			}
			if err := uow.Commit(); err != nil {
				t.Fatal(err)
			}
			withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
				item, err := read.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
				if err != nil {
					t.Fatal(err)
				}
				readiness, err := evaluateTaskAcquisition(ctx, read, item, operationalGateNow)
				if err != nil {
					t.Fatal(err)
				}
				if !hasGateReason(readiness, test.wantReason) {
					t.Fatalf("missing %s in reasons %v", test.wantReason, readiness.Reasons)
				}
			})
		})
	}
}

func TestOperationalTaskGateCorruptionBlockersDependenciesScheduleAndAuthority(t *testing.T) {
	t.Run("missing ancestor fails closed", func(t *testing.T) {
		fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
		missing := operationalGateID(999)
		fixture.root.ParentObjectiveID = &missing
		ctx := context.Background()
		uow, err := fixture.store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		root, err := uow.Objectives().Get(ctx, fixture.scope, fixture.root.ID)
		if err != nil {
			t.Fatal(err)
		}
		root.ParentObjectiveID = &missing
		root.Version++
		if err := uow.Objectives().Save(ctx, root, fixture.root.Version); err != nil {
			t.Fatal(err)
		}
		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}
		withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
			readiness, err := evaluateTaskAcquisition(ctx, read, fixture.task, operationalGateNow)
			if err != nil {
				t.Fatal(err)
			}
			if readiness.Ready || !hasGateReason(readiness, domain.ReadinessReasonObjectiveAncestryMissing) {
				t.Fatalf("missing parent did not fail closed: %+v", readiness)
			}
		})
	})

	t.Run("cycle is reported", func(t *testing.T) {
		fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
		ctx := context.Background()
		uow, err := fixture.store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		root, err := uow.Objectives().Get(ctx, fixture.scope, fixture.root.ID)
		if err != nil {
			t.Fatal(err)
		}
		root.ParentObjectiveID = &fixture.child.ID
		root.Version++
		if err := uow.Objectives().Save(ctx, root, fixture.root.Version); err != nil {
			t.Fatal(err)
		}
		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}
		withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
			readiness, err := evaluateTaskAcquisition(ctx, read, fixture.task, operationalGateNow)
			if err != nil {
				t.Fatal(err)
			}
			if readiness.Ready || !hasGateReason(readiness, domain.ReadinessReasonObjectiveAncestryCycle) {
				t.Fatalf("cycle did not fail closed: %+v", readiness)
			}
		})
	})

	t.Run("invalid parent id is reported as corrupt", func(t *testing.T) {
		fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
		ctx := context.Background()
		uow, err := fixture.store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		root, err := uow.Objectives().Get(ctx, fixture.scope, fixture.root.ID)
		if err != nil {
			t.Fatal(err)
		}
		invalid := domain.ID("not-an-id")
		root.ParentObjectiveID = &invalid
		root.Version++
		if err := uow.Objectives().Save(ctx, root, fixture.root.Version); err != nil {
			t.Fatal(err)
		}
		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}
		withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
			readiness, err := evaluateTaskAcquisition(ctx, read, fixture.task, operationalGateNow)
			if err != nil {
				t.Fatal(err)
			}
			if readiness.Ready || !hasGateReason(readiness, domain.ReadinessReasonObjectiveAncestryCorrupt) {
				t.Fatalf("corrupt parent did not fail closed: %+v", readiness)
			}
		})
	})

	t.Run("hard dependency blocks but advisory does not", func(t *testing.T) {
		for _, strength := range []domain.DependencyStrength{domain.DependencyStrengthHard, domain.DependencyStrengthAdvisory} {
			t.Run(string(strength), func(t *testing.T) {
				fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
				ctx := context.Background()
				target, err := domain.NewWorkItemForObjective(operationalGateID(40), fixture.scope, &fixture.child.ID, "Dependency target", "", domain.PriorityNormal, domain.WorkItemLifecycleBacklog, operationalGateNow)
				if err != nil {
					t.Fatal(err)
				}
				relation, err := domain.NewDependencyRelation(operationalGateID(41), fixture.task.Ref(), target.Ref(), strength, operationalGateNow)
				if err != nil {
					t.Fatal(err)
				}
				uow, err := fixture.store.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if err := uow.WorkItems().Insert(ctx, target); err != nil {
					t.Fatal(err)
				}
				if err := uow.Relations().Insert(ctx, relation); err != nil {
					t.Fatal(err)
				}
				if err := uow.Commit(); err != nil {
					t.Fatal(err)
				}
				withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
					item, err := read.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
					if err != nil {
						t.Fatal(err)
					}
					readiness, err := evaluateTaskAcquisition(ctx, read, item, operationalGateNow)
					if err != nil {
						t.Fatal(err)
					}
					blocked := hasGateReason(readiness, domain.ReadinessReasonHardDependencyUnsatisfied)
					if blocked != (strength == domain.DependencyStrengthHard) {
						t.Fatalf("strength %s blocked=%v readiness=%+v", strength, blocked, readiness)
					}
					if blocked && !containsEntityRef(readiness.RelatedRefs, target.Ref()) {
						t.Fatalf("hard dependency target absent from related refs: %+v", readiness.RelatedRefs)
					}
				})
			})
		}
	})

	t.Run("blocker and schedule are scope reasons", func(t *testing.T) {
		fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
		ctx := context.Background()
		blockedAt := operationalGateNow.Add(time.Hour)
		fixture.task.NotBefore = &blockedAt
		cause := &domain.ExternalCause{Provider: "test", ID: "external-1", Description: "waiting for external condition"}
		blocker, err := domain.NewBlocker(operationalGateID(50), fixture.task.Ref(), nil, cause, "External prerequisite", domain.BlockerPropagationDirect, operationalGateNow)
		if err != nil {
			t.Fatal(err)
		}
		uow, err := fixture.store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		item, err := uow.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
		if err != nil {
			t.Fatal(err)
		}
		item.NotBefore = &blockedAt
		item.Version++
		if err := uow.WorkItems().Save(ctx, item, fixture.task.Version); err != nil {
			t.Fatal(err)
		}
		if err := uow.(ports.IssueBlockerUnitOfWork).Blockers().Insert(ctx, blocker); err != nil {
			t.Fatal(err)
		}
		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}
		withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
			item, err := read.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
			if err != nil {
				t.Fatal(err)
			}
			readiness, err := evaluateTaskAcquisition(ctx, read, item, operationalGateNow)
			if err != nil {
				t.Fatal(err)
			}
			if !hasGateReason(readiness, domain.ReadinessReasonNotBefore) || !hasGateReason(readiness, domain.ReadinessReasonBlocked) {
				t.Fatalf("expected both schedule and blocker reasons: %+v", readiness)
			}
			if !containsEntityRef(readiness.RelatedRefs, blocker.Ref()) {
				t.Fatalf("blocker was not linked as a same-scope related ref: %+v", readiness.RelatedRefs)
			}
		})
	})

	t.Run("active lease is authority availability, not definition", func(t *testing.T) {
		fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
		ctx := context.Background()
		uow, err := fixture.store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		item, err := uow.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := item.Claim(operationalGateID(60), "worker", domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "worker"}, time.Minute, operationalGateNow); err != nil {
			t.Fatal(err)
		}
		if err := uow.WorkItems().Save(ctx, item, fixture.task.Version); err != nil {
			t.Fatal(err)
		}
		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}
		withOperationalGateRead(t, fixture.store, func(read ports.UnitOfWork) {
			item, err := read.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
			if err != nil {
				t.Fatal(err)
			}
			readiness, err := evaluateTaskAcquisition(ctx, read, item, operationalGateNow)
			if err != nil {
				t.Fatal(err)
			}
			if !hasGateReason(readiness, domain.ReadinessReasonActiveLease) || readiness.Authority.Ready {
				t.Fatalf("active authority was not isolated: %+v", readiness)
			}
			if !readiness.Definition.Ready {
				t.Fatal("active authority incorrectly invalidated the Task definition")
			}
		})
	})
}

func containsEntityRef(values []domain.EntityRef, target domain.EntityRef) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type operationalGateOutcomeOverlay struct {
	ports.UnitOfWork
	outcome domain.Outcome
}

func (u operationalGateOutcomeOverlay) Outcomes() ports.OutcomeRepository {
	return operationalGateOutcomeRepository{OutcomeRepository: u.UnitOfWork.Outcomes(), outcome: u.outcome}
}

type operationalGateOutcomeRepository struct {
	ports.OutcomeRepository
	outcome domain.Outcome
}

func (r operationalGateOutcomeRepository) Get(context.Context, domain.ID, domain.ID) (domain.Outcome, error) {
	return r.outcome, nil
}

func TestOperationalGateRejectsUnknownProfile(t *testing.T) {
	fixture := newOperationalGateFixture(t, domain.OperationalProfileCompatibilityV1, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
	uow, err := fixture.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	invalid := fixture.outcome
	invalid.OperationalProfile = domain.OperationalProfile("future_profile")
	readiness, err := evaluateTaskAcquisition(context.Background(), operationalGateOutcomeOverlay{UnitOfWork: uow, outcome: invalid}, fixture.task, operationalGateNow)
	if err == nil {
		t.Fatal("unknown profile was accepted")
	}
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil || typed.Details.ReasonCode != string(domain.ReadinessReasonOperationalProfileUnknown) {
		t.Fatalf("unknown profile error lacks stable reason: %#v", err)
	}
	if readiness.Ready {
		t.Fatal("unknown profile returned ready work")
	}
}
