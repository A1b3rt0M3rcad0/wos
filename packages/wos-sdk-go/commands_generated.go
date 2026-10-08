// Code generated from public Application commands; DO NOT EDIT.
package wossdk

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (c *Client) AbandonOutcome(ctx context.Context, key string, cmd application.AbandonOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "abandon_outcome", key, cmd)
}
func (c *Client) AcceptDecision(ctx context.Context, key string, cmd application.AcceptDecisionCommand) (CommandResult[domain.Decision], error) {
	return command[domain.Decision](ctx, c, "accept_decision", key, cmd)
}
func (c *Client) AchieveObjective(ctx context.Context, key string, cmd application.AchieveObjectiveCommand) (CommandResult[domain.Objective], error) {
	return command[domain.Objective](ctx, c, "achieve_objective", key, cmd)
}
func (c *Client) AchieveOutcome(ctx context.Context, key string, cmd application.AchieveOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "achieve_outcome", key, cmd)
}
func (c *Client) AcquireNextWorkContract(ctx context.Context, key string, cmd application.AcquireNextWorkContractCommand) (CommandResult[application.WorkContractAcquisition], error) {
	return command[application.WorkContractAcquisition](ctx, c, "acquire_next_work_contract", key, cmd)
}
func (c *Client) AcquireSignedReviewContract(ctx context.Context, key string, cmd application.AcquireSignedReviewContractCommand) (CommandResult[application.SignedReviewContractResult], error) {
	return command[application.SignedReviewContractResult](ctx, c, "acquire_signed_review_contract", key, cmd)
}
func (c *Client) AcquireSignedWorkContract(ctx context.Context, key string, cmd application.AcquireSignedWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "acquire_signed_work_contract", key, cmd)
}
func (c *Client) AcquireWorkContract(ctx context.Context, key string, cmd application.AcquireWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "acquire_work_contract", key, cmd)
}
func (c *Client) ActivateOutcome(ctx context.Context, key string, cmd application.ActivateOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "activate_outcome", key, cmd)
}
func (c *Client) ActivateRoadmapRevision(ctx context.Context, key string, cmd application.ActivateRoadmapRevisionCommand) (CommandResult[domain.RoadmapActiveSlot], error) {
	return command[domain.RoadmapActiveSlot](ctx, c, "activate_roadmap_revision", key, cmd)
}
func (c *Client) ActivateWorkItem(ctx context.Context, key string, cmd application.ActivateWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "activate_work_item", key, cmd)
}
func (c *Client) AddCriterion(ctx context.Context, key string, cmd application.AddCriterionCommand) (CommandResult[domain.SuccessCriterion], error) {
	return command[domain.SuccessCriterion](ctx, c, "add_criterion", key, cmd)
}
func (c *Client) AddDependency(ctx context.Context, key string, cmd application.AddDependencyCommand) (CommandResult[domain.Relation], error) {
	return command[domain.Relation](ctx, c, "add_dependency", key, cmd)
}
func (c *Client) AdministrativeCancelWorkItem(ctx context.Context, key string, cmd application.AdministrativeCancelWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "administrative_cancel_work_item", key, cmd)
}
func (c *Client) AdministrativeCompleteWorkItem(ctx context.Context, key string, cmd application.AdministrativeCompleteWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "administrative_complete_work_item", key, cmd)
}
func (c *Client) ArchiveOutcome(ctx context.Context, key string, cmd application.ArchiveOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "archive_outcome", key, cmd)
}
func (c *Client) ArchiveRoadmap(ctx context.Context, key string, cmd application.ArchiveRoadmapCommand) (CommandResult[domain.Roadmap], error) {
	return command[domain.Roadmap](ctx, c, "archive_roadmap", key, cmd)
}
func (c *Client) AttestCriterion(ctx context.Context, key string, cmd application.AttestCriterionCommand) (CommandResult[domain.CriterionAssessment], error) {
	return command[domain.CriterionAssessment](ctx, c, "attest_criterion", key, cmd)
}
func (c *Client) CancelBlocker(ctx context.Context, key string, cmd application.CancelBlockerCommand) (CommandResult[domain.Blocker], error) {
	return command[domain.Blocker](ctx, c, "cancel_blocker", key, cmd)
}
func (c *Client) CancelObjective(ctx context.Context, key string, cmd application.CancelObjectiveCommand) (CommandResult[domain.Objective], error) {
	return command[domain.Objective](ctx, c, "cancel_objective", key, cmd)
}
func (c *Client) CancelWorkItem(ctx context.Context, key string, cmd application.CancelWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "cancel_work_item", key, cmd)
}
func (c *Client) ClaimWorkItem(ctx context.Context, key string, cmd application.ClaimWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "claim_work_item", key, cmd)
}
func (c *Client) CompleteWorkItem(ctx context.Context, key string, cmd application.CompleteWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "complete_work_item", key, cmd)
}
func (c *Client) ConfigureTrigger(ctx context.Context, key string, cmd application.ConfigureTriggerCommand) (CommandResult[domain.Trigger], error) {
	return command[domain.Trigger](ctx, c, "configure_trigger", key, cmd)
}
func (c *Client) CreateBlocker(ctx context.Context, key string, cmd application.CreateBlockerCommand) (CommandResult[domain.Blocker], error) {
	return command[domain.Blocker](ctx, c, "create_blocker", key, cmd)
}
func (c *Client) CreateEvidenceLink(ctx context.Context, key string, cmd application.CreateEvidenceLinkCommand) (CommandResult[domain.EvidenceLink], error) {
	return command[domain.EvidenceLink](ctx, c, "create_evidence_link", key, cmd)
}
func (c *Client) CreateIssue(ctx context.Context, key string, cmd application.CreateIssueCommand) (CommandResult[domain.Issue], error) {
	return command[domain.Issue](ctx, c, "create_issue", key, cmd)
}
func (c *Client) CreateObjective(ctx context.Context, key string, cmd application.CreateObjectiveCommand) (CommandResult[domain.Objective], error) {
	return command[domain.Objective](ctx, c, "create_objective", key, cmd)
}
func (c *Client) CreateOutcome(ctx context.Context, key string, cmd application.CreateOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "create_outcome", key, cmd)
}
func (c *Client) CreateRoadmap(ctx context.Context, key string, cmd application.CreateRoadmapCommand) (CommandResult[domain.Roadmap], error) {
	return command[domain.Roadmap](ctx, c, "create_roadmap", key, cmd)
}
func (c *Client) CreateWorkItem(ctx context.Context, key string, cmd application.CreateWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "create_work_item", key, cmd)
}
func (c *Client) DeactivateRoadmapRevision(ctx context.Context, key string, cmd application.DeactivateRoadmapRevisionCommand) (CommandResult[domain.RoadmapActivationRecord], error) {
	return command[domain.RoadmapActivationRecord](ctx, c, "deactivate_roadmap_revision", key, cmd)
}
func (c *Client) DeferWorkItem(ctx context.Context, key string, cmd application.DeferWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "defer_work_item", key, cmd)
}
func (c *Client) DiscardRoadmapDraft(ctx context.Context, key string, cmd application.DiscardRoadmapDraftCommand) (CommandResult[domain.Roadmap], error) {
	return command[domain.Roadmap](ctx, c, "discard_roadmap_draft", key, cmd)
}
func (c *Client) FailOutcome(ctx context.Context, key string, cmd application.FailOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "fail_outcome", key, cmd)
}
func (c *Client) FinalizeWorkContract(ctx context.Context, key string, cmd application.FinalizeWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "finalize_work_contract", key, cmd)
}
func (c *Client) InterveneSignedReviewCase(ctx context.Context, key string, cmd application.InterveneSignedReviewCaseCommand) (CommandResult[application.SignedInterventionResult], error) {
	return command[application.SignedInterventionResult](ctx, c, "intervene_signed_review_case", key, cmd)
}
func (c *Client) InvestigateIssue(ctx context.Context, key string, cmd application.InvestigateIssueCommand) (CommandResult[domain.Issue], error) {
	return command[domain.Issue](ctx, c, "investigate_issue", key, cmd)
}
func (c *Client) LinkExternalReference(ctx context.Context, key string, cmd application.LinkExternalReferenceCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "link_external_reference", key, cmd)
}
func (c *Client) MarkIssueDuplicate(ctx context.Context, key string, cmd application.MarkIssueDuplicateCommand) (CommandResult[domain.Issue], error) {
	return command[domain.Issue](ctx, c, "mark_issue_duplicate", key, cmd)
}
func (c *Client) MarkIssueWontFix(ctx context.Context, key string, cmd application.MarkIssueWontFixCommand) (CommandResult[domain.Issue], error) {
	return command[domain.Issue](ctx, c, "mark_issue_wont_fix", key, cmd)
}
func (c *Client) OpenRoadmapDraft(ctx context.Context, key string, cmd application.OpenRoadmapDraftCommand) (CommandResult[domain.Roadmap], error) {
	return command[domain.Roadmap](ctx, c, "open_roadmap_draft", key, cmd)
}
func (c *Client) ProposeDecision(ctx context.Context, key string, cmd application.ProposeDecisionCommand) (CommandResult[domain.Decision], error) {
	return command[domain.Decision](ctx, c, "propose_decision", key, cmd)
}
func (c *Client) PublishRoadmapDraft(ctx context.Context, key string, cmd application.PublishRoadmapDraftCommand) (CommandResult[domain.Roadmap], error) {
	return command[domain.Roadmap](ctx, c, "publish_roadmap_draft", key, cmd)
}
func (c *Client) ReclaimWorkItem(ctx context.Context, key string, cmd application.ReclaimWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "reclaim_work_item", key, cmd)
}
func (c *Client) ReconcileExpiredWorkContracts(ctx context.Context, key string, cmd application.ReconcileExpiredWorkContractsCommand) (CommandResult[application.ReconciledWorkContracts], error) {
	return command[application.ReconciledWorkContracts](ctx, c, "reconcile_expired_work_contracts", key, cmd)
}
func (c *Client) RecordCriterionAssessment(ctx context.Context, key string, cmd application.RecordCriterionAssessmentCommand) (CommandResult[domain.CriterionAssessment], error) {
	return command[domain.CriterionAssessment](ctx, c, "record_criterion_assessment", key, cmd)
}
func (c *Client) RedeliverDelivery(ctx context.Context, key string, cmd application.RedeliverDeliveryCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "redeliver_delivery", key, cmd)
}
func (c *Client) RegisterArtifact(ctx context.Context, key string, cmd application.RegisterArtifactCommand) (CommandResult[domain.Artifact], error) {
	return command[domain.Artifact](ctx, c, "register_artifact", key, cmd)
}
func (c *Client) RegisterEvidence(ctx context.Context, key string, cmd application.RegisterEvidenceCommand) (CommandResult[domain.Evidence], error) {
	return command[domain.Evidence](ctx, c, "register_evidence", key, cmd)
}
func (c *Client) RejectDecision(ctx context.Context, key string, cmd application.RejectDecisionCommand) (CommandResult[domain.Decision], error) {
	return command[domain.Decision](ctx, c, "reject_decision", key, cmd)
}
func (c *Client) ReleaseWorkItem(ctx context.Context, key string, cmd application.ReleaseWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "release_work_item", key, cmd)
}
func (c *Client) RemoveDependency(ctx context.Context, key string, cmd application.RemoveDependencyCommand) (CommandResult[domain.Relation], error) {
	return command[domain.Relation](ctx, c, "remove_dependency", key, cmd)
}
func (c *Client) RemoveExternalReference(ctx context.Context, key string, cmd application.RemoveExternalReferenceCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "remove_external_reference", key, cmd)
}
func (c *Client) RenewSignedReviewContract(ctx context.Context, key string, cmd application.RenewSignedReviewContractCommand) (CommandResult[application.SignedReviewContractResult], error) {
	return command[application.SignedReviewContractResult](ctx, c, "renew_signed_review_contract", key, cmd)
}
func (c *Client) RenewSignedWorkContract(ctx context.Context, key string, cmd application.RenewSignedWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "renew_signed_work_contract", key, cmd)
}
func (c *Client) RenewWorkContract(ctx context.Context, key string, cmd application.RenewWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "renew_work_contract", key, cmd)
}
func (c *Client) RenewWorkItemLease(ctx context.Context, key string, cmd application.RenewWorkItemLeaseCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "renew_work_item_lease", key, cmd)
}
func (c *Client) ReopenIssue(ctx context.Context, key string, cmd application.ReopenIssueCommand) (CommandResult[domain.Issue], error) {
	return command[domain.Issue](ctx, c, "reopen_issue", key, cmd)
}
func (c *Client) ReopenObjective(ctx context.Context, key string, cmd application.ReopenObjectiveCommand) (CommandResult[domain.Objective], error) {
	return command[domain.Objective](ctx, c, "reopen_objective", key, cmd)
}
func (c *Client) ReopenOutcome(ctx context.Context, key string, cmd application.ReopenOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "reopen_outcome", key, cmd)
}
func (c *Client) ReopenRoadmap(ctx context.Context, key string, cmd application.ReopenRoadmapCommand) (CommandResult[domain.Roadmap], error) {
	return command[domain.Roadmap](ctx, c, "reopen_roadmap", key, cmd)
}
func (c *Client) ReopenWorkItem(ctx context.Context, key string, cmd application.ReopenWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "reopen_work_item", key, cmd)
}
func (c *Client) ReplaceExternalContext(ctx context.Context, key string, cmd application.ReplaceExternalContextCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "replace_external_context", key, cmd)
}
func (c *Client) ReplaceRoadmapDraft(ctx context.Context, key string, cmd application.ReplaceRoadmapDraftCommand) (CommandResult[domain.Roadmap], error) {
	return command[domain.Roadmap](ctx, c, "replace_roadmap_draft", key, cmd)
}
func (c *Client) ReportIssueWithBlocker(ctx context.Context, key string, cmd application.ReportIssueWithBlockerCommand) (CommandResult[application.ReportIssueWithBlockerResult], error) {
	return command[application.ReportIssueWithBlockerResult](ctx, c, "report_issue_with_blocker", key, cmd)
}
func (c *Client) ResolveBlocker(ctx context.Context, key string, cmd application.ResolveBlockerCommand) (CommandResult[domain.Blocker], error) {
	return command[domain.Blocker](ctx, c, "resolve_blocker", key, cmd)
}
func (c *Client) ResolveIssue(ctx context.Context, key string, cmd application.ResolveIssueCommand) (CommandResult[domain.Issue], error) {
	return command[domain.Issue](ctx, c, "resolve_issue", key, cmd)
}
func (c *Client) ResolveIssueAndBlockers(ctx context.Context, key string, cmd application.ResolveIssueAndBlockersCommand) (CommandResult[application.ResolveIssueAndBlockersResult], error) {
	return command[application.ResolveIssueAndBlockersResult](ctx, c, "resolve_issue_and_blockers", key, cmd)
}
func (c *Client) ResumeSignedReviewContract(ctx context.Context, key string, cmd application.ResumeSignedReviewContractCommand) (CommandResult[application.SignedReviewContractResult], error) {
	return command[application.SignedReviewContractResult](ctx, c, "resume_signed_review_contract", key, cmd)
}
func (c *Client) ResumeSignedWorkContract(ctx context.Context, key string, cmd application.ResumeSignedWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "resume_signed_work_contract", key, cmd)
}
func (c *Client) ResumeWorkContract(ctx context.Context, key string, cmd application.ResumeWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "resume_work_contract", key, cmd)
}
func (c *Client) RetireCriterion(ctx context.Context, key string, cmd application.RetireCriterionCommand) (CommandResult[domain.SuccessCriterion], error) {
	return command[domain.SuccessCriterion](ctx, c, "retire_criterion", key, cmd)
}
func (c *Client) RetractEvidence(ctx context.Context, key string, cmd application.RetractEvidenceCommand) (CommandResult[domain.Evidence], error) {
	return command[domain.Evidence](ctx, c, "retract_evidence", key, cmd)
}
func (c *Client) RetractEvidenceLink(ctx context.Context, key string, cmd application.RetractEvidenceLinkCommand) (CommandResult[domain.EvidenceLink], error) {
	return command[domain.EvidenceLink](ctx, c, "retract_evidence_link", key, cmd)
}
func (c *Client) ReturnSignedReview(ctx context.Context, key string, cmd application.ReturnSignedReviewCommand) (CommandResult[application.SignedReturnResult], error) {
	return command[application.SignedReturnResult](ctx, c, "return_signed_review", key, cmd)
}
func (c *Client) ReturnSignedWork(ctx context.Context, key string, cmd application.ReturnSignedWorkCommand) (CommandResult[application.SignedReturnResult], error) {
	return command[application.SignedReturnResult](ctx, c, "return_signed_work", key, cmd)
}
func (c *Client) ReviseCriterion(ctx context.Context, key string, cmd application.ReviseCriterionCommand) (CommandResult[domain.SuccessCriterion], error) {
	return command[domain.SuccessCriterion](ctx, c, "revise_criterion", key, cmd)
}
func (c *Client) RevokeSignedContract(ctx context.Context, key string, cmd application.RevokeSignedContractCommand) (CommandResult[application.SignedInterventionResult], error) {
	return command[application.SignedInterventionResult](ctx, c, "revoke_signed_contract", key, cmd)
}
func (c *Client) RevokeWorkContract(ctx context.Context, key string, cmd application.RevokeWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "revoke_work_contract", key, cmd)
}
func (c *Client) SetNamespaceWorkProtocol(ctx context.Context, key string, cmd application.SetNamespaceWorkProtocolCommand) (CommandResult[application.NamespaceWorkProtocolResult], error) {
	return command[application.NamespaceWorkProtocolResult](ctx, c, "set_namespace_work_protocol", key, cmd)
}
func (c *Client) SetObjectiveOwners(ctx context.Context, key string, cmd application.SetObjectiveOwnersCommand) (CommandResult[domain.Objective], error) {
	return command[domain.Objective](ctx, c, "set_objective_owners", key, cmd)
}
func (c *Client) SetOutcomeOwners(ctx context.Context, key string, cmd application.SetOutcomeOwnersCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "set_outcome_owners", key, cmd)
}
func (c *Client) SetTriggerEnabled(ctx context.Context, key string, cmd application.SetTriggerEnabledCommand) (CommandResult[domain.Trigger], error) {
	return command[domain.Trigger](ctx, c, "set_trigger_enabled", key, cmd)
}
func (c *Client) SetWorkItemAssignees(ctx context.Context, key string, cmd application.SetWorkItemAssigneesCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "set_work_item_assignees", key, cmd)
}
func (c *Client) StartObjective(ctx context.Context, key string, cmd application.StartObjectiveCommand) (CommandResult[domain.Objective], error) {
	return command[domain.Objective](ctx, c, "start_objective", key, cmd)
}
func (c *Client) SubmitWorkResult(ctx context.Context, key string, cmd application.SubmitWorkResultCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "submit_work_result", key, cmd)
}
func (c *Client) SupersedeDecision(ctx context.Context, key string, cmd application.SupersedeDecisionCommand) (CommandResult[application.DecisionSupersessionResult], error) {
	return command[application.DecisionSupersessionResult](ctx, c, "supersede_decision", key, cmd)
}
func (c *Client) SyncWorkContract(ctx context.Context, key string, cmd application.SyncWorkContractCommand) (CommandResult[application.WorkContractResult], error) {
	return command[application.WorkContractResult](ctx, c, "sync_work_contract", key, cmd)
}
func (c *Client) UnarchiveOutcome(ctx context.Context, key string, cmd application.UnarchiveOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "unarchive_outcome", key, cmd)
}
func (c *Client) UpdateBlockerDescription(ctx context.Context, key string, cmd application.UpdateBlockerDescriptionCommand) (CommandResult[domain.Blocker], error) {
	return command[domain.Blocker](ctx, c, "update_blocker_description", key, cmd)
}
func (c *Client) UpdateDecision(ctx context.Context, key string, cmd application.UpdateDecisionCommand) (CommandResult[domain.Decision], error) {
	return command[domain.Decision](ctx, c, "update_decision", key, cmd)
}
func (c *Client) UpdateIssue(ctx context.Context, key string, cmd application.UpdateIssueCommand) (CommandResult[domain.Issue], error) {
	return command[domain.Issue](ctx, c, "update_issue", key, cmd)
}
func (c *Client) UpdateObjective(ctx context.Context, key string, cmd application.UpdateObjectiveCommand) (CommandResult[domain.Objective], error) {
	return command[domain.Objective](ctx, c, "update_objective", key, cmd)
}
func (c *Client) UpdateOutcome(ctx context.Context, key string, cmd application.UpdateOutcomeCommand) (CommandResult[domain.Outcome], error) {
	return command[domain.Outcome](ctx, c, "update_outcome", key, cmd)
}
func (c *Client) UpdateTrigger(ctx context.Context, key string, cmd application.UpdateTriggerCommand) (CommandResult[domain.Trigger], error) {
	return command[domain.Trigger](ctx, c, "update_trigger", key, cmd)
}
func (c *Client) UpdateWorkItem(ctx context.Context, key string, cmd application.UpdateWorkItemCommand) (CommandResult[domain.WorkItem], error) {
	return command[domain.WorkItem](ctx, c, "update_work_item", key, cmd)
}
func (c *Client) WithdrawArtifact(ctx context.Context, key string, cmd application.WithdrawArtifactCommand) (CommandResult[domain.Artifact], error) {
	return command[domain.Artifact](ctx, c, "withdraw_artifact", key, cmd)
}
