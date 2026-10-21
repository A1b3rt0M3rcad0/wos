package memory

import (
	"context"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type taskVerificationReadRepository struct{ tx *transaction }

func (tx *transaction) TaskVerificationReads() ports.TaskVerificationReadRepository {
	return taskVerificationReadRepository{tx: tx}
}

func (r taskVerificationReadRepository) TaskVerificationFacts(ctx context.Context, scope d.Scope, ids []d.ID) ([]ports.TaskVerificationFacts, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if len(ids) > ports.MaxTaskVerificationBatch {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "Task verification batch exceeds 100 Tasks")
	}
	seen := make(map[d.ID]bool, len(ids))
	out := make([]ports.TaskVerificationFacts, 0, len(ids))
	for _, id := range ids {
		if id.Validate() != nil || seen[id] {
			return nil, d.NewError(d.ErrorCodeInvalidArgument, "Task verification requires distinct valid Task IDs")
		}
		seen[id] = true
		item, ok := r.tx.workItems[entityKey(scope, id)]
		if !ok {
			return nil, d.NewError(d.ErrorCodeNotFound, "Task verification source not found")
		}
		if item.Scope != scope || item.ID != id || item.Validate() != nil || validateTaskDeliveryNavigation(item, r.tx) != nil {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification WorkItem source is inconsistent")
		}
		facts := ports.TaskVerificationFacts{
			WorkItemID: item.ID, Version: item.Version, Lifecycle: item.Lifecycle, ContractsEnabled: item.ContractsEnabled,
			CurrentLease: contractCopy(item.CurrentLease), CurrentContractID: cloneVerificationID(item.CurrentContractID),
			PendingDeliveryID: cloneVerificationID(item.PendingDeliveryID), LatestDeliveryID: cloneVerificationID(item.LatestDeliveryID),
			LatestDeliveryDecisionID: cloneVerificationID(item.LatestDeliveryDecisionID), LatestDeliveryWithdrawalID: cloneVerificationID(item.LatestDeliveryWithdrawalID),
			DeliveryRound: item.DeliveryRound, PendingReviewCaseID: cloneVerificationID(item.PendingReviewCaseID),
			LatestReviewCaseID: cloneVerificationID(item.LatestReviewCaseID), CorrectionReviewCaseID: cloneVerificationID(item.CorrectionReviewCaseID),
		}
		if item.CurrentContractID != nil {
			key := entityKey(scope, *item.CurrentContractID)
			contract, found := r.tx.contracts[key]
			if !found {
				contract, found = r.tx.signedContracts[key]
			}
			if !found || contract.ID != *item.CurrentContractID || contract.Scope != scope || contract.WorkItemID != item.ID || contract.ValidateHeader() != nil {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification execution contract does not match its WorkItem")
			}
			copy := contractCopy(contract)
			facts.ExecutionContract = &copy
		}
		if item.LatestDeliveryID != nil {
			delivery, found := r.tx.taskDeliveries[entityKey(scope, *item.LatestDeliveryID)]
			if !found || delivery.Scope != scope || delivery.WorkItemID != item.ID || delivery.Round != item.DeliveryRound || delivery.Validate() != nil {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification delivery does not match its WorkItem")
			}
			if err := validateMemoryTaskDeliveryMaterial(r.tx, delivery); err != nil {
				return nil, err
			}
			if item.PendingDeliveryID != nil {
				if err := delivery.ValidateForWork(item); err != nil {
					return nil, d.WrapError(d.ErrorCodeInvalidScope, "pending delivery does not match the current Task definition", err)
				}
			}
			df := &ports.TaskVerificationDeliveryFacts{ID: delivery.ID, Scope: delivery.Scope, WorkItemID: delivery.WorkItemID,
				Authority: contractCopy(delivery.Authority), Round: delivery.Round, SupersedesDeliveryID: cloneVerificationID(delivery.SupersedesDeliveryID),
				DefinitionDigest: delivery.DefinitionDigest, Digest: delivery.Digest, Summary: delivery.Summary, DeliveredAt: delivery.DeliveredAt,
				SubmissionRef: contractCopy(delivery.SubmissionRef)}
			if delivery.Material != nil {
				df.ArtifactCount, df.EvidenceCount, df.CriterionCount = len(delivery.Material.Artifacts), len(delivery.Material.EvidenceIDs), len(delivery.Material.CriterionEvidence)
			} else if delivery.SubmissionRef != nil {
				submission, found := r.tx.submissions[entityKey(scope, delivery.SubmissionRef.ID)]
				contract, contractFound := r.tx.contracts[entityKey(scope, delivery.SubmissionRef.ContractID)]
				if !found || !contractFound || submission.Validate() != nil || contract.Validate() != nil || contract.SignedBinding != nil || contract.ID != delivery.SubmissionRef.ContractID || contract.WorkItemID != item.ID || contract.LatestSubmissionID == nil || *contract.LatestSubmissionID != delivery.SubmissionRef.ID || submission.Scope != scope || submission.Material.WorkItemID != item.ID || submission.Material.ContractID != delivery.SubmissionRef.ContractID || submission.Digest != delivery.SubmissionRef.Digest || submission.ProtocolVersion == 2 {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification delivery submission binding is inconsistent")
				}
				df.ArtifactCount, df.EvidenceCount, df.CriterionCount = len(submission.Material.Artifacts), len(submission.Material.EvidenceIDs), len(submission.Material.CriterionEvidence)
			}
			facts.NativeDelivery = df
			if item.LatestDeliveryDecisionID != nil {
				decision, found := r.tx.taskDeliveryDecisions[entityKey(scope, *item.LatestDeliveryDecisionID)]
				if !found || decision.ValidateForDelivery(delivery) != nil {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification decision does not match its delivery")
				}
				copy := contractCopy(decision)
				facts.NativeDecision = &copy
			}
			if item.LatestDeliveryWithdrawalID != nil {
				withdrawal, found := r.tx.taskDeliveryWithdrawals[entityKey(scope, *item.LatestDeliveryWithdrawalID)]
				if !found || withdrawal.ValidateForDelivery(delivery) != nil {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification withdrawal does not match its delivery")
				}
				copy := withdrawal
				facts.NativeWithdrawal = &copy
			}
		}
		caseIDs := []*d.ID{item.PendingReviewCaseID, item.LatestReviewCaseID, item.CorrectionReviewCaseID}
		caseValues := map[d.ID]d.ReviewCase{}
		for _, caseID := range caseIDs {
			if caseID == nil {
				continue
			}
			value, found := r.tx.signedReview.Cases[entityKey(scope, *caseID)]
			if !found || value.ID != *caseID || value.Scope != scope || value.WorkItemID != item.ID || value.Validate() != nil {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification review case does not match its WorkItem")
			}
			caseValues[value.ID] = value
			execution, executionFound := r.tx.signedContracts[entityKey(scope, value.WorkContractID)]
			submission, submissionFound := r.tx.signedSubmissions[entityKey(scope, value.SubmissionID)]
			if !executionFound || !submissionFound || execution.Validate() != nil || execution.SignedBinding == nil || execution.Status != d.ContractDelivered || execution.WorkItemID != item.ID || execution.LatestSubmissionID == nil || *execution.LatestSubmissionID != value.SubmissionID || submission.Validate() != nil || submission.ProtocolVersion != 2 || submission.Material.ContractID != value.WorkContractID || submission.Material.WorkItemID != item.ID || submission.Digest != value.SubmissionDigest {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification signed review case does not bind its exact execution submission")
			}
		}
		selectedCase := item.PendingReviewCaseID
		if selectedCase == nil {
			selectedCase = item.LatestReviewCaseID
		}
		if selectedCase != nil {
			value := caseValues[*selectedCase]
			copy := contractCopy(value)
			facts.SignedReview = &copy
			if value.CurrentContractID != nil {
				reviewContract, found := r.tx.signedReview.Contracts[entityKey(scope, *value.CurrentContractID)]
				if !found || reviewContract.ID != *value.CurrentContractID || reviewContract.Scope != scope || reviewContract.CaseID != value.ID || reviewContract.WorkItemID != item.ID || reviewContract.Validate() != nil {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification review contract does not match its case")
				}
				rc := contractCopy(reviewContract)
				facts.ReviewContract = &rc
			}
		}
		if item.CorrectionReviewCaseID != nil {
			value := caseValues[*item.CorrectionReviewCaseID]
			copy := contractCopy(value)
			facts.SignedCorrection = &copy
		}
		out = append(out, facts)
	}
	return out, nil
}

func cloneVerificationID(value *d.ID) *d.ID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func validateMemoryTaskDeliveryMaterial(tx *transaction, delivery d.TaskDelivery) error {
	validateMaterial := func(material d.WorkResultMaterial) error {
		for _, artifact := range material.Artifacts {
			if _, ok := tx.artifacts[entityKey(delivery.Scope, artifact.ArtifactID)]; !ok {
				return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery artifact reference is unavailable")
			}
		}
		for _, id := range material.EvidenceIDs {
			if _, ok := tx.evidence[entityKey(delivery.Scope, id)]; !ok {
				return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery evidence reference is unavailable")
			}
		}
		for _, link := range material.CriterionEvidence {
			for _, id := range link.EvidenceIDs {
				if _, ok := tx.evidence[entityKey(delivery.Scope, id)]; !ok {
					return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery criterion evidence reference is unavailable")
				}
			}
		}
		return nil
	}
	if delivery.Material != nil {
		if err := validateMaterial(d.WorkResultMaterial{Artifacts: delivery.Material.Artifacts, EvidenceIDs: delivery.Material.EvidenceIDs, CriterionEvidence: delivery.Material.CriterionEvidence}); err != nil {
			return err
		}
	} else if delivery.SubmissionRef != nil {
		submission, ok := tx.submissions[entityKey(delivery.Scope, delivery.SubmissionRef.ID)]
		if !ok {
			submission, ok = tx.signedSubmissions[entityKey(delivery.Scope, delivery.SubmissionRef.ID)]
		}
		if !ok || submission.Validate() != nil || submission.Digest != delivery.SubmissionRef.Digest || submission.Material.ContractID != delivery.SubmissionRef.ContractID || submission.Material.WorkItemID != delivery.WorkItemID {
			return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery submission reference is inconsistent")
		}
		if err := validateMaterial(submission.Material); err != nil {
			return err
		}
	}
	return nil
}

var _ ports.TaskVerificationReadUnitOfWork = (*transaction)(nil)
