package application

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"

// Historical labels remain separate from live execution state at the same read revision.
type PlanLiveReference struct {
	RoadmapID               domain.ID                        `json:"roadmap_id"`
	RevisionNumber          uint64                           `json:"revision_number"`
	PlanScope               domain.RoadmapPlanScope          `json:"plan_scope"`
	NodeKey                 string                           `json:"node_key"`
	PlanLabel               string                           `json:"plan_label"`
	PublishedReferenceTitle string                           `json:"published_reference_title"`
	Target                  domain.EntityRef                 `json:"target_ref"`
	Current                 *EntitySummary                   `json:"current,omitempty"`
	Operational             *domain.WorkItemOperationalState `json:"operational_state,omitempty"`
	Missing                 bool                             `json:"missing"`
}

func projectPlanReferences(slot domain.RoadmapActiveSlot, revision domain.RoadmapRevision, objectives []domain.Objective, work []domain.WorkItem, operational []domain.WorkItemOperationalState) []PlanLiveReference {
	current := map[domain.EntityRef]EntitySummary{}
	states := map[domain.EntityRef]domain.WorkItemOperationalState{}
	for _, v := range objectives {
		current[v.Ref()] = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority)
	}
	for _, v := range work {
		current[v.Ref()] = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority)
	}
	for _, v := range operational {
		states[v.Ref] = v
	}
	values := []PlanLiveReference{}
	for _, node := range revision.Nodes {
		if node.TargetRef == nil {
			continue
		}
		v := PlanLiveReference{RoadmapID: slot.RoadmapID, RevisionNumber: slot.RevisionNumber, PlanScope: slot.PlanScope, NodeKey: node.NodeKey, PlanLabel: node.Title, Target: *node.TargetRef}
		if node.ReferenceSnapshot != nil {
			v.PublishedReferenceTitle = node.ReferenceSnapshot.Title
		}
		if entity, ok := current[v.Target]; ok {
			v.Current = &entity
		} else {
			v.Missing = true
		}
		if state, ok := states[v.Target]; ok {
			v.Operational = &state
		}
		values = append(values, v)
	}
	return values
}
