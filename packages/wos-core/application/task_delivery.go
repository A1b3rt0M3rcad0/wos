package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// SubmitTaskForReviewCommand atomically ends the current execution authority
// and records a TaskDelivery. Exactly one of the legacy lease or contracts_v1
// authority/material pairs must be supplied. Signed-v2 delivery remains on
// its separate signed return protocol.
type SubmitTaskForReviewCommand struct {
	Scope                   domain.Scope   `json:"scope"`
	WorkItemID              domain.ID      `json:"work_item_id"`
	ExpectedWorkItemVersion domain.Version `json:"expected_work_item_version,string"`

	ClaimID        *domain.ID                   `wos:"optional" json:"claim_id,omitempty"`
	FencingToken   uint64                       `wos:"optional" json:"fencing_token,omitempty,string"`
	LegacyMaterial *domain.TaskDeliveryMaterial `wos:"optional" json:"legacy_material,omitempty"`

	ContractID              *domain.ID                 `wos:"optional" json:"contract_id,omitempty"`
	Authority               *ContractAuthority         `wos:"optional" json:"authority,omitempty"`
	ExpectedContractVersion *domain.Version            `wos:"optional" json:"expected_contract_version,omitempty,string"`
	SubmissionMaterial      *domain.WorkResultMaterial `wos:"optional" json:"submission_material,omitempty"`
	SupersedesSubmissionID  *domain.ID                 `wos:"optional" json:"supersedes_submission_id,omitempty"`
}

type TaskDeliveryResult struct {
	WorkItem                domain.WorkItem        `json:"work_item"`
	Delivery                domain.TaskDelivery    `json:"delivery"`
	Contract                *domain.WorkContract   `json:"contract,omitempty"`
	Submission              *domain.WorkSubmission `json:"submission,omitempty"`
	ExpectedWorkItemVersion domain.Version         `json:"expected_work_item_version,string"`
	ExpectedContractVersion domain.Version         `json:"expected_contract_version,omitempty,string"`
	EvaluatedAt             time.Time              `json:"evaluated_at"`
}

func (s *Service) SubmitTaskForReview(ctx context.Context, cc domain.CommandContext, cmd SubmitTaskForReviewCommand) (MutationResult[TaskDeliveryResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[TaskDeliveryResult]{}, err
	}
	if strings.TrimSpace(cc.IdempotencyKey) == "" {
		return MutationResult[TaskDeliveryResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "Task delivery submission requires a non-empty idempotency key")
	}
	if err := validateSubmitTaskForReviewCommand(cmd); err != nil {
		return MutationResult[TaskDeliveryResult]{}, err
	}
	encoded, err := json.Marshal(cmd)
	if err != nil {
		return MutationResult[TaskDeliveryResult]{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "delivery command cannot be encoded", err)
	}
	if len(encoded) > 256*1024 {
		return MutationResult[TaskDeliveryResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "delivery command exceeds 256 KiB")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (TaskDeliveryResult, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, u, cmd.Scope); err != nil {
			return TaskDeliveryResult{}, 0, err
		}
		deliveryUOW, ok := u.(ports.TaskDeliveryUnitOfWork)
		if !ok {
			return TaskDeliveryResult{}, 0, domain.NewError(domain.ErrorCodeInvalidConfig, "adapter lacks TaskDelivery persistence")
		}
		pending, err := deliveryUOW.TaskDeliveries().Pending(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return TaskDeliveryResult{}, 0, err
		}
		if pending != nil {
			return TaskDeliveryResult{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "Task already has a delivery awaiting review")
		}

		if cmd.ContractID != nil {
			return s.submitContractsV1TaskForReview(ctx, u, deliveryUOW.TaskDeliveries(), cc, cmd)
		}
		return s.submitLegacyTaskForReview(ctx, u, deliveryUOW.TaskDeliveries(), cc, cmd)
	})
}

func validateSubmitTaskForReviewCommand(cmd SubmitTaskForReviewCommand) error {
	if err := cmd.Scope.Validate(); err != nil {
		return err
	}
	if err := cmd.WorkItemID.Validate(); err != nil {
		return err
	}
	if err := cmd.ExpectedWorkItemVersion.Validate(); err != nil {
		return err
	}
	if cmd.ContractID == nil {
		if cmd.ClaimID == nil || cmd.ClaimID.Validate() != nil || cmd.FencingToken == 0 || cmd.LegacyMaterial == nil || cmd.Authority != nil || cmd.ExpectedContractVersion != nil || cmd.SubmissionMaterial != nil || cmd.SupersedesSubmissionID != nil {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "legacy delivery requires only a claim, fencing token, and inline material")
		}
		return nil
	}
	if err := cmd.ContractID.Validate(); err != nil {
		return err
	}
	if cmd.ClaimID != nil || cmd.FencingToken != 0 || cmd.LegacyMaterial != nil || cmd.Authority == nil || cmd.ExpectedContractVersion == nil || cmd.SubmissionMaterial == nil || cmd.SupersedesSubmissionID != nil && cmd.SupersedesSubmissionID.Validate() != nil {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "contracts_v1 delivery requires only a contract authority and submission material")
	}
	if err := cmd.ExpectedContractVersion.Validate(); err != nil {
		return err
	}
	if cmd.Authority.ExecutionID.Validate() != nil || cmd.Authority.FencingToken == 0 || strings.TrimSpace(cmd.Authority.SpecDigest) == "" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "contracts_v1 delivery requires exact execution authority")
	}
	return nil
}

func (s *Service) submitLegacyTaskForReview(
	ctx context.Context,
	u ports.UnitOfWork,
	deliveries ports.TaskDeliveryRepository,
	cc domain.CommandContext,
	cmd SubmitTaskForReviewCommand,
) (TaskDeliveryResult, domain.OutcomeRevision, error) {
	var zero TaskDeliveryResult
	w, err := u.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
	if err != nil {
		return zero, 0, err
	}
	if w.Version != cmd.ExpectedWorkItemVersion {
		return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "Task version conflict")
	}
	if w.ContractsEnabled || w.CurrentContractID != nil || w.CurrentLease == nil {
		return zero, 0, domain.NewError(domain.ErrorCodeContractProtocolRequired, "legacy delivery requires the current legacy lease protocol")
	}
	if w.CurrentLease.ClaimID != *cmd.ClaimID || w.CurrentLease.FencingToken != cmd.FencingToken {
		return zero, 0, domain.NewError(domain.ErrorCodeLease, "delivery claim or fencing token is stale")
	}
	now, err := s.transactionTime(ctx, u)
	if err != nil {
		return zero, 0, err
	}
	credentialAuthority, err := s.resolveLeaseCredentialAuthority(ctx, u, cc, cmd.Scope, w.CurrentLease, "SubmitTaskForReview", now)
	if err != nil {
		return zero, 0, err
	}
	if credentialAuthority == nil {
		return zero, 0, leaseCredentialBindingUnavailable()
	}
	if err := requireNotBlocked(ctx, u, w.Ref()); err != nil {
		return zero, 0, err
	}
	if err := requireHardDependenciesSatisfied(ctx, u, w.Ref()); err != nil {
		return zero, 0, err
	}
	material := domain.NormalizeTaskDeliveryMaterial(*cmd.LegacyMaterial)
	if err := validateSubmittedMaterial(ctx, u, w, domain.WorkResultMaterial{
		Summary: material.Summary, Artifacts: material.Artifacts,
		EvidenceIDs: material.EvidenceIDs, CriterionEvidence: material.CriterionEvidence,
	}); err != nil {
		return zero, 0, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	credentialID := credentialIDPointer(credentialAuthority.holderID)
	delivery, err := domain.NewLegacyTaskDelivery(id, w, material, cc.PrincipalID, credentialID, now)
	if err != nil {
		return zero, 0, err
	}
	before := w.Version
	if err := w.RecordTaskDelivery(delivery, nil, nil, before, now); err != nil {
		return zero, 0, err
	}
	if err := deliveries.Insert(ctx, delivery); err != nil {
		return zero, 0, err
	}
	if err := u.WorkItems().Save(ctx, w, before); err != nil {
		return zero, 0, err
	}
	revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
	return TaskDeliveryResult{WorkItem: w, Delivery: delivery, ExpectedWorkItemVersion: before, EvaluatedAt: now}, revision, err
}

func (s *Service) submitContractsV1TaskForReview(
	ctx context.Context,
	u ports.UnitOfWork,
	deliveries ports.TaskDeliveryRepository,
	cc domain.CommandContext,
	cmd SubmitTaskForReviewCommand,
) (TaskDeliveryResult, domain.OutcomeRevision, error) {
	var zero TaskDeliveryResult
	repo, contract, w, credentialAuthority, now, err := s.contractForWrite(ctx, u, cc, cmd.Scope, *cmd.ContractID, *cmd.Authority, "SubmitTaskForReview")
	if err != nil {
		return zero, 0, err
	}
	if w.ID != cmd.WorkItemID {
		return zero, 0, domain.NewError(domain.ErrorCodeInvalidScope, "contract does not belong to the requested Task")
	}
	if w.Version != cmd.ExpectedWorkItemVersion {
		return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "Task version conflict")
	}
	if contract.Version != *cmd.ExpectedContractVersion {
		return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "contract version conflict")
	}
	if contract.SignedBinding != nil {
		return zero, 0, domain.NewError(domain.ErrorCodeSignedProtocolRequired, "signed-v2 delivery uses the signed return protocol")
	}
	if !w.ContractsEnabled || contract.CredentialBindingVersion != domain.WorkContractCredentialBindingV1 || credentialAuthority == nil || credentialAuthority.holderID != contract.HolderCredentialID {
		return zero, 0, leaseCredentialBindingUnavailable()
	}
	if err := requireNotBlocked(ctx, u, w.Ref()); err != nil {
		return zero, 0, err
	}
	if err := requireHardDependenciesSatisfied(ctx, u, w.Ref()); err != nil {
		return zero, 0, err
	}
	material := domain.NormalizeResultMaterial(*cmd.SubmissionMaterial)
	if err := validateSubmittedMaterial(ctx, u, w, material); err != nil {
		return zero, 0, err
	}
	digest, err := domain.SemanticDigest(material)
	if err != nil {
		return zero, 0, err
	}
	submissionID, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	submission := domain.WorkSubmission{
		ID: submissionID, Scope: cmd.Scope, Material: material, Digest: digest,
		PrincipalID: cc.PrincipalID, Actor: cc.Actor, SubmittedAt: now,
		SupersedesSubmissionID: cloneIDPointer(cmd.SupersedesSubmissionID),
	}
	contractVersion, leaseVersion, workVersion := contract.Version, contract.LeaseVersion, w.Version
	if err := contract.SubmitForCredential(submission, cc.PrincipalID, credentialAuthority.holderID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, *cmd.ExpectedContractVersion, now); err != nil {
		return zero, 0, err
	}
	if err := repo.InsertSubmission(ctx, submission); err != nil {
		return zero, 0, err
	}
	submittedContractVersion := contract.Version
	if err := repo.Save(ctx, contract, contractVersion, leaseVersion); err != nil {
		return zero, 0, err
	}
	deliveryID, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	credentialID := credentialIDPointer(credentialAuthority.holderID)
	delivery, err := domain.NewContractsV1TaskDelivery(deliveryID, w, contract, submission, cc.PrincipalID, credentialID, now)
	if err != nil {
		return zero, 0, err
	}
	if err := contract.DeliverV1ForCredential(submission, cc.PrincipalID, credentialAuthority.holderID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, contract.Version, now); err != nil {
		return zero, 0, err
	}
	if err := w.RecordTaskDelivery(delivery, &contract, &submission, workVersion, now); err != nil {
		return zero, 0, err
	}
	if err := repo.Save(ctx, contract, submittedContractVersion, leaseVersion); err != nil {
		return zero, 0, err
	}
	if err := deliveries.Insert(ctx, delivery); err != nil {
		return zero, 0, err
	}
	if err := u.WorkItems().Save(ctx, w, workVersion); err != nil {
		return zero, 0, err
	}
	revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
	contractCopy := contract
	submissionCopy := submission
	return TaskDeliveryResult{
		WorkItem: w, Delivery: delivery, Contract: &contractCopy, Submission: &submissionCopy,
		ExpectedWorkItemVersion: workVersion, ExpectedContractVersion: *cmd.ExpectedContractVersion, EvaluatedAt: now,
	}, revision, err
}

func taskDeliveryCommandEvents[T any](s *Service, cc domain.CommandContext, meta commandMetadata, value T, revision domain.OutcomeRevision) ([]domain.DomainEvent, bool, error) {
	if withdrawal, ok := any(value).(TaskDeliveryWithdrawalResult); ok && meta.Name == "WithdrawTaskDelivery" {
		before := withdrawal.WorkItem.Version - 1
		spec := compoundEventSpec{eventType: "task_delivery.withdrawn", ref: withdrawal.WorkItem.Ref(), before: versionPtr(before), after: versionPtr(withdrawal.WorkItem.Version)}
		events, err := buildCompoundEvents(s, cc, meta, revision, []compoundEventSpec{spec})
		if err != nil {
			return nil, true, err
		}
		payload := map[string]any{
			"withdrawal_id": withdrawal.Withdrawal.ID, "delivery_id": withdrawal.Delivery.ID,
			"work_item_id": withdrawal.WorkItem.ID, "delivery_digest": withdrawal.Delivery.Digest,
			"round": withdrawal.Delivery.Round,
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, true, err
		}
		events[0].Payload = encoded
		events[0].RecordedAt = withdrawal.EvaluatedAt
		if err := events[0].Validate(); err != nil {
			return nil, true, err
		}
		return events, true, nil
	}
	result, ok := any(value).(TaskDeliveryResult)
	if !ok || meta.Name != "SubmitTaskForReview" {
		return nil, false, nil
	}
	workBefore := result.ExpectedWorkItemVersion
	specs := make([]compoundEventSpec, 0, 3)
	if result.Contract != nil && result.Submission != nil {
		contractBefore := result.ExpectedContractVersion
		submittedVersion, err := contractBefore.Next()
		if err != nil {
			return nil, true, err
		}
		specs = append(specs,
			compoundEventSpec{eventType: "work_contract.result_submitted", ref: result.Contract.Ref(), before: versionPtr(contractBefore), after: versionPtr(submittedVersion)},
			compoundEventSpec{eventType: "work_contract.execution_closed_for_delivery", ref: result.Contract.Ref(), before: versionPtr(submittedVersion), after: versionPtr(result.Contract.Version)},
		)
	}
	specs = append(specs, compoundEventSpec{eventType: "task_delivery.submitted", ref: result.WorkItem.Ref(), before: versionPtr(workBefore), after: versionPtr(result.WorkItem.Version)})
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, true, err
	}
	for i := range events {
		var payload any
		switch events[i].EventType {
		case "work_contract.result_submitted":
			payload = map[string]any{"contract_id": result.Contract.ID, "work_item_id": result.WorkItem.ID, "submission_id": result.Submission.ID, "submission_digest": result.Submission.Digest, "contract_version": result.Contract.Version}
		case "work_contract.execution_closed_for_delivery":
			payload = map[string]any{"contract_id": result.Contract.ID, "work_item_id": result.WorkItem.ID, "status": result.Contract.Status, "close_reason": result.Contract.CloseReason, "closed_at": result.Contract.ClosedAt, "contract_version": result.Contract.Version}
		default:
			payload = map[string]any{"delivery_id": result.Delivery.ID, "work_item_id": result.WorkItem.ID, "round": result.Delivery.Round, "delivery_digest": result.Delivery.Digest, "authority_kind": result.Delivery.Authority.Kind, "submission_id": taskDeliverySubmissionID(result.Delivery)}
		}
		events[i].Payload, err = json.Marshal(payload)
		if err != nil {
			return nil, true, err
		}
		events[i].RecordedAt = result.EvaluatedAt
		if err := events[i].Validate(); err != nil {
			return nil, true, err
		}
	}
	return events, true, nil
}

func taskDeliverySubmissionID(delivery domain.TaskDelivery) *domain.ID {
	if delivery.SubmissionRef == nil {
		return nil
	}
	id := delivery.SubmissionRef.ID
	return &id
}
