package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
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
	decisions     map[string]domain.Decision
	evidenceLinks map[string]domain.EvidenceLink
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
		decisions:     make(map[string]domain.Decision),
		evidenceLinks: make(map[string]domain.EvidenceLink),
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
		decisions:     cloneDecisions(s.decisions),
		evidenceLinks: cloneEvidenceLinks(s.evidenceLinks),
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
	tx.decisionRepo = decisionRepository{tx: tx}
	tx.evidenceLinkRepo = evidenceLinkRepository{tx: tx}
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
	decisions     map[string]domain.Decision
	evidenceLinks map[string]domain.EvidenceLink
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
	decisionRepo     decisionRepository
	evidenceLinkRepo evidenceLinkRepository
	coordination     coordinationStore
	eventLog         eventLog
	idempotencyStore idempotencyStore
}

func (tx *transaction) Outcomes() ports.OutcomeRepository     { return tx.outcomeRepo }
func (tx *transaction) Objectives() ports.ObjectiveRepository { return tx.objectiveRepo }
func (tx *transaction) WorkItems() ports.WorkItemRepository   { return tx.workItemRepo }
func (tx *transaction) Relations() ports.RelationRepository   { return tx.relationRepo }
func (tx *transaction) Issues() ports.IssueRepository         { return tx.issueRepo }
func (tx *transaction) Blockers() ports.BlockerRepository     { return tx.blockerRepo }
func (tx *transaction) Artifacts() ports.ArtifactRepository   { return tx.artifactRepo }
func (tx *transaction) Evidence() ports.EvidenceRepository    { return tx.evidenceRepo }
func (tx *transaction) Decisions() ports.DecisionRepository   { return tx.decisionRepo }
func (tx *transaction) EvidenceLinks() ports.EvidenceLinkRepository {
	return tx.evidenceLinkRepo
}
func (tx *transaction) Coordination() ports.CoordinationStore { return tx.coordination }
func (tx *transaction) Events() ports.DomainEventLog          { return tx.eventLog }
func (tx *transaction) Idempotency() ports.IdempotencyStore   { return tx.idempotencyStore }

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
	tx.store.decisions = cloneDecisions(tx.decisions)
	tx.store.evidenceLinks = cloneEvidenceLinks(tx.evidenceLinks)
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

func cloneDecisions(src map[string]domain.Decision) map[string]domain.Decision {
	dst := make(map[string]domain.Decision, len(src))
	for k, v := range src {
		dst[k] = cloneDecision(v)
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

func cloneArtifact(v domain.Artifact) domain.Artifact {
	if v.ProducedAt != nil {
		value := *v.ProducedAt
		v.ProducedAt = &value
	}
	return v
}

func cloneEvidenceValue(v domain.Evidence) domain.Evidence {
	if v.ArtifactID != nil {
		id := *v.ArtifactID
		v.ArtifactID = &id
	}
	if v.Measurement != nil {
		measurement := *v.Measurement
		measurement.Value = append([]byte(nil), v.Measurement.Value...)
		measurement.Conditions = append([]byte(nil), v.Measurement.Conditions...)
		v.Measurement = &measurement
	}
	return v
}

func cloneDecision(v domain.Decision) domain.Decision {
	v.Alternatives = append([]string(nil), v.Alternatives...)
	if v.DecidedBy != nil {
		actor := *v.DecidedBy
		v.DecidedBy = &actor
	}
	if v.DecidedAt != nil {
		value := *v.DecidedAt
		v.DecidedAt = &value
	}
	if v.SupersedesDecisionID != nil {
		id := *v.SupersedesDecisionID
		v.SupersedesDecisionID = &id
	}
	return v
}

func cloneEvidenceLink(v domain.EvidenceLink) domain.EvidenceLink {
	if v.CriterionID != nil {
		id := *v.CriterionID
		v.CriterionID = &id
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
		Items:              append([]domain.SuccessCriterion(nil), v.Items...),
		Assessments:        make([]domain.CriterionAssessment, len(v.Assessments)),
		CurrentAssessments: make(map[domain.ID]domain.CriterionAssessment, len(v.CurrentAssessments)),
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
	return &c
}

func cloneConclusions(src []domain.Conclusion) []domain.Conclusion {
	dst := make([]domain.Conclusion, len(src))
	for i := range src {
		dst[i] = src[i]
		dst[i].Assessments = append([]domain.CriterionAssessmentRef(nil), src[i].Assessments...)
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
