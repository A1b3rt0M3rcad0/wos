// Code generated from public Application command signatures; DO NOT EDIT.
package mcptransport

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerCommands(server *mcp.Server, service *application.Service, ids ports.IDGenerator, options Options) {
	registerCommand[application.AbandonOutcomeCommand](server, ids, options, "wos_abandon_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.AbandonOutcomeCommand) (any, error) {
		return service.AbandonOutcome(ctx, cc, cmd)
	})
	registerCommand[application.AcceptDecisionCommand](server, ids, options, "wos_accept_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.AcceptDecisionCommand) (any, error) {
		return service.AcceptDecision(ctx, cc, cmd)
	})
	registerCommand[application.AchieveObjectiveCommand](server, ids, options, "wos_achieve_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.AchieveObjectiveCommand) (any, error) {
		return service.AchieveObjective(ctx, cc, cmd)
	})
	registerCommand[application.AchieveOutcomeCommand](server, ids, options, "wos_achieve_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.AchieveOutcomeCommand) (any, error) {
		return service.AchieveOutcome(ctx, cc, cmd)
	})
	registerCommand[application.AcquireNextWorkContractCommand](server, ids, options, "wos_acquire_next_work_contract", func(ctx context.Context, cc domain.CommandContext, cmd application.AcquireNextWorkContractCommand) (any, error) {
		return service.AcquireNextWorkContract(ctx, cc, cmd)
	})
	registerCommand[application.AcquireWorkContractCommand](server, ids, options, "wos_acquire_work_contract", func(ctx context.Context, cc domain.CommandContext, cmd application.AcquireWorkContractCommand) (any, error) {
		return service.AcquireWorkContract(ctx, cc, cmd)
	})
	registerCommand[application.ActivateOutcomeCommand](server, ids, options, "wos_activate_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.ActivateOutcomeCommand) (any, error) {
		return service.ActivateOutcome(ctx, cc, cmd)
	})
	registerCommand[application.ActivateRoadmapRevisionCommand](server, ids, options, "wos_activate_roadmap_revision", func(ctx context.Context, cc domain.CommandContext, cmd application.ActivateRoadmapRevisionCommand) (any, error) {
		return service.ActivateRoadmapRevision(ctx, cc, cmd)
	})
	registerCommand[application.ActivateWorkItemCommand](server, ids, options, "wos_activate_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ActivateWorkItemCommand) (any, error) {
		return service.ActivateWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.AddCriterionCommand](server, ids, options, "wos_add_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.AddCriterionCommand) (any, error) {
		return service.AddCriterion(ctx, cc, cmd)
	})
	registerCommand[application.AddDependencyCommand](server, ids, options, "wos_add_dependency", func(ctx context.Context, cc domain.CommandContext, cmd application.AddDependencyCommand) (any, error) {
		return service.AddDependency(ctx, cc, cmd)
	})
	registerCommand[application.AdministrativeCancelWorkItemCommand](server, ids, options, "wos_administrative_cancel_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.AdministrativeCancelWorkItemCommand) (any, error) {
		return service.AdministrativeCancelWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.AdministrativeCompleteWorkItemCommand](server, ids, options, "wos_administrative_complete_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.AdministrativeCompleteWorkItemCommand) (any, error) {
		return service.AdministrativeCompleteWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.ArchiveOutcomeCommand](server, ids, options, "wos_archive_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.ArchiveOutcomeCommand) (any, error) {
		return service.ArchiveOutcome(ctx, cc, cmd)
	})
	registerCommand[application.ArchiveRoadmapCommand](server, ids, options, "wos_archive_roadmap", func(ctx context.Context, cc domain.CommandContext, cmd application.ArchiveRoadmapCommand) (any, error) {
		return service.ArchiveRoadmap(ctx, cc, cmd)
	})
	registerCommand[application.AttestCriterionCommand](server, ids, options, "wos_attest_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.AttestCriterionCommand) (any, error) {
		return service.AttestCriterion(ctx, cc, cmd)
	})
	registerCommand[application.CancelBlockerCommand](server, ids, options, "wos_cancel_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.CancelBlockerCommand) (any, error) {
		return service.CancelBlocker(ctx, cc, cmd)
	})
	registerCommand[application.CancelObjectiveCommand](server, ids, options, "wos_cancel_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.CancelObjectiveCommand) (any, error) {
		return service.CancelObjective(ctx, cc, cmd)
	})
	registerCommand[application.CancelWorkItemCommand](server, ids, options, "wos_cancel_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.CancelWorkItemCommand) (any, error) {
		return service.CancelWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.ClaimWorkItemCommand](server, ids, options, "wos_claim_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ClaimWorkItemCommand) (any, error) {
		return service.ClaimWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.CompleteWorkItemCommand](server, ids, options, "wos_complete_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.CompleteWorkItemCommand) (any, error) {
		return service.CompleteWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.ConfigureTriggerCommand](server, ids, options, "wos_configure_trigger", func(ctx context.Context, cc domain.CommandContext, cmd application.ConfigureTriggerCommand) (any, error) {
		return service.ConfigureTrigger(ctx, cc, cmd)
	})
	registerCommand[application.CreateBlockerCommand](server, ids, options, "wos_create_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateBlockerCommand) (any, error) {
		return service.CreateBlocker(ctx, cc, cmd)
	})
	registerCommand[application.CreateEvidenceLinkCommand](server, ids, options, "wos_create_evidence_link", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateEvidenceLinkCommand) (any, error) {
		return service.CreateEvidenceLink(ctx, cc, cmd)
	})
	registerCommand[application.CreateIssueCommand](server, ids, options, "wos_create_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateIssueCommand) (any, error) {
		return service.CreateIssue(ctx, cc, cmd)
	})
	registerCommand[application.CreateObjectiveCommand](server, ids, options, "wos_create_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateObjectiveCommand) (any, error) {
		return service.CreateObjective(ctx, cc, cmd)
	})
	registerCommand[application.CreateOutcomeCommand](server, ids, options, "wos_create_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateOutcomeCommand) (any, error) {
		return service.CreateOutcome(ctx, cc, cmd)
	})
	registerCommand[application.CreateRoadmapCommand](server, ids, options, "wos_create_roadmap", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateRoadmapCommand) (any, error) {
		return service.CreateRoadmap(ctx, cc, cmd)
	})
	registerCommand[application.CreateWorkItemCommand](server, ids, options, "wos_create_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.CreateWorkItemCommand) (any, error) {
		return service.CreateWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.DeactivateRoadmapRevisionCommand](server, ids, options, "wos_deactivate_roadmap_revision", func(ctx context.Context, cc domain.CommandContext, cmd application.DeactivateRoadmapRevisionCommand) (any, error) {
		return service.DeactivateRoadmapRevision(ctx, cc, cmd)
	})
	registerCommand[application.DeferWorkItemCommand](server, ids, options, "wos_defer_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.DeferWorkItemCommand) (any, error) {
		return service.DeferWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.DiscardRoadmapDraftCommand](server, ids, options, "wos_discard_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.DiscardRoadmapDraftCommand) (any, error) {
		return service.DiscardRoadmapDraft(ctx, cc, cmd)
	})
	registerCommand[application.FailOutcomeCommand](server, ids, options, "wos_fail_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.FailOutcomeCommand) (any, error) {
		return service.FailOutcome(ctx, cc, cmd)
	})
	registerCommand[application.FinalizeWorkContractCommand](server, ids, options, "wos_finalize_work_contract", func(ctx context.Context, cc domain.CommandContext, cmd application.FinalizeWorkContractCommand) (any, error) {
		return service.FinalizeWorkContract(ctx, cc, cmd)
	})
	registerCommand[application.InvestigateIssueCommand](server, ids, options, "wos_investigate_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.InvestigateIssueCommand) (any, error) {
		return service.InvestigateIssue(ctx, cc, cmd)
	})
	registerCommand[application.LinkExternalReferenceCommand](server, ids, options, "wos_link_external_reference", func(ctx context.Context, cc domain.CommandContext, cmd application.LinkExternalReferenceCommand) (any, error) {
		return service.LinkExternalReference(ctx, cc, cmd)
	})
	registerCommand[application.MarkIssueDuplicateCommand](server, ids, options, "wos_mark_issue_duplicate", func(ctx context.Context, cc domain.CommandContext, cmd application.MarkIssueDuplicateCommand) (any, error) {
		return service.MarkIssueDuplicate(ctx, cc, cmd)
	})
	registerCommand[application.MarkIssueWontFixCommand](server, ids, options, "wos_mark_issue_wont_fix", func(ctx context.Context, cc domain.CommandContext, cmd application.MarkIssueWontFixCommand) (any, error) {
		return service.MarkIssueWontFix(ctx, cc, cmd)
	})
	registerCommand[application.OpenRoadmapDraftCommand](server, ids, options, "wos_open_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.OpenRoadmapDraftCommand) (any, error) {
		return service.OpenRoadmapDraft(ctx, cc, cmd)
	})
	registerCommand[application.ProposeDecisionCommand](server, ids, options, "wos_propose_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.ProposeDecisionCommand) (any, error) {
		return service.ProposeDecision(ctx, cc, cmd)
	})
	registerCommand[application.PublishRoadmapDraftCommand](server, ids, options, "wos_publish_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.PublishRoadmapDraftCommand) (any, error) {
		return service.PublishRoadmapDraft(ctx, cc, cmd)
	})
	registerCommand[application.ReclaimWorkItemCommand](server, ids, options, "wos_reclaim_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ReclaimWorkItemCommand) (any, error) {
		return service.ReclaimWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.ReconcileExpiredWorkContractsCommand](server, ids, options, "wos_reconcile_expired_work_contracts", func(ctx context.Context, cc domain.CommandContext, cmd application.ReconcileExpiredWorkContractsCommand) (any, error) {
		return service.ReconcileExpiredWorkContracts(ctx, cc, cmd)
	})
	registerCommand[application.RecordCriterionAssessmentCommand](server, ids, options, "wos_record_criterion_assessment", func(ctx context.Context, cc domain.CommandContext, cmd application.RecordCriterionAssessmentCommand) (any, error) {
		return service.RecordCriterionAssessment(ctx, cc, cmd)
	})
	registerCommand[application.RedeliverDeliveryCommand](server, ids, options, "wos_redeliver_delivery", func(ctx context.Context, cc domain.CommandContext, cmd application.RedeliverDeliveryCommand) (any, error) {
		return service.RedeliverDelivery(ctx, cc, cmd)
	})
	registerCommand[application.RegisterArtifactCommand](server, ids, options, "wos_register_artifact", func(ctx context.Context, cc domain.CommandContext, cmd application.RegisterArtifactCommand) (any, error) {
		return service.RegisterArtifact(ctx, cc, cmd)
	})
	registerCommand[application.RegisterEvidenceCommand](server, ids, options, "wos_register_evidence", func(ctx context.Context, cc domain.CommandContext, cmd application.RegisterEvidenceCommand) (any, error) {
		return service.RegisterEvidence(ctx, cc, cmd)
	})
	registerCommand[application.RejectDecisionCommand](server, ids, options, "wos_reject_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.RejectDecisionCommand) (any, error) {
		return service.RejectDecision(ctx, cc, cmd)
	})
	registerCommand[application.ReleaseWorkItemCommand](server, ids, options, "wos_release_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ReleaseWorkItemCommand) (any, error) {
		return service.ReleaseWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.RemoveDependencyCommand](server, ids, options, "wos_remove_dependency", func(ctx context.Context, cc domain.CommandContext, cmd application.RemoveDependencyCommand) (any, error) {
		return service.RemoveDependency(ctx, cc, cmd)
	})
	registerCommand[application.RemoveExternalReferenceCommand](server, ids, options, "wos_remove_external_reference", func(ctx context.Context, cc domain.CommandContext, cmd application.RemoveExternalReferenceCommand) (any, error) {
		return service.RemoveExternalReference(ctx, cc, cmd)
	})
	registerCommand[application.RenewWorkContractCommand](server, ids, options, "wos_renew_work_contract", func(ctx context.Context, cc domain.CommandContext, cmd application.RenewWorkContractCommand) (any, error) {
		return service.RenewWorkContract(ctx, cc, cmd)
	})
	registerCommand[application.RenewWorkItemLeaseCommand](server, ids, options, "wos_renew_work_item_lease", func(ctx context.Context, cc domain.CommandContext, cmd application.RenewWorkItemLeaseCommand) (any, error) {
		return service.RenewWorkItemLease(ctx, cc, cmd)
	})
	registerCommand[application.ReopenIssueCommand](server, ids, options, "wos_reopen_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenIssueCommand) (any, error) {
		return service.ReopenIssue(ctx, cc, cmd)
	})
	registerCommand[application.ReopenObjectiveCommand](server, ids, options, "wos_reopen_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenObjectiveCommand) (any, error) {
		return service.ReopenObjective(ctx, cc, cmd)
	})
	registerCommand[application.ReopenOutcomeCommand](server, ids, options, "wos_reopen_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenOutcomeCommand) (any, error) {
		return service.ReopenOutcome(ctx, cc, cmd)
	})
	registerCommand[application.ReopenRoadmapCommand](server, ids, options, "wos_reopen_roadmap", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenRoadmapCommand) (any, error) {
		return service.ReopenRoadmap(ctx, cc, cmd)
	})
	registerCommand[application.ReopenWorkItemCommand](server, ids, options, "wos_reopen_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.ReopenWorkItemCommand) (any, error) {
		return service.ReopenWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.ReplaceExternalContextCommand](server, ids, options, "wos_replace_external_context", func(ctx context.Context, cc domain.CommandContext, cmd application.ReplaceExternalContextCommand) (any, error) {
		return service.ReplaceExternalContext(ctx, cc, cmd)
	})
	registerCommand[application.ReplaceRoadmapDraftCommand](server, ids, options, "wos_replace_roadmap_draft", func(ctx context.Context, cc domain.CommandContext, cmd application.ReplaceRoadmapDraftCommand) (any, error) {
		return service.ReplaceRoadmapDraft(ctx, cc, cmd)
	})
	registerCommand[application.ReportIssueWithBlockerCommand](server, ids, options, "wos_report_issue_with_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.ReportIssueWithBlockerCommand) (any, error) {
		return service.ReportIssueWithBlocker(ctx, cc, cmd)
	})
	registerCommand[application.ResolveBlockerCommand](server, ids, options, "wos_resolve_blocker", func(ctx context.Context, cc domain.CommandContext, cmd application.ResolveBlockerCommand) (any, error) {
		return service.ResolveBlocker(ctx, cc, cmd)
	})
	registerCommand[application.ResolveIssueCommand](server, ids, options, "wos_resolve_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.ResolveIssueCommand) (any, error) {
		return service.ResolveIssue(ctx, cc, cmd)
	})
	registerCommand[application.ResolveIssueAndBlockersCommand](server, ids, options, "wos_resolve_issue_and_blockers", func(ctx context.Context, cc domain.CommandContext, cmd application.ResolveIssueAndBlockersCommand) (any, error) {
		return service.ResolveIssueAndBlockers(ctx, cc, cmd)
	})
	registerCommand[application.ResumeWorkContractCommand](server, ids, options, "wos_resume_work_contract", func(ctx context.Context, cc domain.CommandContext, cmd application.ResumeWorkContractCommand) (any, error) {
		return service.ResumeWorkContract(ctx, cc, cmd)
	})
	registerCommand[application.RetireCriterionCommand](server, ids, options, "wos_retire_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.RetireCriterionCommand) (any, error) {
		return service.RetireCriterion(ctx, cc, cmd)
	})
	registerCommand[application.RetractEvidenceCommand](server, ids, options, "wos_retract_evidence", func(ctx context.Context, cc domain.CommandContext, cmd application.RetractEvidenceCommand) (any, error) {
		return service.RetractEvidence(ctx, cc, cmd)
	})
	registerCommand[application.RetractEvidenceLinkCommand](server, ids, options, "wos_retract_evidence_link", func(ctx context.Context, cc domain.CommandContext, cmd application.RetractEvidenceLinkCommand) (any, error) {
		return service.RetractEvidenceLink(ctx, cc, cmd)
	})
	registerCommand[application.ReviseCriterionCommand](server, ids, options, "wos_revise_criterion", func(ctx context.Context, cc domain.CommandContext, cmd application.ReviseCriterionCommand) (any, error) {
		return service.ReviseCriterion(ctx, cc, cmd)
	})
	registerCommand[application.RevokeWorkContractCommand](server, ids, options, "wos_revoke_work_contract", func(ctx context.Context, cc domain.CommandContext, cmd application.RevokeWorkContractCommand) (any, error) {
		return service.RevokeWorkContract(ctx, cc, cmd)
	})
	registerCommand[application.SetNamespaceWorkProtocolCommand](server, ids, options, "wos_set_namespace_work_protocol", func(ctx context.Context, cc domain.CommandContext, cmd application.SetNamespaceWorkProtocolCommand) (any, error) {
		return service.SetNamespaceWorkProtocol(ctx, cc, cmd)
	})
	registerCommand[application.SetObjectiveOwnersCommand](server, ids, options, "wos_set_objective_owners", func(ctx context.Context, cc domain.CommandContext, cmd application.SetObjectiveOwnersCommand) (any, error) {
		return service.SetObjectiveOwners(ctx, cc, cmd)
	})
	registerCommand[application.SetOutcomeOwnersCommand](server, ids, options, "wos_set_outcome_owners", func(ctx context.Context, cc domain.CommandContext, cmd application.SetOutcomeOwnersCommand) (any, error) {
		return service.SetOutcomeOwners(ctx, cc, cmd)
	})
	registerCommand[application.SetTriggerEnabledCommand](server, ids, options, "wos_set_trigger_enabled", func(ctx context.Context, cc domain.CommandContext, cmd application.SetTriggerEnabledCommand) (any, error) {
		return service.SetTriggerEnabled(ctx, cc, cmd)
	})
	registerCommand[application.SetWorkItemAssigneesCommand](server, ids, options, "wos_set_work_item_assignees", func(ctx context.Context, cc domain.CommandContext, cmd application.SetWorkItemAssigneesCommand) (any, error) {
		return service.SetWorkItemAssignees(ctx, cc, cmd)
	})
	registerCommand[application.StartObjectiveCommand](server, ids, options, "wos_start_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.StartObjectiveCommand) (any, error) {
		return service.StartObjective(ctx, cc, cmd)
	})
	registerCommand[application.SubmitWorkResultCommand](server, ids, options, "wos_submit_work_result", func(ctx context.Context, cc domain.CommandContext, cmd application.SubmitWorkResultCommand) (any, error) {
		return service.SubmitWorkResult(ctx, cc, cmd)
	})
	registerCommand[application.SupersedeDecisionCommand](server, ids, options, "wos_supersede_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.SupersedeDecisionCommand) (any, error) {
		return service.SupersedeDecision(ctx, cc, cmd)
	})
	registerCommand[application.SyncWorkContractCommand](server, ids, options, "wos_sync_work_contract", func(ctx context.Context, cc domain.CommandContext, cmd application.SyncWorkContractCommand) (any, error) {
		return service.SyncWorkContract(ctx, cc, cmd)
	})
	registerCommand[application.UnarchiveOutcomeCommand](server, ids, options, "wos_unarchive_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.UnarchiveOutcomeCommand) (any, error) {
		return service.UnarchiveOutcome(ctx, cc, cmd)
	})
	registerCommand[application.UpdateBlockerDescriptionCommand](server, ids, options, "wos_update_blocker_description", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateBlockerDescriptionCommand) (any, error) {
		return service.UpdateBlockerDescription(ctx, cc, cmd)
	})
	registerCommand[application.UpdateDecisionCommand](server, ids, options, "wos_update_decision", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateDecisionCommand) (any, error) {
		return service.UpdateDecision(ctx, cc, cmd)
	})
	registerCommand[application.UpdateIssueCommand](server, ids, options, "wos_update_issue", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateIssueCommand) (any, error) {
		return service.UpdateIssue(ctx, cc, cmd)
	})
	registerCommand[application.UpdateObjectiveCommand](server, ids, options, "wos_update_objective", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateObjectiveCommand) (any, error) {
		return service.UpdateObjective(ctx, cc, cmd)
	})
	registerCommand[application.UpdateOutcomeCommand](server, ids, options, "wos_update_outcome", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateOutcomeCommand) (any, error) {
		return service.UpdateOutcome(ctx, cc, cmd)
	})
	registerCommand[application.UpdateTriggerCommand](server, ids, options, "wos_update_trigger", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateTriggerCommand) (any, error) {
		return service.UpdateTrigger(ctx, cc, cmd)
	})
	registerCommand[application.UpdateWorkItemCommand](server, ids, options, "wos_update_work_item", func(ctx context.Context, cc domain.CommandContext, cmd application.UpdateWorkItemCommand) (any, error) {
		return service.UpdateWorkItem(ctx, cc, cmd)
	})
	registerCommand[application.WithdrawArtifactCommand](server, ids, options, "wos_withdraw_artifact", func(ctx context.Context, cc domain.CommandContext, cmd application.WithdrawArtifactCommand) (any, error) {
		return service.WithdrawArtifact(ctx, cc, cmd)
	})
}
