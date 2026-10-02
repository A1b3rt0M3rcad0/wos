package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type Store struct {
	mu          sync.Mutex
	outcomes    map[string]domain.Outcome
	objectives  map[string]domain.Objective
	workItems   map[string]domain.WorkItem
	revisions   map[string]domain.OutcomeRevision
	events      []domain.DomainEvent
	idempotency map[string]idempotencyRecord
}

type idempotencyRecord struct {
	Identity    domain.IdempotencyIdentity
	Fingerprint string
	Completed   bool
	Result      domain.StoredCommandResult
}

func New() *Store {
	return &Store{
		outcomes:    make(map[string]domain.Outcome),
		objectives:  make(map[string]domain.Objective),
		workItems:   make(map[string]domain.WorkItem),
		revisions:   make(map[string]domain.OutcomeRevision),
		events:      make([]domain.DomainEvent, 0),
		idempotency: make(map[string]idempotencyRecord),
	}
}

func (s *Store) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, err
	}

	tx := &transaction{
		store:       s,
		outcomes:    cloneOutcomes(s.outcomes),
		objectives:  cloneObjectives(s.objectives),
		workItems:   cloneWorkItems(s.workItems),
		revisions:   cloneRevisions(s.revisions),
		events:      cloneEvents(s.events),
		idempotency: cloneIdempotency(s.idempotency),
	}
	tx.outcomeRepo = outcomeRepository{tx: tx}
	tx.objectiveRepo = objectiveRepository{tx: tx}
	tx.workItemRepo = workItemRepository{tx: tx}
	tx.coordination = coordinationStore{tx: tx}
	tx.eventLog = eventLog{tx: tx}
	tx.idempotencyStore = idempotencyStore{tx: tx}
	return tx, nil
}

type transaction struct {
	store       *Store
	closed      bool
	outcomes    map[string]domain.Outcome
	objectives  map[string]domain.Objective
	workItems   map[string]domain.WorkItem
	revisions   map[string]domain.OutcomeRevision
	events      []domain.DomainEvent
	idempotency map[string]idempotencyRecord

	outcomeRepo       outcomeRepository
	objectiveRepo     objectiveRepository
	workItemRepo      workItemRepository
	coordination      coordinationStore
	eventLog          eventLog
	idempotencyStore  idempotencyStore
}

func (tx *transaction) Outcomes() ports.OutcomeRepository     { return tx.outcomeRepo }
func (tx *transaction) Objectives() ports.ObjectiveRepository { return tx.objectiveRepo }
func (tx *transaction) WorkItems() ports.WorkItemRepository   { return tx.workItemRepo }
func (tx *transaction) Coordination() ports.CoordinationStore { return tx.coordination }
func (tx *transaction) Events() ports.DomainEventLog           { return tx.eventLog }
func (tx *transaction) Idempotency() ports.IdempotencyStore    { return tx.idempotencyStore }

func (tx *transaction) Commit() error {
	if tx.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction already closed")
	}
	tx.store.outcomes = cloneOutcomes(tx.outcomes)
	tx.store.objectives = cloneObjectives(tx.objectives)
	tx.store.workItems = cloneWorkItems(tx.workItems)
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
