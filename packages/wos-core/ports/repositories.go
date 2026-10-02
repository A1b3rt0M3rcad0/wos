package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type OutcomeRepository interface {
	Get(ctx context.Context, namespaceID, outcomeID domain.ID) (domain.Outcome, error)
	Insert(ctx context.Context, outcome domain.Outcome) error
	Save(ctx context.Context, outcome domain.Outcome, expected domain.Version) error
}

type ObjectiveRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Objective, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Objective, error)
	Insert(ctx context.Context, objective domain.Objective) error
	Save(ctx context.Context, objective domain.Objective, expected domain.Version) error
}

type WorkItemRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.WorkItem, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.WorkItem, error)
	Insert(ctx context.Context, item domain.WorkItem) error
	Save(ctx context.Context, item domain.WorkItem, expected domain.Version) error
}

type RelationRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Relation, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Relation, error)
	Insert(ctx context.Context, relation domain.Relation) error
	Save(ctx context.Context, relation domain.Relation, expected domain.Version) error
}

type IssueRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Issue, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Issue, error)
	Insert(ctx context.Context, issue domain.Issue) error
	Save(ctx context.Context, issue domain.Issue, expected domain.Version) error
}

type BlockerRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Blocker, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Blocker, error)
	Insert(ctx context.Context, blocker domain.Blocker) error
	Save(ctx context.Context, blocker domain.Blocker, expected domain.Version) error
}


type ArtifactRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Artifact, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Artifact, error)
	Insert(ctx context.Context, artifact domain.Artifact) error
	Save(ctx context.Context, artifact domain.Artifact, expected domain.Version) error
}

type EvidenceRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Evidence, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Evidence, error)
	Insert(ctx context.Context, evidence domain.Evidence) error
	Save(ctx context.Context, evidence domain.Evidence, expected domain.Version) error
}

type EvidenceLinkRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.EvidenceLink, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.EvidenceLink, error)
	Insert(ctx context.Context, link domain.EvidenceLink) error
	Save(ctx context.Context, link domain.EvidenceLink, expected domain.Version) error
}

type DecisionRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Decision, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Decision, error)
	Insert(ctx context.Context, decision domain.Decision) error
	Save(ctx context.Context, decision domain.Decision, expected domain.Version) error
}
