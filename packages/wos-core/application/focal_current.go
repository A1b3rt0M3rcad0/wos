package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) currentFocalWorkContext(ctx context.Context, u ports.UnitOfWork, repo ports.FocalReadRepository, scope d.Scope, id d.ID, revision d.OutcomeRevision) (WorkContext, error) {
	selected, e := repo.CurrentWork(ctx, scope, id)
	if e != nil {
		return WorkContext{}, e
	}
	work := selected.Work
	outcome, e := repo.FocalOutcome(ctx, scope)
	if e != nil {
		return WorkContext{}, e
	}
	now, e := s.transactionTime(ctx, u)
	if e != nil {
		return WorkContext{}, e
	}
	value := WorkContext{Work: work, Outcome: summary(outcome.Ref(), outcome.Version, outcome.Title, string(outcome.Lifecycle), outcome.Priority), OutcomeRevision: revision, EvaluatedAt: now, Dependencies: []d.Relation{}, Decisions: []d.Decision{}, EvidenceLinks: []d.EvidenceLink{}, Omitted: map[string]int{}}
	var objectiveHistory ports.FocalHistoryCounts
	if work.ObjectiveID != nil {
		selected, e := repo.CurrentObjective(ctx, scope, *work.ObjectiveID)
		if e != nil {
			return WorkContext{}, e
		}
		value.Objective = &selected.Objective
		objectiveHistory = selected.History
	}
	records, e := repo.FocalRecords(ctx, scope, id, work.ObjectiveID)
	if e != nil {
		return WorkContext{}, e
	}
	ancestors := map[d.ID]d.Objective{}
	for _, v := range records.ObjectiveAncestors {
		ancestors[v.ID] = v
	}
	blocking := projectBlockingState(work.Ref(), work.ObjectiveID, records.Blockers, ancestors)
	value.Blocking = &blocking
	value.Operational = d.ProjectWorkItemOperationalState(work, outcome, value.Objective, records.DependencyEvaluations, blocking.IsBlocked, now)
	value.Dependencies = records.Dependencies[:min(100, len(records.Dependencies))]
	value.Decisions = records.Decisions[:min(25, len(records.Decisions))]
	value.EvidenceLinks = records.EvidenceLinks[:min(25, len(records.EvidenceLinks))]
	for key, count := range map[string]int{"dependencies": records.DependencyCount - 100, "decisions": records.DecisionCount - 25, "evidence_links": records.EvidenceLinkCount - 25, "assessment_history": selected.History.Assessments, "criterion_definition_history": selected.History.Definitions, "conclusion_history": selected.History.Conclusions} {
		if count > 0 {
			value.Omitted[key] = count
			value.Truncated = true
		}
	}
	value.Work.Criteria.Assessments = nil
	value.Work.Criteria.DefinitionRevisions = nil
	value.Work.ConclusionHistory = nil
	if value.Objective != nil {
		value.Omitted["objective_assessment_history"] = objectiveHistory.Assessments
		value.Omitted["objective_definition_history"] = objectiveHistory.Definitions
		value.Omitted["objective_conclusion_history"] = objectiveHistory.Conclusions
		if objectiveHistory.Assessments+objectiveHistory.Definitions+objectiveHistory.Conclusions > 0 {
			value.Truncated = true
		}
		value.Objective.Criteria.Assessments = nil
		value.Objective.Criteria.DefinitionRevisions = nil
		value.Objective.ConclusionHistory = nil
	}
	if work.CurrentContractID != nil {
		contracts, e := contractRepository(u)
		if e != nil {
			return WorkContext{}, e
		}
		contract, e := contracts.Get(ctx, scope, *work.CurrentContractID)
		if e != nil {
			return WorkContext{}, e
		}
		candidate := work
		candidate.CurrentLease = nil
		candidate.Lifecycle = d.WorkItemLifecycleTodo
		ready := d.ApplyBlockingToReadiness(d.WorkItemReadiness(candidate, outcome, value.Objective, records.DependencyEvaluations, now), blocking.IsBlocked)
		view, e := renderWorkContractView(ctx, u, scope, contract, work, revision, now, ready)
		if e != nil {
			return WorkContext{}, e
		}
		value.Contract = &view.Value
	}
	raw, e := json.Marshal(value)
	if e != nil {
		return WorkContext{}, e
	}
	if len(raw) > MaxSnapshotBytes {
		return WorkContext{}, d.NewError(d.ErrorCodeGraphLimitExceeded, "work context exceeds byte limit; read individual sections")
	}
	return value, nil
}
