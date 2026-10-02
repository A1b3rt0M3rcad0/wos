package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type Store struct {
	mu         sync.Mutex
	outcomes   map[string]domain.Outcome
	objectives map[string]domain.Objective
	workItems  map[string]domain.WorkItem
	revisions  map[string]domain.OutcomeRevision
}

func New() *Store {
	return &Store{
		outcomes:   make(map[string]domain.Outcome),
		objectives: make(map[string]domain.Objective),
		workItems:  make(map[string]domain.WorkItem),
		revisions:  make(map[string]domain.OutcomeRevision),
	}
}

func (s *Store) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, err
	}

	tx := &transaction{
		store:      s,
		outcomes:   cloneOutcomes(s.outcomes),
		objectives: cloneObjectives(s.objectives),
		workItems:  cloneWorkItems(s.workItems),
		revisions:  cloneRevisions(s.revisions),
	}
	tx.outcomeRepo = outcomeRepository{tx: tx}
	tx.objectiveRepo = objectiveRepository{tx: tx}
	tx.workItemRepo = workItemRepository{tx: tx}
	tx.coordination = coordinationStore{tx: tx}
	return tx, nil
}

type transaction struct {
	store      *Store
	closed     bool
	outcomes   map[string]domain.Outcome
	objectives map[string]domain.Objective
	workItems  map[string]domain.WorkItem
	revisions  map[string]domain.OutcomeRevision

	outcomeRepo   outcomeRepository
	objectiveRepo objectiveRepository
	workItemRepo  workItemRepository
	coordination  coordinationStore
}

func (tx *transaction) Outcomes() ports.OutcomeRepository     { return tx.outcomeRepo }
func (tx *transaction) Objectives() ports.ObjectiveRepository { return tx.objectiveRepo }
func (tx *transaction) WorkItems() ports.WorkItemRepository   { return tx.workItemRepo }
func (tx *transaction) Coordination() ports.CoordinationStore { return tx.coordination }

func (tx *transaction) Commit() error {
	if tx.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction already closed")
	}
	tx.store.outcomes = cloneOutcomes(tx.outcomes)
	tx.store.objectives = cloneObjectives(tx.objectives)
	tx.store.workItems = cloneWorkItems(tx.workItems)
	tx.store.revisions = cloneRevisions(tx.revisions)
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
