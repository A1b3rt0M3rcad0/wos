package application

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// ParentID is required and nullable: null deliberately places the entity
// directly below its Outcome. This selector never changes persisted state.
type PlacementPreviewQuery struct {
	ParentID *d.ID `json:"parent_id" wos:"required"`
}

func (q *PlacementPreviewQuery) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	value, present := fields["parent_id"]
	if !present || len(fields) != 1 {
		return d.NewError(d.ErrorCodeInvalidArgument, "placement_preview requires only parent_id, including explicit null for the Outcome")
	}
	return json.Unmarshal(value, &q.ParentID)
}

type WorkspacePlacementError struct {
	Code    d.ErrorCode     `json:"code"`
	Message string          `json:"message"`
	Details *d.ErrorDetails `json:"details,omitempty"`
}

type WorkspacePlacementBlocker struct {
	Ref           d.EntityRef  `json:"ref"`
	Version       d.Version    `json:"version,string"`
	Description   string       `json:"description"`
	BlockedRef    d.EntityRef  `json:"blocked_ref"`
	Inherited     bool         `json:"inherited"`
	InheritedFrom *d.EntityRef `json:"inherited_from,omitempty"`
}

type WorkspacePlacementBlocking struct {
	IsBlocked bool                        `json:"is_blocked"`
	Blockers  []WorkspacePlacementBlocker `json:"blockers"`
	Coverage  WorkspaceCoverage           `json:"coverage"`
}

type WorkspacePlacementBlockingChange struct {
	Ref    d.EntityRef                `json:"ref"`
	Title  string                     `json:"title"`
	Before WorkspacePlacementBlocking `json:"before"`
	After  WorkspacePlacementBlocking `json:"after"`
}

type WorkspacePlacementPreview struct {
	OldParent          EntitySummary                      `json:"old_parent"`
	NewParent          EntitySummary                      `json:"new_parent"`
	Changed            bool                               `json:"changed"`
	Availability       string                             `json:"availability"`
	ReasonCodes        []string                           `json:"reason_codes"`
	RelatedRefs        []d.EntityRef                      `json:"related_refs"`
	Remedies           []d.ErrorRemedy                    `json:"remedies"`
	Error              *WorkspacePlacementError           `json:"error,omitempty"`
	AffectedObjectives int                                `json:"affected_objectives"`
	AffectedTasks      int                                `json:"affected_tasks"`
	BlockingChanges    []WorkspacePlacementBlockingChange `json:"blocking_changes"`
	ReadinessBefore    *d.WorkItemOperationalState        `json:"readiness_before,omitempty"`
	ReadinessAfter     *d.WorkItemOperationalState        `json:"readiness_after,omitempty"`
	Coverage           WorkspaceCoverage                  `json:"coverage"`
}

func (s *Service) workspacePlacementPreview(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, q *PlacementPreviewQuery) (*WorkspacePlacementPreview, error) {
	ref := v.Subject.Ref
	if v.Subject.Kind != "entity" || ref.Kind != d.EntityKindWorkItem && ref.Kind != d.EntityKindObjective {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "placement preview requires a Task or Objective entity")
	}
	if q.ParentID != nil {
		if err := q.ParentID.Validate(); err != nil {
			return nil, err
		}
	}
	repo := f.repo.(ports.WorkspaceReadRepository)
	focal := f.repo.(ports.FocalReadRepository)
	graph, ok := f.repo.(ports.WorkspacePlacementReadRepository)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "bounded structural Blocker reads are unavailable")
	}
	p := &WorkspacePlacementPreview{OldParent: v.Placement.Outcome, NewParent: v.Placement.Outcome, Availability: "available", ReasonCodes: []string{}, RelatedRefs: []d.EntityRef{}, Remedies: []d.ErrorRemedy{}, BlockingChanges: []WorkspacePlacementBlockingChange{}}
	if v.Placement.Parent != nil {
		p.OldParent = *v.Placement.Parent
	}
	var condition error
	if q.ParentID != nil {
		parent, err := repo.WorkspaceObjectiveHeader(ctx, ref.Scope, *q.ParentID)
		if err != nil {
			condition = err
			p.NewParent = summary(d.EntityRef{Scope: ref.Scope, Kind: d.EntityKindObjective, ID: *q.ParentID}, 0, "Unavailable Objective", "", "")
		} else {
			p.NewParent = summary(parent.Ref(), parent.Version, parent.Title, string(parent.Lifecycle), parent.Priority)
		}
	}
	p.Changed = p.OldParent.Ref != p.NewParent.Ref
	outcome, err := focal.FocalOutcome(ctx, ref.Scope)
	if err != nil {
		return nil, err
	}
	if condition == nil && (outcome.IsArchived() || outcome.Lifecycle == d.OutcomeLifecycleAchieved || outcome.Lifecycle == d.OutcomeLifecycleFailed || outcome.Lifecycle == d.OutcomeLifecycleAbandoned) {
		condition = definitionPrecondition(d.ErrorCodePreconditionFailed, "outcome_closed", "reopen or unarchive the Outcome before reorganizing its work", ref, nil, outcome.Ref())
	}
	objectives, items, err := repo.WorkspaceStructure(ctx, ref.Scope, d.DefaultDependencyGraphLimit+1)
	if err != nil {
		return nil, err
	}
	if len(objectives) > d.DefaultDependencyGraphLimit || len(items) > d.DefaultDependencyGraphLimit {
		return nil, d.NewError(d.ErrorCodeGraphLimitExceeded, "placement preview exceeds the structural graph limit; no partial impact is presented as complete")
	}
	before := make(map[d.ID]d.Objective, len(objectives))
	after := make(map[d.ID]d.Objective, len(objectives))
	for _, item := range objectives {
		before[item.ID], after[item.ID] = item, item
	}
	affected := []d.EntityRef{ref}
	parents := map[d.EntityRef]*d.ID{}
	var selectedWork d.WorkItem
	action := "SetObjectiveParent"
	if ref.Kind == d.EntityKindWorkItem {
		action = "SetWorkItemObjective"
		current, err := focal.CurrentWork(ctx, ref.Scope, ref.ID)
		if err != nil {
			return nil, err
		}
		selectedWork = current.Work
		parents[ref] = selectedWork.ObjectiveID
		p.AffectedTasks = 1
		if condition == nil {
			condition = validateWorkDefinitionChange(ctx, s, f.u, selectedWork, f.at)
		}
		if condition == nil {
			condition = validateWorkObjectiveChange(ctx, f.u, selectedWork, q.ParentID)
		}
	} else {
		current, err := focal.CurrentObjective(ctx, ref.Scope, ref.ID)
		if err != nil {
			return nil, err
		}
		if condition == nil {
			condition = validateObjectiveParentChange(ctx, s, f.u, current.Objective, q.ParentID, f.at)
		}
		moved := after[ref.ID]
		moved.ParentObjectiveID = q.ParentID
		after[ref.ID] = moved
		p.AffectedObjectives = 1
		// Tasks precede other descendant Objectives in the bounded sample, so
		// a large hierarchy still exposes affected operational work.
		for _, item := range items {
			if item.ObjectiveID != nil && objectiveDescendsFrom(*item.ObjectiveID, ref.ID, before) {
				affected = append(affected, item.Ref())
				parents[item.Ref()] = item.ObjectiveID
				p.AffectedTasks++
			}
		}
		for _, item := range objectives {
			if item.ID != ref.ID && objectiveDescendsFrom(item.ID, ref.ID, before) {
				affected = append(affected, item.Ref())
				p.AffectedObjectives++
			}
		}
	}
	// Base capabilities include permission and protocol checks. A preview is
	// explanatory; neither an available result nor its revision grants authority.
	permitted := false
	for _, capability := range v.Capabilities {
		if capability.ActionID == action {
			permitted = true
			if condition == nil && capability.Availability != "available" {
				p.Availability = "blocked"
				p.ReasonCodes = append(p.ReasonCodes, capability.ReasonCodes...)
				p.RelatedRefs = append(p.RelatedRefs, capability.RelatedRefs...)
				p.Remedies = append(p.Remedies, capability.Remedies...)
			}
		}
	}
	if !permitted && condition == nil {
		condition = d.NewError(d.ErrorCodeForbidden, "the current identity cannot change this placement")
	}
	if condition != nil {
		var detail *d.Error
		if !errors.As(condition, &detail) {
			return nil, condition
		}
		if detail.Code == d.ErrorCodeInvalidConfig {
			return nil, condition
		}
		p.Availability = "blocked"
		p.ReasonCodes = []string{string(detail.Code)}
		p.Error = &WorkspacePlacementError{Code: detail.Code, Message: detail.Message, Details: detail.Details}
		if detail.Details != nil {
			if detail.Details.ReasonCode != "" {
				p.ReasonCodes = []string{detail.Details.ReasonCode}
			}
			p.RelatedRefs = append(p.RelatedRefs, detail.Details.RelatedRefs...)
			p.Remedies = append(p.Remedies, detail.Details.Remedies...)
		}
		if detail.Code == d.ErrorCodeNotFound || detail.Code == d.ErrorCodeDependencyCycle {
			// There is no valid candidate ancestry to project. Do not present
			// missing/cyclic ancestors as an empty inherited-Blocker set.
			p.Coverage = WorkspaceCoverage{Status: "unknown", ReturnedCount: workspaceCount(0), TotalCount: workspaceCount(len(affected)), Reason: "invalid_candidate_placement; fix the destination before reviewing operational impact"}
			v.Coverage["placement_preview"] = p.Coverage
			return p, nil
		}
	}
	blockers, err := graph.WorkspacePlacementBlockers(ctx, ref.Scope, d.DefaultDependencyGraphLimit+1)
	if err != nil {
		return nil, err
	}
	if len(blockers) > d.DefaultDependencyGraphLimit {
		return nil, d.NewError(d.ErrorCodeGraphLimitExceeded, "placement preview exceeds the active Blocker graph limit")
	}
	const sampleLimit = 20
	sample := affected[:min(len(affected), sampleLimit)]
	names := map[d.EntityRef]string{ref: v.Header.Title}
	if nodes, ok := f.repo.(ports.GraphReadRepository); ok && len(sample) > 1 {
		values, err := nodes.GraphNodes(ctx, ref.Scope, sample[1:])
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			names[value.Ref] = value.Title
		}
	}
	for _, target := range sample {
		oldParent, newParent := parents[target], parents[target]
		if target.Kind == d.EntityKindObjective {
			id := target.ID
			oldParent, newParent = &id, &id
		} else if target == ref {
			newParent = q.ParentID
		}
		oldBlocking := projectBlockingState(target, oldParent, blockers, before)
		newBlocking := projectBlockingState(target, newParent, blockers, after)
		p.BlockingChanges = append(p.BlockingChanges, WorkspacePlacementBlockingChange{Ref: target, Title: names[target], Before: workspacePlacementBlocking(oldBlocking), After: workspacePlacementBlocking(newBlocking)})
	}
	if ref.Kind == d.EntityKindWorkItem {
		records, err := focal.FocalRecords(ctx, ref.Scope, ref.ID, selectedWork.ObjectiveID)
		if err != nil {
			return nil, err
		}
		var oldParent, newParent *d.Objective
		if selectedWork.ObjectiveID != nil {
			value := before[*selectedWork.ObjectiveID]
			oldParent = &value
		}
		if q.ParentID != nil {
			if value, found := after[*q.ParentID]; found {
				newParent = &value
			}
		}
		oldBlocking := projectBlockingState(ref, selectedWork.ObjectiveID, blockers, before)
		newBlocking := projectBlockingState(ref, q.ParentID, blockers, after)
		old := d.ProjectWorkItemOperationalState(selectedWork, outcome, oldParent, records.DependencyEvaluations, oldBlocking.IsBlocked, f.at)
		candidate := selectedWork
		candidate.ObjectiveID = q.ParentID
		next := d.ProjectWorkItemOperationalState(candidate, outcome, newParent, records.DependencyEvaluations, newBlocking.IsBlocked, f.at)
		p.ReadinessBefore, p.ReadinessAfter = &old, &next
	}
	p.Coverage = WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(sample)), TotalCount: workspaceCount(len(affected))}
	if len(sample) < len(affected) {
		p.Coverage.Status, p.Coverage.Reason = "partial", "bounded_affected_entity_sample; affected counts include the entire validated branch"
	}
	v.Coverage["placement_preview"] = p.Coverage
	return p, nil
}

func workspacePlacementBlocking(state BlockingState) WorkspacePlacementBlocking {
	result := WorkspacePlacementBlocking{IsBlocked: state.IsBlocked, Blockers: []WorkspacePlacementBlocker{}}
	sort.Slice(state.ActiveBlockers, func(i, j int) bool { return state.ActiveBlockers[i].Blocker.ID < state.ActiveBlockers[j].Blocker.ID })
	// A structural sample can contain twenty entities. Keep each side's
	// documentary Blocker sample small enough for the shared 256-KiB envelope;
	// the boolean and total always evaluate the complete validated graph.
	for _, item := range state.ActiveBlockers[:min(5, len(state.ActiveBlockers))] {
		result.Blockers = append(result.Blockers, WorkspacePlacementBlocker{Ref: item.Blocker.Ref(), Version: item.Blocker.Version, Description: workspaceText(item.Blocker.Description, 512), BlockedRef: item.Blocker.BlockedRef, Inherited: item.Inherited, InheritedFrom: item.InheritedFrom})
	}
	result.Coverage = WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(result.Blockers)), TotalCount: workspaceCount(len(state.ActiveBlockers))}
	if len(result.Blockers) < len(state.ActiveBlockers) {
		result.Coverage.Status, result.Coverage.Reason = "partial", "bounded_active_Blocker_sample; is_blocked uses every active Blocker"
	}
	return result
}
