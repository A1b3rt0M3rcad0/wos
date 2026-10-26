package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

func workspaceDocument(ctx context.Context, u ports.UnitOfWork, ref d.EntityRef) (any, error) {
	switch ref.Kind {
	case d.EntityKindIssue:
		if r, ok := u.(ports.IssueBlockerUnitOfWork); ok {
			return r.Issues().Get(ctx, ref.Scope, ref.ID)
		}
	case d.EntityKindBlocker:
		if r, ok := u.(ports.IssueBlockerUnitOfWork); ok {
			return r.Blockers().Get(ctx, ref.Scope, ref.ID)
		}
	case d.EntityKindRelation:
		return u.Relations().Get(ctx, ref.Scope, ref.ID)
	case d.EntityKindRoadmap:
		if r, ok := u.(ports.PlanningUnitOfWork); ok {
			return r.Roadmaps().Get(ctx, ref.Scope, ref.ID)
		}
	default:
		if r, ok := u.(ports.DocumentaryUnitOfWork); ok {
			switch ref.Kind {
			case d.EntityKindEvidence:
				return r.Evidence().Get(ctx, ref.Scope, ref.ID)
			case d.EntityKindEvidenceLink:
				return r.EvidenceLinks().Get(ctx, ref.Scope, ref.ID)
			case d.EntityKindArtifact:
				return r.Artifacts().Get(ctx, ref.Scope, ref.ID)
			case d.EntityKindDecision:
				return r.Decisions().Get(ctx, ref.Scope, ref.ID)
			}
		}
	}
	return nil, d.NewError(d.ErrorCodeInvalidArgument, "unsupported workspace entity kind")
}
func workspaceDocumentHeader(record any, ref d.EntityRef) (WorkspaceHeader, d.Version, []d.ActorRef) {
	h := WorkspaceHeader{Kind: ref.Kind.String()}
	var version d.Version
	people := []d.ActorRef{}
	switch v := record.(type) {
	case d.Issue:
		h.Title = v.Title
		h.Description = v.Description
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
		people = append(people, v.ReportedBy)
	case d.Blocker:
		h.Description = v.Description
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
	case d.Evidence:
		h.Description = v.Description
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
		people = append(people, v.ProducerRef)
	case d.Artifact:
		h.Title = v.Name
		h.Description = v.URI
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
		people = append(people, v.ProducerRef)
	case d.Decision:
		h.Title = v.Title
		h.Description = v.Proposal
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
	case d.Relation:
		h.Title = string(v.RelationType)
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
	case d.EvidenceLink:
		h.Title = string(v.Stance)
		h.Description = v.Rationale
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
	case d.Roadmap:
		h.Title = v.Title
		h.Lifecycle = string(v.Lifecycle)
		version = v.Version
	}
	h.Description = workspaceText(h.Description, 1024)
	return h, version, people
}
func workspaceExecutionRecord(ctx context.Context, u ports.UnitOfWork, scope d.Scope, subject WorkspaceSubject, at time.Time) (any, *WorkspaceDelivery, error) {
	legacy, e := contractRepository(u)
	if e != nil {
		return nil, nil, e
	}
	switch subject.RecordKind {
	case "contract":
		c, e := focalWorkContract(ctx, u, legacy, scope, *subject.RecordID)
		if e != nil {
			return nil, nil, e
		}
		if c.WorkItemID != subject.Ref.ID {
			return nil, nil, d.NewError(d.ErrorCodeNotFound, "execution record not found for this Task")
		}
		v := map[string]any{"id": c.ID, "version": c.Version, "status": c.Status, "effective_status": c.EffectiveStatus(at), "work_item_id": c.WorkItemID, "holder_principal_id": c.HolderPrincipalID, "actor_ref": c.Actor, "acquired_at": c.AcquiredAt, "expires_at": c.ExpiresAt}
		return v, &WorkspaceDelivery{ContractID: &c.ID, Status: string(c.EffectiveStatus(at)), SubmissionID: c.LatestSubmissionID}, nil
	case "submission":
		sub, e := legacy.GetSubmission(ctx, scope, *subject.RecordID)
		if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
			if r, ok := u.(ports.SignedContractUnitOfWork); ok {
				sub, e = r.SignedWorkContracts().GetSubmission(ctx, scope, *subject.RecordID)
			}
		}
		if e != nil {
			return nil, nil, e
		}
		if sub.Material.WorkItemID != subject.Ref.ID {
			return nil, nil, d.NewError(d.ErrorCodeNotFound, "submission not found for this Task")
		}
		v := map[string]any{"id": sub.ID, "work_item_id": sub.Material.WorkItemID, "contract_id": sub.Material.ContractID, "submission_digest": sub.Digest, "principal_id": sub.PrincipalID, "actor_ref": sub.Actor, "submitted_at": sub.SubmittedAt, "summary": workspaceText(sub.Material.Summary, 4096)}
		return v, &WorkspaceDelivery{ContractID: &sub.Material.ContractID, SubmissionID: &sub.ID, SubmissionDigest: sub.Digest, Summary: workspaceText(sub.Material.Summary, 1024), Status: "recorded"}, nil
	case "review_case":
		r, ok := u.(ports.SignedContractUnitOfWork)
		if !ok {
			return nil, nil, d.NewError(d.ErrorCodeInvalidConfig, "signed review reads unavailable")
		}
		c, e := r.SignedContracts().Case(ctx, scope, *subject.RecordID)
		if e != nil {
			return nil, nil, e
		}
		if c.WorkItemID != subject.Ref.ID {
			return nil, nil, d.NewError(d.ErrorCodeNotFound, "review case not found for this Task")
		}
		v := map[string]any{"id": c.ID, "version": c.Version, "work_item_id": c.WorkItemID, "status": c.Status, "submission_id": c.SubmissionID, "submission_digest": c.SubmissionDigest, "created_at": c.CreatedAt, "updated_at": c.UpdatedAt}
		return v, &WorkspaceDelivery{ContractID: &c.WorkContractID, SubmissionID: &c.SubmissionID, SubmissionDigest: c.SubmissionDigest, ReviewCaseID: &c.ID, Status: string(c.Status)}, nil
	}
	return nil, nil, d.NewError(d.ErrorCodeInvalidArgument, "unknown execution record kind")
}
