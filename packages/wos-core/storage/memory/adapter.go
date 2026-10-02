package memory

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type Store struct {
	mu            sync.Mutex
	outcomes      map[string]domain.Outcome
	objectives    map[string]domain.Objective
	workItems     map[string]domain.WorkItem
	relations     map[string]domain.Relation
	issues        map[string]domain.Issue
	blockers      map[string]domain.Blocker
	artifacts     map[string]domain.Artifact
	evidence      map[string]domain.Evidence
	evidenceLinks map[string]domain.EvidenceLink
	decisions     map[string]domain.Decision
	roadmaps      map[string]domain.Roadmap
	revisions     map[string]domain.OutcomeRevision
	events        []domain.DomainEvent
	idempotency   map[string]idempotencyRecord
}

type idempotencyRecord struct {
	Identity    domain.IdempotencyIdentity
	Fingerprint string
	Completed   bool
	Result      domain.StoredCommandResult
}

func New() *Store {
	return &Store{
		outcomes:      make(map[string]domain.Outcome),
		objectives:    make(map[string]domain.Objective),
		workItems:     make(map[string]domain.WorkItem),
		relations:     make(map[string]domain.Relation),
		issues:        make(map[string]domain.Issue),
		blockers:      make(map[string]domain.Blocker),
		artifacts:     make(map[string]domain.Artifact),
		evidence:      make(map[string]domain.Evidence),
		evidenceLinks: make(map[string]domain.EvidenceLink),
		decisions:     make(map[string]domain.Decision),
		roadmaps:      make(map[string]domain.Roadmap),
		revisions:     make(map[string]domain.OutcomeRevision),
		events:        make([]domain.DomainEvent, 0),
		idempotency:   make(map[string]idempotencyRecord),
	}
}

func (s *Store) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, err
	}

	tx := &transaction{
		store:         s,
		outcomes:      cloneOutcomes(s.outcomes),
		objectives:    cloneObjectives(s.objectives),
		workItems:     cloneWorkItems(s.workItems),
		relations:     cloneRelations(s.relations),
		issues:        cloneIssues(s.issues),
		blockers:      cloneBlockers(s.blockers),
		artifacts:     cloneArtifacts(s.artifacts),
		evidence:      cloneEvidence(s.evidence),
		evidenceLinks: cloneEvidenceLinks(s.evidenceLinks),
		decisions:     cloneDecisions(s.decisions),
		roadmaps:      cloneRoadmaps(s.roadmaps),
		revisions:     cloneRevisions(s.revisions),
		events:        cloneEvents(s.events),
		idempotency:   cloneIdempotency(s.idempotency),
	}
	tx.outcomeRepo = outcomeRepository{tx: tx}
	tx.objectiveRepo = objectiveRepository{tx: tx}
	tx.workItemRepo = workItemRepository{tx: tx}
	tx.relationRepo = relationRepository{tx: tx}
	tx.issueRepo = issueRepository{tx: tx}
	tx.blockerRepo = blockerRepository{tx: tx}
	tx.artifactRepo = artifactRepository{tx: tx}
	tx.evidenceRepo = evidenceRepository{tx: tx}
	tx.evidenceLinkRepo = evidenceLinkRepository{tx: tx}
	tx.decisionRepo = decisionRepository{tx: tx}
	tx.roadmapRepo = roadmapRepository{tx: tx}
	tx.coordination = coordinationStore{tx: tx}
	tx.eventLog = eventLog{tx: tx}
	tx.idempotencyStore = idempotencyStore{tx: tx}
	return tx, nil
}

type transaction struct {
	store         *Store
	closed        bool
	outcomes      map[string]domain.Outcome
	objectives    map[string]domain.Objective
	workItems     map[string]domain.WorkItem
	relations     map[string]domain.Relation
	issues        map[string]domain.Issue
	blockers      map[string]domain.Blocker
	artifacts     map[string]domain.Artifact
	evidence      map[string]domain.Evidence
	evidenceLinks map[string]domain.EvidenceLink
	decisions     map[string]domain.Decision
	roadmaps      map[string]domain.Roadmap
	revisions     map[string]domain.OutcomeRevision
	events        []domain.DomainEvent
	idempotency   map[string]idempotencyRecord

	outcomeRepo      outcomeRepository
	objectiveRepo    objectiveRepository
	workItemRepo     workItemRepository
	relationRepo     relationRepository
	issueRepo        issueRepository
	blockerRepo      blockerRepository
	artifactRepo     artifactRepository
	evidenceRepo     evidenceRepository
	evidenceLinkRepo evidenceLinkRepository
	decisionRepo     decisionRepository
	roadmapRepo      roadmapRepository
	coordination     coordinationStore
	eventLog         eventLog
	idempotencyStore idempotencyStore
}

func (tx *transaction) Outcomes() ports.OutcomeRepository           { return tx.outcomeRepo }
func (tx *transaction) Objectives() ports.ObjectiveRepository       { return tx.objectiveRepo }
func (tx *transaction) WorkItems() ports.WorkItemRepository         { return tx.workItemRepo }
func (tx *transaction) Relations() ports.RelationRepository         { return tx.relationRepo }
func (tx *transaction) Issues() ports.IssueRepository               { return tx.issueRepo }
func (tx *transaction) Blockers() ports.BlockerRepository           { return tx.blockerRepo }
func (tx *transaction) Artifacts() ports.ArtifactRepository         { return tx.artifactRepo }
func (tx *transaction) Evidence() ports.EvidenceRepository          { return tx.evidenceRepo }
func (tx *transaction) EvidenceLinks() ports.EvidenceLinkRepository { return tx.evidenceLinkRepo }
func (tx *transaction) Decisions() ports.DecisionRepository         { return tx.decisionRepo }
func (tx *transaction) Roadmaps() ports.RoadmapRepository           { return tx.roadmapRepo }
func (tx *transaction) Coordination() ports.CoordinationStore       { return tx.coordination }
func (tx *transaction) Events() ports.DomainEventLog                { return tx.eventLog }
func (tx *transaction) Idempotency() ports.IdempotencyStore         { return tx.idempotencyStore }

func (tx *transaction) Commit() error {
	if tx.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction already closed")
	}
	tx.store.outcomes = cloneOutcomes(tx.outcomes)
	tx.store.objectives = cloneObjectives(tx.objectives)
	tx.store.workItems = cloneWorkItems(tx.workItems)
	tx.store.relations = cloneRelations(tx.relations)
	tx.store.issues = cloneIssues(tx.issues)
	tx.store.blockers = cloneBlockers(tx.blockers)
	tx.store.artifacts = cloneArtifacts(tx.artifacts)
	tx.store.evidence = cloneEvidence(tx.evidence)
	tx.store.evidenceLinks = cloneEvidenceLinks(tx.evidenceLinks)
	tx.store.decisions = cloneDecisions(tx.decisions)
	tx.store.roadmaps = cloneRoadmaps(tx.roadmaps)
	tx.store.revisions = cloneRevisions(tx.revisions)
	tx.store.events = cloneEvents(tx.events)
	tx.store.idempotency = cloneIdempotency(tx.idempotency)
	tx.closed = true
	tx.store.mu.Unlock()
	return nil
}

func (tx *transaction) Rollback() error {
	if tx.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction already closed")
	}
	tx.closed = true
	tx.store.mu.Unlock()
	return nil
}

func (tx *transaction) ensureOpen() error {
	if tx.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction is closed")
	}
	return nil
}

type roadmapRepository struct{ tx *transaction }

func (r roadmapRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Roadmap, error) {
	if err := ctx.Err(); err != nil {
		return domain.Roadmap{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Roadmap{}, err
	}
	value, ok := r.tx.roadmaps[entityKey(scope, id)]
	if !ok {
		return domain.Roadmap{}, domain.NewError(domain.ErrorCodeNotFound, "roadmap not found")
	}
	return cloneRoadmap(value), nil
}

func (r roadmapRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Roadmap, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	result := make([]domain.Roadmap, 0)
	for _, value := range r.tx.roadmaps {
		if value.Scope == scope {
			result = append(result, cloneRoadmap(value))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID.String() < result[j].ID.String()
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r roadmapRepository) Insert(ctx context.Context, roadmap domain.Roadmap) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := roadmap.Validate(); err != nil {
		return err
	}
	key := entityKey(roadmap.Scope, roadmap.ID)
	if _, exists := r.tx.roadmaps[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "roadmap already exists")
	}
	r.tx.roadmaps[key] = cloneRoadmap(roadmap)
	return nil
}

func (r roadmapRepository) Save(ctx context.Context, roadmap domain.Roadmap, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := roadmap.Validate(); err != nil {
		return err
	}
	key := entityKey(roadmap.Scope, roadmap.ID)
	current, ok := r.tx.roadmaps[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "roadmap not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "roadmap expected_version does not match")
	}
	if roadmap.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "roadmap version must advance exactly once per save")
	}
	if len(roadmap.Revisions) < len(current.Revisions) {
		return domain.NewError(domain.ErrorCodeRoadmap, "published roadmap revision history is append-only")
	}
	for i := range current.Revisions {
		if !reflect.DeepEqual(current.Revisions[i], roadmap.Revisions[i]) {
			return domain.NewError(domain.ErrorCodeRoadmap, "published roadmap revision is immutable")
		}
	}
	r.tx.roadmaps[key] = cloneRoadmap(roadmap)
	return nil
}

type outcomeRepository struct{ tx *transaction }

func (r outcomeRepository) Get(ctx context.Context, namespaceID, outcomeID domain.ID) (domain.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return domain.Outcome{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Outcome{}, err
	}
	value, ok := r.tx.outcomes[outcomeKey(namespaceID, outcomeID)]
	if !ok {
		return domain.Outcome{}, domain.NewError(domain.ErrorCodeNotFound, "outcome not found")
	}
	return cloneOutcome(value), nil
}

func (r outcomeRepository) Insert(ctx context.Context, outcome domain.Outcome) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := outcome.Validate(); err != nil {
		return err
	}
	key := outcomeKey(outcome.NamespaceID, outcome.ID)
	if _, exists := r.tx.outcomes[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "outcome already exists")
	}
	r.tx.outcomes[key] = cloneOutcome(outcome)
	return nil
}

func (r outcomeRepository) Save(ctx context.Context, outcome domain.Outcome, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := outcome.Validate(); err != nil {
		return err
	}
	key := outcomeKey(outcome.NamespaceID, outcome.ID)
	current, ok := r.tx.outcomes[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "outcome not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "outcome expected_version does not match")
	}
	if outcome.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "outcome version must advance exactly once per save")
	}
	if err := validateOwnedValidationHistoryAppendOnly(
		current.Criteria,
		outcome.Criteria,
		current.CurrentConclusion,
		current.ConclusionHistory,
		outcome.CurrentConclusion,
		outcome.ConclusionHistory,
	); err != nil {
		return err
	}
	r.tx.outcomes[key] = cloneOutcome(outcome)
	return nil
}

type objectiveRepository struct{ tx *transaction }

func (r objectiveRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Objective, error) {
	if err := ctx.Err(); err != nil {
		return domain.Objective{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Objective{}, err
	}
	value, ok := r.tx.objectives[entityKey(scope, id)]
	if !ok {
		return domain.Objective{}, domain.NewError(domain.ErrorCodeNotFound, "objective not found")
	}
	return cloneObjective(value), nil
}

func (r objectiveRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Objective, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.Objective, 0)
	for _, item := range r.tx.objectives {
		if item.Scope == scope {
			items = append(items, cloneObjective(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r objectiveRepository) Insert(ctx context.Context, objective domain.Objective) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := objective.Validate(); err != nil {
		return err
	}
	key := entityKey(objective.Scope, objective.ID)
	if _, exists := r.tx.objectives[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "objective already exists")
	}
	r.tx.objectives[key] = cloneObjective(objective)
	return nil
}

func (r objectiveRepository) Save(ctx context.Context, objective domain.Objective, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := objective.Validate(); err != nil {
		return err
	}
	key := entityKey(objective.Scope, objective.ID)
	current, ok := r.tx.objectives[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "objective not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "objective expected_version does not match")
	}
	if objective.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "objective version must advance exactly once per save")
	}
	if err := validateOwnedValidationHistoryAppendOnly(
		current.Criteria,
		objective.Criteria,
		current.CurrentConclusion,
		current.ConclusionHistory,
		objective.CurrentConclusion,
		objective.ConclusionHistory,
	); err != nil {
		return err
	}
	r.tx.objectives[key] = cloneObjective(objective)
	return nil
}

type workItemRepository struct{ tx *transaction }

func (r workItemRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.WorkItem, error) {
	if err := ctx.Err(); err != nil {
		return domain.WorkItem{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.WorkItem{}, err
	}
	value, ok := r.tx.workItems[entityKey(scope, id)]
	if !ok {
		return domain.WorkItem{}, domain.NewError(domain.ErrorCodeNotFound, "work item not found")
	}
	return cloneWorkItem(value), nil
}

func (r workItemRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.WorkItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.WorkItem, 0)
	for _, item := range r.tx.workItems {
		if item.Scope == scope {
			items = append(items, cloneWorkItem(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r workItemRepository) Insert(ctx context.Context, item domain.WorkItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := item.Validate(); err != nil {
		return err
	}
	key := entityKey(item.Scope, item.ID)
	if _, exists := r.tx.workItems[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "work item already exists")
	}
	r.tx.workItems[key] = cloneWorkItem(item)
	return nil
}

func (r workItemRepository) Save(ctx context.Context, item domain.WorkItem, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := item.Validate(); err != nil {
		return err
	}
	key := entityKey(item.Scope, item.ID)
	current, ok := r.tx.workItems[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "work item not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "work item expected_version does not match")
	}
	if item.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "work item version must advance exactly once per save")
	}
	if err := validateOwnedValidationHistoryAppendOnly(
		current.Criteria,
		item.Criteria,
		current.CurrentConclusion,
		current.ConclusionHistory,
		item.CurrentConclusion,
		item.ConclusionHistory,
	); err != nil {
		return err
	}
	r.tx.workItems[key] = cloneWorkItem(item)
	return nil
}

type relationRepository struct{ tx *transaction }

func (r relationRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Relation, error) {
	if err := ctx.Err(); err != nil {
		return domain.Relation{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Relation{}, err
	}
	value, ok := r.tx.relations[entityKey(scope, id)]
	if !ok {
		return domain.Relation{}, domain.NewError(domain.ErrorCodeNotFound, "relation not found")
	}
	return value, nil
}

func (r relationRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Relation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.Relation, 0)
	for _, item := range r.tx.relations {
		if item.Scope == scope {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r relationRepository) Insert(ctx context.Context, relation domain.Relation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := relation.Validate(); err != nil {
		return err
	}
	key := entityKey(relation.Scope, relation.ID)
	if _, exists := r.tx.relations[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "relation already exists")
	}
	for _, existing := range r.tx.relations {
		if existing.Scope == relation.Scope &&
			existing.Lifecycle == domain.RelationLifecycleActive &&
			relation.Lifecycle == domain.RelationLifecycleActive &&
			existing.SourceRef == relation.SourceRef &&
			existing.RelationType == relation.RelationType &&
			existing.TargetRef == relation.TargetRef {
			return domain.NewError(domain.ErrorCodeAlreadyExists, "active relation already exists")
		}
	}
	r.tx.relations[key] = relation
	return nil
}

func (r relationRepository) Save(ctx context.Context, relation domain.Relation, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := relation.Validate(); err != nil {
		return err
	}
	key := entityKey(relation.Scope, relation.ID)
	current, ok := r.tx.relations[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "relation not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "relation expected_version does not match")
	}
	if relation.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "relation version must advance exactly once per save")
	}
	r.tx.relations[key] = relation
	return nil
}

type issueRepository struct{ tx *transaction }

func (r issueRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Issue, error) {
	if err := ctx.Err(); err != nil {
		return domain.Issue{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Issue{}, err
	}
	value, ok := r.tx.issues[entityKey(scope, id)]
	if !ok {
		return domain.Issue{}, domain.NewError(domain.ErrorCodeNotFound, "issue not found")
	}
	return cloneIssue(value), nil
}

func (r issueRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Issue, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.Issue, 0)
	for _, item := range r.tx.issues {
		if item.Scope == scope {
			items = append(items, cloneIssue(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r issueRepository) Insert(ctx context.Context, issue domain.Issue) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := issue.Validate(); err != nil {
		return err
	}
	key := entityKey(issue.Scope, issue.ID)
	if _, exists := r.tx.issues[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "issue already exists")
	}
	r.tx.issues[key] = cloneIssue(issue)
	return nil
}

func (r issueRepository) Save(ctx context.Context, issue domain.Issue, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := issue.Validate(); err != nil {
		return err
	}
	key := entityKey(issue.Scope, issue.ID)
	current, ok := r.tx.issues[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "issue not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "issue expected_version does not match")
	}
	if issue.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "issue version must advance exactly once per save")
	}
	r.tx.issues[key] = cloneIssue(issue)
	return nil
}

type blockerRepository struct{ tx *transaction }

func (r blockerRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Blocker, error) {
	if err := ctx.Err(); err != nil {
		return domain.Blocker{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Blocker{}, err
	}
	value, ok := r.tx.blockers[entityKey(scope, id)]
	if !ok {
		return domain.Blocker{}, domain.NewError(domain.ErrorCodeNotFound, "blocker not found")
	}
	return cloneBlocker(value), nil
}

func (r blockerRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Blocker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.Blocker, 0)
	for _, item := range r.tx.blockers {
		if item.Scope == scope {
			items = append(items, cloneBlocker(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r blockerRepository) Insert(ctx context.Context, blocker domain.Blocker) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := blocker.Validate(); err != nil {
		return err
	}
	key := entityKey(blocker.Scope, blocker.ID)
	if _, exists := r.tx.blockers[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "blocker already exists")
	}
	r.tx.blockers[key] = cloneBlocker(blocker)
	return nil
}

func (r blockerRepository) Save(ctx context.Context, blocker domain.Blocker, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := blocker.Validate(); err != nil {
		return err
	}
	key := entityKey(blocker.Scope, blocker.ID)
	current, ok := r.tx.blockers[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "blocker not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "blocker expected_version does not match")
	}
	if blocker.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "blocker version must advance exactly once per save")
	}
	r.tx.blockers[key] = cloneBlocker(blocker)
	return nil
}

type artifactRepository struct{ tx *transaction }

func (r artifactRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return domain.Artifact{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Artifact{}, err
	}
	value, ok := r.tx.artifacts[entityKey(scope, id)]
	if !ok {
		return domain.Artifact{}, domain.NewError(domain.ErrorCodeNotFound, "artifact not found")
	}
	return cloneArtifact(value), nil
}

func (r artifactRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.Artifact, 0)
	for _, item := range r.tx.artifacts {
		if item.Scope == scope {
			items = append(items, cloneArtifact(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r artifactRepository) Insert(ctx context.Context, value domain.Artifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, exists := r.tx.artifacts[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "artifact already exists")
	}
	r.tx.artifacts[key] = cloneArtifact(value)
	return nil
}

func (r artifactRepository) Save(ctx context.Context, value domain.Artifact, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.artifacts[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "artifact not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "artifact expected_version does not match")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "artifact version must advance exactly once per save")
	}
	if !sameArtifactContent(current, value) {
		return domain.NewError(domain.ErrorCodeArtifact, "artifact content is immutable")
	}
	r.tx.artifacts[key] = cloneArtifact(value)
	return nil
}

type evidenceRepository struct{ tx *transaction }

func (r evidenceRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return domain.Evidence{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Evidence{}, err
	}
	value, ok := r.tx.evidence[entityKey(scope, id)]
	if !ok {
		return domain.Evidence{}, domain.NewError(domain.ErrorCodeNotFound, "evidence not found")
	}
	return cloneEvidenceValue(value), nil
}

func (r evidenceRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.Evidence, 0)
	for _, item := range r.tx.evidence {
		if item.Scope == scope {
			items = append(items, cloneEvidenceValue(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r evidenceRepository) Insert(ctx context.Context, value domain.Evidence) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, exists := r.tx.evidence[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "evidence already exists")
	}
	r.tx.evidence[key] = cloneEvidenceValue(value)
	return nil
}

func (r evidenceRepository) Save(ctx context.Context, value domain.Evidence, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.evidence[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "evidence not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence expected_version does not match")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence version must advance exactly once per save")
	}
	if !sameEvidenceContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidence, "evidence content is immutable")
	}
	r.tx.evidence[key] = cloneEvidenceValue(value)
	return nil
}

type evidenceLinkRepository struct{ tx *transaction }

func (r evidenceLinkRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.EvidenceLink, error) {
	if err := ctx.Err(); err != nil {
		return domain.EvidenceLink{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.EvidenceLink{}, err
	}
	value, ok := r.tx.evidenceLinks[entityKey(scope, id)]
	if !ok {
		return domain.EvidenceLink{}, domain.NewError(domain.ErrorCodeNotFound, "evidence link not found")
	}
	return cloneEvidenceLink(value), nil
}

func (r evidenceLinkRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.EvidenceLink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.EvidenceLink, 0)
	for _, item := range r.tx.evidenceLinks {
		if item.Scope == scope {
			items = append(items, cloneEvidenceLink(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r evidenceLinkRepository) Insert(ctx context.Context, value domain.EvidenceLink) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, exists := r.tx.evidenceLinks[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "evidence link already exists")
	}
	r.tx.evidenceLinks[key] = cloneEvidenceLink(value)
	return nil
}

func (r evidenceLinkRepository) Save(ctx context.Context, value domain.EvidenceLink, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.evidenceLinks[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "evidence link not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence link expected_version does not match")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence link version must advance exactly once per save")
	}
	if !sameEvidenceLinkContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidenceLink, "evidence link content is immutable")
	}
	r.tx.evidenceLinks[key] = cloneEvidenceLink(value)
	return nil
}

type decisionRepository struct{ tx *transaction }

func (r decisionRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Decision, error) {
	if err := ctx.Err(); err != nil {
		return domain.Decision{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Decision{}, err
	}
	value, ok := r.tx.decisions[entityKey(scope, id)]
	if !ok {
		return domain.Decision{}, domain.NewError(domain.ErrorCodeNotFound, "decision not found")
	}
	return cloneDecision(value), nil
}

func (r decisionRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Decision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	items := make([]domain.Decision, 0)
	for _, item := range r.tx.decisions {
		if item.Scope == scope {
			items = append(items, cloneDecision(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (r decisionRepository) Insert(ctx context.Context, value domain.Decision) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, exists := r.tx.decisions[key]; exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "decision already exists")
	}
	r.tx.decisions[key] = cloneDecision(value)
	return nil
}

func (r decisionRepository) Save(ctx context.Context, value domain.Decision, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.decisions[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "decision not found")
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "decision expected_version does not match")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "decision version must advance exactly once per save")
	}
	if current.Lifecycle != domain.DecisionLifecycleProposed && !sameDecisionContent(current, value) {
		return domain.NewError(domain.ErrorCodeDecision, "accepted/rejected/superseded decision content is immutable")
	}
	r.tx.decisions[key] = cloneDecision(value)
	return nil
}

type coordinationStore struct{ tx *transaction }

func (s coordinationStore) LockOutcome(ctx context.Context, scope domain.Scope) (ports.OutcomeCoordination, error) {
	if err := ctx.Err(); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	if err := s.tx.ensureOpen(); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	if err := scope.Validate(); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	return ports.OutcomeCoordination{Scope: scope, Revision: s.tx.revisions[scopeKey(scope)]}, nil
}

func (s coordinationStore) AdvanceOutcome(ctx context.Context, scope domain.Scope) (domain.OutcomeRevision, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := s.tx.ensureOpen(); err != nil {
		return 0, err
	}
	current := s.tx.revisions[scopeKey(scope)]
	next, err := current.Next()
	if err != nil {
		return 0, err
	}
	s.tx.revisions[scopeKey(scope)] = next
	return next, nil
}

type eventLog struct{ tx *transaction }

func (l eventLog) Append(ctx context.Context, events []domain.DomainEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := l.tx.ensureOpen(); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	for i, event := range events {
		if err := event.Validate(); err != nil {
			return err
		}
		if i > 0 {
			prev := events[i-1]
			if event.Scope() != prev.Scope() || event.OutcomeRevision != prev.OutcomeRevision {
				return domain.NewError(domain.ErrorCodeInvalidEvent, "events in one append must share scope and outcome_revision")
			}
			if event.EventIndex != prev.EventIndex+1 {
				return domain.NewError(domain.ErrorCodeInvalidEvent, "event_index must be contiguous within one command")
			}
		}
		for _, existing := range l.tx.events {
			if existing.EventID == event.EventID {
				return domain.NewError(domain.ErrorCodeAlreadyExists, "domain event already exists")
			}
			if existing.Scope() == event.Scope() &&
				existing.OutcomeRevision == event.OutcomeRevision &&
				existing.EventIndex == event.EventIndex {
				return domain.NewError(domain.ErrorCodeAlreadyExists, "domain event ordering slot already exists")
			}
		}
		l.tx.events = append(l.tx.events, cloneEvent(event))
	}
	return nil
}

type idempotencyStore struct{ tx *transaction }

func (s idempotencyStore) Reserve(ctx context.Context, identity domain.IdempotencyIdentity, fingerprint string) (domain.IdempotencyReservation, error) {
	if err := ctx.Err(); err != nil {
		return domain.IdempotencyReservation{}, err
	}
	if err := s.tx.ensureOpen(); err != nil {
		return domain.IdempotencyReservation{}, err
	}
	if err := identity.Validate(); err != nil {
		return domain.IdempotencyReservation{}, err
	}
	if fingerprint == "" {
		return domain.IdempotencyReservation{}, domain.NewError(domain.ErrorCodeIdempotencyState, "fingerprint is required")
	}

	key := idempotencyKey(identity)
	if existing, ok := s.tx.idempotency[key]; ok {
		if existing.Fingerprint != fingerprint {
			return domain.IdempotencyReservation{}, domain.NewError(domain.ErrorCodeIdempotencyConflict, "idempotency key was already used with a different command fingerprint")
		}
		reservation := domain.IdempotencyReservation{Identity: identity, Fingerprint: fingerprint}
		if existing.Completed {
			result := cloneStoredCommandResult(existing.Result)
			reservation.Replay = &result
			return reservation, nil
		}
		return domain.IdempotencyReservation{}, domain.NewError(domain.ErrorCodeIdempotencyState, "idempotency reservation is still processing")
	}

	s.tx.idempotency[key] = idempotencyRecord{
		Identity:    identity,
		Fingerprint: fingerprint,
	}
	return domain.IdempotencyReservation{Identity: identity, Fingerprint: fingerprint}, nil
}

func (s idempotencyStore) Complete(ctx context.Context, reservation domain.IdempotencyReservation, result domain.StoredCommandResult) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.tx.ensureOpen(); err != nil {
		return err
	}
	if reservation.IsReplay() {
		return domain.NewError(domain.ErrorCodeIdempotencyState, "replay reservation cannot be completed again")
	}
	if err := reservation.Identity.Validate(); err != nil {
		return err
	}
	if err := result.Validate(); err != nil {
		return err
	}
	key := idempotencyKey(reservation.Identity)
	record, ok := s.tx.idempotency[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeIdempotencyState, "idempotency reservation does not exist")
	}
	if record.Fingerprint != reservation.Fingerprint {
		return domain.NewError(domain.ErrorCodeIdempotencyConflict, "reservation fingerprint changed")
	}
	if record.Completed {
		return domain.NewError(domain.ErrorCodeIdempotencyState, "idempotency result already completed")
	}
	record.Completed = true
	record.Result = cloneStoredCommandResult(result)
	s.tx.idempotency[key] = record
	return nil
}

func (s *Store) SnapshotDomainEvents(scope domain.Scope) []domain.DomainEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]domain.DomainEvent, 0)
	for _, event := range s.events {
		if event.Scope() == scope {
			result = append(result, cloneEvent(event))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].OutcomeRevision == result[j].OutcomeRevision {
			return result[i].EventIndex < result[j].EventIndex
		}
		return result[i].OutcomeRevision < result[j].OutcomeRevision
	})
	return result
}

func (s *Store) IdempotencyRecordCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.idempotency)
}

func idempotencyKey(identity domain.IdempotencyIdentity) string {
	return identity.NamespaceID.String() + "/" + identity.PrincipalID + "/" + identity.CommandName + "/" + identity.IdempotencyKey
}

func outcomeKey(namespaceID, outcomeID domain.ID) string {
	return namespaceID.String() + "/" + outcomeID.String()
}

func scopeKey(scope domain.Scope) string {
	return outcomeKey(scope.NamespaceID, scope.OutcomeID)
}

func entityKey(scope domain.Scope, id domain.ID) string {
	return scopeKey(scope) + "/" + id.String()
}

func cloneRoadmaps(src map[string]domain.Roadmap) map[string]domain.Roadmap {
	dst := make(map[string]domain.Roadmap, len(src))
	for k, v := range src {
		dst[k] = cloneRoadmap(v)
	}
	return dst
}

func cloneRoadmap(v domain.Roadmap) domain.Roadmap {
	if v.ArchivedAt != nil {
		value := *v.ArchivedAt
		v.ArchivedAt = &value
	}
	if v.Draft != nil {
		draft := *v.Draft
		if v.Draft.BaseRevisionNumber != nil {
			value := *v.Draft.BaseRevisionNumber
			draft.BaseRevisionNumber = &value
		}
		if v.Draft.DiscardedAt != nil {
			value := *v.Draft.DiscardedAt
			draft.DiscardedAt = &value
		}
		draft.Nodes = cloneRoadmapNodes(v.Draft.Nodes)
		draft.AfterLinks = append([]domain.RoadmapAfterLink(nil), v.Draft.AfterLinks...)
		v.Draft = &draft
	}
	sourceRevisions := v.Revisions
	v.Revisions = make([]domain.RoadmapRevision, len(sourceRevisions))
	for i := range sourceRevisions {
		v.Revisions[i] = sourceRevisions[i]
		v.Revisions[i].Nodes = cloneRoadmapNodes(sourceRevisions[i].Nodes)
		v.Revisions[i].AfterLinks = append([]domain.RoadmapAfterLink(nil), sourceRevisions[i].AfterLinks...)
		v.Revisions[i].DependencySnapshots = append([]domain.RoadmapDependencySnapshot(nil), sourceRevisions[i].DependencySnapshots...)
	}
	return v
}

func cloneRoadmapNodes(src []domain.RoadmapNode) []domain.RoadmapNode {
	dst := make([]domain.RoadmapNode, len(src))
	for i := range src {
		dst[i] = src[i]
		dst[i].CriterionRefs = append([]domain.RoadmapCriterionRef(nil), src[i].CriterionRefs...)
		dst[i].CriterionSnapshots = append([]domain.RoadmapCriterionSnapshot(nil), src[i].CriterionSnapshots...)
		if src[i].TargetRef != nil {
			value := *src[i].TargetRef
			dst[i].TargetRef = &value
		}
		if src[i].PlannedStart != nil {
			value := *src[i].PlannedStart
			dst[i].PlannedStart = &value
		}
		if src[i].PlannedEnd != nil {
			value := *src[i].PlannedEnd
			dst[i].PlannedEnd = &value
		}
		if src[i].ReferenceSnapshot != nil {
			value := *src[i].ReferenceSnapshot
			dst[i].ReferenceSnapshot = &value
		}
	}
	return dst
}

func cloneRevisions(src map[string]domain.OutcomeRevision) map[string]domain.OutcomeRevision {
	dst := make(map[string]domain.OutcomeRevision, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneOutcomes(src map[string]domain.Outcome) map[string]domain.Outcome {
	dst := make(map[string]domain.Outcome, len(src))
	for k, v := range src {
		dst[k] = cloneOutcome(v)
	}
	return dst
}

func cloneObjectives(src map[string]domain.Objective) map[string]domain.Objective {
	dst := make(map[string]domain.Objective, len(src))
	for k, v := range src {
		dst[k] = cloneObjective(v)
	}
	return dst
}

func cloneWorkItems(src map[string]domain.WorkItem) map[string]domain.WorkItem {
	dst := make(map[string]domain.WorkItem, len(src))
	for k, v := range src {
		dst[k] = cloneWorkItem(v)
	}
	return dst
}

func cloneRelations(src map[string]domain.Relation) map[string]domain.Relation {
	dst := make(map[string]domain.Relation, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneIssues(src map[string]domain.Issue) map[string]domain.Issue {
	dst := make(map[string]domain.Issue, len(src))
	for k, v := range src {
		dst[k] = cloneIssue(v)
	}
	return dst
}

func cloneBlockers(src map[string]domain.Blocker) map[string]domain.Blocker {
	dst := make(map[string]domain.Blocker, len(src))
	for k, v := range src {
		dst[k] = cloneBlocker(v)
	}
	return dst
}

func sameArtifactContent(a, b domain.Artifact) bool {
	return a.ID == b.ID &&
		a.Scope == b.Scope &&
		a.ArtifactType == b.ArtifactType &&
		a.Name == b.Name &&
		a.URI == b.URI &&
		a.MediaType == b.MediaType &&
		a.Checksum == b.Checksum &&
		a.SourceVersion == b.SourceVersion &&
		a.ProducerRef == b.ProducerRef &&
		optionalTimeEqual(a.ProducedAt, b.ProducedAt) &&
		a.RegisteredAt.Equal(b.RegisteredAt)
}

func sameEvidenceContent(a, b domain.Evidence) bool {
	return a.ID == b.ID &&
		a.Scope == b.Scope &&
		a.EvidenceType == b.EvidenceType &&
		a.Description == b.Description &&
		a.SourceRef == b.SourceRef &&
		a.ProducerRef == b.ProducerRef &&
		a.CapturedAt.Equal(b.CapturedAt) &&
		a.RegisteredAt.Equal(b.RegisteredAt) &&
		optionalIDEqual(a.ArtifactID, b.ArtifactID) &&
		optionalMeasurementEqual(a.Measurement, b.Measurement) &&
		a.SourceVersion == b.SourceVersion &&
		a.Checksum == b.Checksum
}

func sameEvidenceLinkContent(a, b domain.EvidenceLink) bool {
	return a.ID == b.ID &&
		a.Scope == b.Scope &&
		a.EvidenceID == b.EvidenceID &&
		a.TargetRef == b.TargetRef &&
		optionalIDEqual(a.CriterionID, b.CriterionID) &&
		a.Stance == b.Stance &&
		a.Rationale == b.Rationale &&
		a.CreatedAt.Equal(b.CreatedAt)
}

func sameDecisionContent(a, b domain.Decision) bool {
	if a.ID != b.ID ||
		a.Scope != b.Scope ||
		a.Title != b.Title ||
		a.Proposal != b.Proposal ||
		a.ChosenAlternative != b.ChosenAlternative ||
		a.Rationale != b.Rationale ||
		a.ProposedBy != b.ProposedBy ||
		!optionalIDEqual(a.SupersedesDecisionID, b.SupersedesDecisionID) ||
		len(a.Alternatives) != len(b.Alternatives) {
		return false
	}
	for i := range a.Alternatives {
		if a.Alternatives[i] != b.Alternatives[i] {
			return false
		}
	}
	return true
}

func optionalIDEqual(a, b *domain.ID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func optionalTimeEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func optionalMeasurementEqual(a, b *domain.Measurement) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func validateOwnedValidationHistoryAppendOnly(
	currentCriteria domain.CriterionSet,
	nextCriteria domain.CriterionSet,
	currentConclusion *domain.Conclusion,
	currentHistory []domain.Conclusion,
	nextConclusion *domain.Conclusion,
	nextHistory []domain.Conclusion,
) error {
	if len(nextCriteria.DefinitionRevisions) < len(currentCriteria.DefinitionRevisions) {
		return domain.NewError(domain.ErrorCodeCriterion, "criterion revision history is append-only")
	}
	for i := range currentCriteria.DefinitionRevisions {
		if currentCriteria.DefinitionRevisions[i] != nextCriteria.DefinitionRevisions[i] {
			return domain.NewError(domain.ErrorCodeCriterion, "criterion definition revision is immutable")
		}
	}
	if len(nextCriteria.Assessments) < len(currentCriteria.Assessments) {
		return domain.NewError(domain.ErrorCodeAssessment, "criterion assessment history is append-only")
	}
	for i := range currentCriteria.Assessments {
		if !reflect.DeepEqual(currentCriteria.Assessments[i], nextCriteria.Assessments[i]) {
			return domain.NewError(domain.ErrorCodeAssessment, "criterion assessment is immutable")
		}
	}
	if len(nextHistory) < len(currentHistory) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "conclusion history is append-only")
	}
	for i := range currentHistory {
		if !reflect.DeepEqual(currentHistory[i], nextHistory[i]) {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "historical conclusion is immutable")
		}
	}

	if currentConclusion == nil {
		if len(nextHistory) != len(currentHistory) {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "conclusion history cannot grow without reopening a current conclusion")
		}
		return nil
	}

	if nextConclusion != nil {
		if !reflect.DeepEqual(*currentConclusion, *nextConclusion) {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "current conclusion cannot be replaced directly")
		}
		if len(nextHistory) != len(currentHistory) {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "conclusion history changed while current conclusion remained bound")
		}
		return nil
	}

	if len(nextHistory) != len(currentHistory)+1 ||
		!reflect.DeepEqual(*currentConclusion, nextHistory[len(currentHistory)]) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "reopening must append the current conclusion unchanged to history")
	}
	return nil
}

func cloneArtifacts(src map[string]domain.Artifact) map[string]domain.Artifact {
	dst := make(map[string]domain.Artifact, len(src))
	for k, v := range src {
		dst[k] = cloneArtifact(v)
	}
	return dst
}

func cloneEvidence(src map[string]domain.Evidence) map[string]domain.Evidence {
	dst := make(map[string]domain.Evidence, len(src))
	for k, v := range src {
		dst[k] = cloneEvidenceValue(v)
	}
	return dst
}

func cloneEvidenceLinks(src map[string]domain.EvidenceLink) map[string]domain.EvidenceLink {
	dst := make(map[string]domain.EvidenceLink, len(src))
	for k, v := range src {
		dst[k] = cloneEvidenceLink(v)
	}
	return dst
}

func cloneDecisions(src map[string]domain.Decision) map[string]domain.Decision {
	dst := make(map[string]domain.Decision, len(src))
	for k, v := range src {
		dst[k] = cloneDecision(v)
	}
	return dst
}

func cloneArtifact(v domain.Artifact) domain.Artifact {
	if v.ProducedAt != nil {
		value := *v.ProducedAt
		v.ProducedAt = &value
	}
	if v.WithdrawnAt != nil {
		value := *v.WithdrawnAt
		v.WithdrawnAt = &value
	}
	return v
}

func cloneEvidenceValue(v domain.Evidence) domain.Evidence {
	if v.ArtifactID != nil {
		value := *v.ArtifactID
		v.ArtifactID = &value
	}
	if v.Measurement != nil {
		value := *v.Measurement
		v.Measurement = &value
	}
	if v.RetractedAt != nil {
		value := *v.RetractedAt
		v.RetractedAt = &value
	}
	return v
}

func cloneEvidenceLink(v domain.EvidenceLink) domain.EvidenceLink {
	if v.CriterionID != nil {
		value := *v.CriterionID
		v.CriterionID = &value
	}
	if v.RetractedAt != nil {
		value := *v.RetractedAt
		v.RetractedAt = &value
	}
	return v
}

func cloneDecision(v domain.Decision) domain.Decision {
	v.Alternatives = append([]string(nil), v.Alternatives...)
	if v.DecidedBy != nil {
		value := *v.DecidedBy
		v.DecidedBy = &value
	}
	if v.DecidedAt != nil {
		value := *v.DecidedAt
		v.DecidedAt = &value
	}
	if v.SupersedesDecisionID != nil {
		value := *v.SupersedesDecisionID
		v.SupersedesDecisionID = &value
	}
	return v
}

func cloneIssue(v domain.Issue) domain.Issue {
	v.AffectedRefs = append([]domain.EntityRef(nil), v.AffectedRefs...)
	if v.DuplicateOfIssueID != nil {
		id := *v.DuplicateOfIssueID
		v.DuplicateOfIssueID = &id
	}
	return v
}

func cloneBlocker(v domain.Blocker) domain.Blocker {
	if v.CauseRef != nil {
		ref := *v.CauseRef
		v.CauseRef = &ref
	}
	if v.ExternalCause != nil {
		cause := *v.ExternalCause
		v.ExternalCause = &cause
	}
	if v.ResolvedAt != nil {
		value := *v.ResolvedAt
		v.ResolvedAt = &value
	}
	return v
}

func cloneOutcome(v domain.Outcome) domain.Outcome {
	v.OwnerRefs = append([]domain.ActorRef(nil), v.OwnerRefs...)
	v.Criteria = cloneCriteria(v.Criteria)
	v.CurrentConclusion = cloneConclusion(v.CurrentConclusion)
	v.ConclusionHistory = cloneConclusions(v.ConclusionHistory)
	if v.ArchivedAt != nil {
		t := *v.ArchivedAt
		v.ArchivedAt = &t
	}
	return v
}

func cloneObjective(v domain.Objective) domain.Objective {
	v.OwnerRefs = append([]domain.ActorRef(nil), v.OwnerRefs...)
	v.Criteria = cloneCriteria(v.Criteria)
	v.CurrentConclusion = cloneConclusion(v.CurrentConclusion)
	v.ConclusionHistory = cloneConclusions(v.ConclusionHistory)
	if v.ParentObjectiveID != nil {
		id := *v.ParentObjectiveID
		v.ParentObjectiveID = &id
	}
	return v
}

func cloneWorkItem(v domain.WorkItem) domain.WorkItem {
	v.AssigneeRefs = append([]domain.ActorRef(nil), v.AssigneeRefs...)
	v.Criteria = cloneCriteria(v.Criteria)
	v.CurrentConclusion = cloneConclusion(v.CurrentConclusion)
	v.ConclusionHistory = cloneConclusions(v.ConclusionHistory)
	if v.ObjectiveID != nil {
		id := *v.ObjectiveID
		v.ObjectiveID = &id
	}
	if v.NotBefore != nil {
		value := *v.NotBefore
		v.NotBefore = &value
	}
	if v.CurrentLease != nil {
		lease := *v.CurrentLease
		v.CurrentLease = &lease
	}
	return v
}

func cloneCriteria(v domain.CriterionSet) domain.CriterionSet {
	result := domain.CriterionSet{
		Items:               append([]domain.SuccessCriterion(nil), v.Items...),
		DefinitionRevisions: append([]domain.CriterionDefinitionRevision(nil), v.DefinitionRevisions...),
		Assessments:         make([]domain.CriterionAssessment, len(v.Assessments)),
		CurrentAssessments:  make(map[domain.ID]domain.CriterionAssessment, len(v.CurrentAssessments)),
	}
	for i, a := range v.Assessments {
		result.Assessments[i] = cloneAssessment(a)
	}
	for k, a := range v.CurrentAssessments {
		result.CurrentAssessments[k] = cloneAssessment(a)
	}
	return result
}

func cloneAssessment(v domain.CriterionAssessment) domain.CriterionAssessment {
	v.EvidenceIDs = append([]domain.ID(nil), v.EvidenceIDs...)
	if v.EvaluatorRef != nil {
		evaluator := *v.EvaluatorRef
		v.EvaluatorRef = &evaluator
	}
	if v.SupersedesAssessmentID != nil {
		id := *v.SupersedesAssessmentID
		v.SupersedesAssessmentID = &id
	}
	return v
}

func cloneConclusion(v *domain.Conclusion) *domain.Conclusion {
	if v == nil {
		return nil
	}
	c := *v
	c.Assessments = append([]domain.CriterionAssessmentRef(nil), v.Assessments...)
	c.Obligations.RequiredCriteria = append([]domain.CriterionObligationSnapshot(nil), v.Obligations.RequiredCriteria...)
	c.Obligations.RequiredObjectiveIDs = append([]domain.ID(nil), v.Obligations.RequiredObjectiveIDs...)
	if v.OwnerRef != nil {
		owner := *v.OwnerRef
		c.OwnerRef = &owner
	}
	if v.OwnerVersion != nil {
		version := *v.OwnerVersion
		c.OwnerVersion = &version
	}
	return &c
}

func cloneConclusions(src []domain.Conclusion) []domain.Conclusion {
	dst := make([]domain.Conclusion, len(src))
	for i := range src {
		cloned := cloneConclusion(&src[i])
		dst[i] = *cloned
	}
	return dst
}

func cloneEvents(src []domain.DomainEvent) []domain.DomainEvent {
	dst := make([]domain.DomainEvent, len(src))
	for i := range src {
		dst[i] = cloneEvent(src[i])
	}
	return dst
}

func cloneEvent(event domain.DomainEvent) domain.DomainEvent {
	event.Payload = append([]byte(nil), event.Payload...)
	if event.AggregateVersionBefore != nil {
		value := *event.AggregateVersionBefore
		event.AggregateVersionBefore = &value
	}
	if event.AggregateVersionAfter != nil {
		value := *event.AggregateVersionAfter
		event.AggregateVersionAfter = &value
	}
	if event.CausationID != nil {
		value := *event.CausationID
		event.CausationID = &value
	}
	return event
}

func cloneIdempotency(src map[string]idempotencyRecord) map[string]idempotencyRecord {
	dst := make(map[string]idempotencyRecord, len(src))
	for key, value := range src {
		value.Result = cloneStoredCommandResult(value.Result)
		dst[key] = value
	}
	return dst
}

func cloneStoredCommandResult(result domain.StoredCommandResult) domain.StoredCommandResult {
	result.ResponseJSON = append([]byte(nil), result.ResponseJSON...)
	return result
}
