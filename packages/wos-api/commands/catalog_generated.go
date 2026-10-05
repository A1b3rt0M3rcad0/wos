// Code generated from public Application commands; DO NOT EDIT.
package commands

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func NewCatalog(service *application.Service, ids ports.IDGenerator) *Catalog {
	c := newCatalog(ids)
	register[application.AbandonOutcomeCommand](c, "abandon_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.AbandonOutcomeCommand) (any, error) {
		return service.AbandonOutcome(ctx, cc, cmd)
	})
	register[application.AcceptDecisionCommand](c, "accept_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.AcceptDecisionCommand) (any, error) {
		return service.AcceptDecision(ctx, cc, cmd)
	})
	register[application.AchieveObjectiveCommand](c, "achieve_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.AchieveObjectiveCommand) (any, error) {
		return service.AchieveObjective(ctx, cc, cmd)
	})
	register[application.AchieveOutcomeCommand](c, "achieve_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.AchieveOutcomeCommand) (any, error) {
		return service.AchieveOutcome(ctx, cc, cmd)
	})
	register[application.ActivateOutcomeCommand](c, "activate_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.ActivateOutcomeCommand) (any, error) {
		return service.ActivateOutcome(ctx, cc, cmd)
	})
	register[application.ActivateRoadmapRevisionCommand](c, "activate_roadmap_revision", func(ctx context.Context, cc domain.CommandContext, cmd application.ActivateRoadmapRevisionCommand) (any, error) {
		return service.ActivateRoadmapRevision(ctx, cc, cmd)
	})
	register[application.ActivateWorkItemCommand](c, "activate_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ActivateWorkItemCommand) (any, error) {
		return service.ActivateWorkItem(ctx, cc, cmd)
	})
	register[application.AddCriterionCommand](c, "add_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.AddCriterionCommand) (any, error) {
		return service.AddCriterion(ctx, cc, cmd)
	})
	register[application.AddDependencyCommand](c, "add_dependency", func(ctx context.Context, cc domain.CommandContext, cmd application.AddDependencyCommand) (any, error) {
		return service.AddDependency(ctx, cc, cmd)
	})
	register[application.AdministrativeCancelWorkItemCommand](c, "administrative_cancel_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.AdministrativeCancelWorkItemCommand) (any, error) {
		return service.AdministrativeCancelWorkItem(ctx, cc, cmd)
	})
	register[application.AdministrativeCompleteWorkItemCommand](c, "administrative_complete_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.AdministrativeCompleteWorkItemCommand) (any, error) {
		return service.AdministrativeCompleteWorkItem(ctx, cc, cmd)
	})
	register[application.ArchiveOutcomeCommand](c, "archive_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.ArchiveOutcomeCommand) (any, error) {
		return service.ArchiveOutcome(ctx, cc, cmd)
	})
	register[application.ArchiveRoadmapCommand](c, "archive_roadmap", func(ctx context.Context, cc domain.CommandContext, cmd application.ArchiveRoadmapCommand) (any, error) {
		return service.ArchiveRoadmap(ctx, cc, cmd)
	})
	register[application.AttestCriterionCommand](c, "attest_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.AttestCriterionCommand) (any, error) {
		return service.AttestCriterion(ctx, cc, cmd)
	})
	register[application.CancelBlockerCommand](c, "cancel_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.CancelBlockerCommand) (any, error) {
		return service.CancelBlocker(ctx, cc, cmd)
	})
	register[application.CancelObjectiveCommand](c, "cancel_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.CancelObjectiveCommand) (any, error) {
		return service.CancelObjective(ctx, cc, cmd)
	})
	register[application.CancelWorkItemCommand](c, "cancel_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.CancelWorkItemCommand) (any, error) {
		return service.CancelWorkItem(ctx, cc, cmd)
	})
	register[application.ClaimWorkItemCommand](c, "claim_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ClaimWorkItemCommand) (any, error) {
		return service.ClaimWorkItem(ctx, cc, cmd)
	})
	register[application.CompleteWorkItemCommand](c, "complete_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.CompleteWorkItemCommand) (any, error) {
		return service.CompleteWorkItem(ctx, cc, cmd)
	})
	register[application.ConfigureTriggerCommand](c, "configure_trigger", func(ctx context.Context, cc domain.CommandContext, cmd application.ConfigureTriggerCommand) (any, error) {
		return service.ConfigureTrigger(ctx, cc, cmd)
	})
	register[application.CreateBlockerCommand](c, "create_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateBlockerCommand) (any, error) {
		return service.CreateBlocker(ctx, cc, cmd)
	})
	register[application.CreateEvidenceLinkCommand](c, "create_evidence_link", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateEvidenceLinkCommand) (any, error) {
		return service.CreateEvidenceLink(ctx, cc, cmd)
	})
	register[application.CreateIssueCommand](c, "create_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateIssueCommand) (any, error) {
		return service.CreateIssue(ctx, cc, cmd)
	})
	register[application.CreateObjectiveCommand](c, "create_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateObjectiveCommand) (any, error) {
		return service.CreateObjective(ctx, cc, cmd)
	})
	register[application.CreateOutcomeCommand](c, "create_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateOutcomeCommand) (any, error) {
		return service.CreateOutcome(ctx, cc, cmd)
	})
	register[application.CreateRoadmapCommand](c, "create_roadmap", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateRoadmapCommand) (any, error) {
		return service.CreateRoadmap(ctx, cc, cmd)
	})
	register[application.CreateWorkItemCommand](c, "create_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateWorkItemCommand) (any, error) {
		return service.CreateWorkItem(ctx, cc, cmd)
	})
	register[application.DeactivateRoadmapRevisionCommand](c, "deactivate_roadmap_revision", func(ctx context.Context, cc domain.CommandContext, cmd application.DeactivateRoadmapRevisionCommand) (any, error) {
		return service.DeactivateRoadmapRevision(ctx, cc, cmd)
	})
	register[application.DeferWorkItemCommand](c, "defer_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.DeferWorkItemCommand) (any, error) {
		return service.DeferWorkItem(ctx, cc, cmd)
	})
	register[application.DiscardRoadmapDraftCommand](c, "discard_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.DiscardRoadmapDraftCommand) (any, error) {
		return service.DiscardRoadmapDraft(ctx, cc, cmd)
	})
	register[application.FailOutcomeCommand](c, "fail_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.FailOutcomeCommand) (any, error) {
		return service.FailOutcome(ctx, cc, cmd)
	})
	register[application.InvestigateIssueCommand](c, "investigate_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.InvestigateIssueCommand) (any, error) {
		return service.InvestigateIssue(ctx, cc, cmd)
	})
	register[application.LinkExternalReferenceCommand](c, "link_external_reference", func(ctx context.Context, cc domain.CommandContext, cmd application.LinkExternalReferenceCommand) (any, error) {
		return service.LinkExternalReference(ctx, cc, cmd)
	})
	register[application.MarkIssueDuplicateCommand](c, "mark_issue_duplicate", func(ctx context.Context, cc domain.CommandContext, cmd application.MarkIssueDuplicateCommand) (any, error) {
		return service.MarkIssueDuplicate(ctx, cc, cmd)
	})
	register[application.MarkIssueWontFixCommand](c, "mark_issue_wont_fix", func(ctx context.Context, cc domain.CommandContext, cmd application.MarkIssueWontFixCommand) (any, error) {
		return service.MarkIssueWontFix(ctx, cc, cmd)
	})
	register[application.OpenRoadmapDraftCommand](c, "open_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.OpenRoadmapDraftCommand) (any, error) {
		return service.OpenRoadmapDraft(ctx, cc, cmd)
	})
	register[application.ProposeDecisionCommand](c, "propose_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.ProposeDecisionCommand) (any, error) {
		return service.ProposeDecision(ctx, cc, cmd)
	})
	register[application.PublishRoadmapDraftCommand](c, "publish_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.PublishRoadmapDraftCommand) (any, error) {
		return service.PublishRoadmapDraft(ctx, cc, cmd)
	})
	register[application.ReclaimWorkItemCommand](c, "reclaim_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ReclaimWorkItemCommand) (any, error) {
		return service.ReclaimWorkItem(ctx, cc, cmd)
	})
	register[application.RecordCriterionAssessmentCommand](c, "record_criterion_assessment", func(ctx context.Context, cc domain.CommandContext, cmd application.RecordCriterionAssessmentCommand) (any, error) {
		return service.RecordCriterionAssessment(ctx, cc, cmd)
	})
	register[application.RedeliverDeliveryCommand](c, "redeliver_delivery", func(ctx context.Context, cc domain.CommandContext, cmd application.RedeliverDeliveryCommand) (any, error) {
		return service.RedeliverDelivery(ctx, cc, cmd)
	})
	register[application.RegisterArtifactCommand](c, "register_artifact", func(ctx context.Context, cc domain.CommandContext, cmd application.RegisterArtifactCommand) (any, error) {
		return service.RegisterArtifact(ctx, cc, cmd)
	})
	register[application.RegisterEvidenceCommand](c, "register_evidence", func(ctx context.Context, cc domain.CommandContext, cmd application.RegisterEvidenceCommand) (any, error) {
		return service.RegisterEvidence(ctx, cc, cmd)
	})
	register[application.RejectDecisionCommand](c, "reject_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.RejectDecisionCommand) (any, error) {
		return service.RejectDecision(ctx, cc, cmd)
	})
	register[application.ReleaseWorkItemCommand](c, "release_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ReleaseWorkItemCommand) (any, error) {
		return service.ReleaseWorkItem(ctx, cc, cmd)
	})
	register[application.RemoveDependencyCommand](c, "remove_dependency", func(ctx context.Context, cc domain.CommandContext, cmd application.RemoveDependencyCommand) (any, error) {
		return service.RemoveDependency(ctx, cc, cmd)
	})
	register[application.RemoveExternalReferenceCommand](c, "remove_external_reference", func(ctx context.Context, cc domain.CommandContext, cmd application.RemoveExternalReferenceCommand) (any, error) {
		return service.RemoveExternalReference(ctx, cc, cmd)
	})
	register[application.RenewWorkItemLeaseCommand](c, "renew_work_item_lease", func(ctx context.Context, cc domain.CommandContext, cmd application.RenewWorkItemLeaseCommand) (any, error) {
		return service.RenewWorkItemLease(ctx, cc, cmd)
	})
	register[application.ReopenIssueCommand](c, "reopen_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenIssueCommand) (any, error) {
		return service.ReopenIssue(ctx, cc, cmd)
	})
	register[application.ReopenObjectiveCommand](c, "reopen_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenObjectiveCommand) (any, error) {
		return service.ReopenObjective(ctx, cc, cmd)
	})
	register[application.ReopenOutcomeCommand](c, "reopen_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenOutcomeCommand) (any, error) {
		return service.ReopenOutcome(ctx, cc, cmd)
	})
	register[application.ReopenRoadmapCommand](c, "reopen_roadmap", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenRoadmapCommand) (any, error) {
		return service.ReopenRoadmap(ctx, cc, cmd)
	})
	register[application.ReopenWorkItemCommand](c, "reopen_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenWorkItemCommand) (any, error) {
		return service.ReopenWorkItem(ctx, cc, cmd)
	})
	register[application.ReplaceRoadmapDraftCommand](c, "replace_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.ReplaceRoadmapDraftCommand) (any, error) {
		return service.ReplaceRoadmapDraft(ctx, cc, cmd)
	})
	register[application.ReportIssueWithBlockerCommand](c, "report_issue_with_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.ReportIssueWithBlockerCommand) (any, error) {
		return service.ReportIssueWithBlocker(ctx, cc, cmd)
	})
	register[application.ResolveBlockerCommand](c, "resolve_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.ResolveBlockerCommand) (any, error) {
		return service.ResolveBlocker(ctx, cc, cmd)
	})
	register[application.ResolveIssueCommand](c, "resolve_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.ResolveIssueCommand) (any, error) {
		return service.ResolveIssue(ctx, cc, cmd)
	})
	register[application.ResolveIssueAndBlockersCommand](c, "resolve_issue_and_blockers", func(ctx context.Context, cc domain.CommandContext, cmd application.ResolveIssueAndBlockersCommand) (any, error) {
		return service.ResolveIssueAndBlockers(ctx, cc, cmd)
	})
	register[application.RetireCriterionCommand](c, "retire_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.RetireCriterionCommand) (any, error) {
		return service.RetireCriterion(ctx, cc, cmd)
	})
	register[application.RetractEvidenceCommand](c, "retract_evidence", func(ctx context.Context, cc domain.CommandContext, cmd application.RetractEvidenceCommand) (any, error) {
		return service.RetractEvidence(ctx, cc, cmd)
	})
	register[application.RetractEvidenceLinkCommand](c, "retract_evidence_link", func(ctx context.Context, cc domain.CommandContext, cmd application.RetractEvidenceLinkCommand) (any, error) {
		return service.RetractEvidenceLink(ctx, cc, cmd)
	})
	register[application.ReviseCriterionCommand](c, "revise_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.ReviseCriterionCommand) (any, error) {
		return service.ReviseCriterion(ctx, cc, cmd)
	})
	register[application.SetObjectiveOwnersCommand](c, "set_objective_owners", func(ctx context.Context, cc domain.CommandContext, cmd application.SetObjectiveOwnersCommand) (any, error) {
		return service.SetObjectiveOwners(ctx, cc, cmd)
	})
	register[application.SetOutcomeOwnersCommand](c, "set_outcome_owners", func(ctx context.Context, cc domain.CommandContext, cmd application.SetOutcomeOwnersCommand) (any, error) {
		return service.SetOutcomeOwners(ctx, cc, cmd)
	})
	register[application.SetTriggerEnabledCommand](c, "set_trigger_enabled", func(ctx context.Context, cc domain.CommandContext, cmd application.SetTriggerEnabledCommand) (any, error) {
		return service.SetTriggerEnabled(ctx, cc, cmd)
	})
	register[application.SetWorkItemAssigneesCommand](c, "set_work_item_assignees", func(ctx context.Context, cc domain.CommandContext, cmd application.SetWorkItemAssigneesCommand) (any, error) {
		return service.SetWorkItemAssignees(ctx, cc, cmd)
	})
	register[application.StartObjectiveCommand](c, "start_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.StartObjectiveCommand) (any, error) {
		return service.StartObjective(ctx, cc, cmd)
	})
	register[application.SupersedeDecisionCommand](c, "supersede_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.SupersedeDecisionCommand) (any, error) {
		return service.SupersedeDecision(ctx, cc, cmd)
	})
	register[application.UnarchiveOutcomeCommand](c, "unarchive_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.UnarchiveOutcomeCommand) (any, error) {
		return service.UnarchiveOutcome(ctx, cc, cmd)
	})
	register[application.UpdateBlockerDescriptionCommand](c, "update_blocker_description", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateBlockerDescriptionCommand) (any, error) {
		return service.UpdateBlockerDescription(ctx, cc, cmd)
	})
	register[application.UpdateDecisionCommand](c, "update_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateDecisionCommand) (any, error) {
		return service.UpdateDecision(ctx, cc, cmd)
	})
	register[application.UpdateIssueCommand](c, "update_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateIssueCommand) (any, error) {
		return service.UpdateIssue(ctx, cc, cmd)
	})
	register[application.UpdateObjectiveCommand](c, "update_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateObjectiveCommand) (any, error) {
		return service.UpdateObjective(ctx, cc, cmd)
	})
	register[application.UpdateOutcomeCommand](c, "update_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateOutcomeCommand) (any, error) {
		return service.UpdateOutcome(ctx, cc, cmd)
	})
	register[application.UpdateTriggerCommand](c, "update_trigger", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateTriggerCommand) (any, error) {
		return service.UpdateTrigger(ctx, cc, cmd)
	})
	register[application.UpdateWorkItemCommand](c, "update_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateWorkItemCommand) (any, error) {
		return service.UpdateWorkItem(ctx, cc, cmd)
	})
	register[application.WithdrawArtifactCommand](c, "withdraw_artifact", func(ctx context.Context, cc domain.CommandContext, cmd application.WithdrawArtifactCommand) (any, error) {
		return service.WithdrawArtifact(ctx, cc, cmd)
	})
	return c
}
