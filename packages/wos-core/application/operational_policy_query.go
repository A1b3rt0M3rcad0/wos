package application

import (
	"context"
	"sort"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type OperationalPolicyFindingSeverity string

const (
	OperationalPolicyFindingInfo    OperationalPolicyFindingSeverity = "info"
	OperationalPolicyFindingBlocked OperationalPolicyFindingSeverity = "blocked"
)

type OperationalPolicyFinding struct {
	Code     string                           `json:"code"`
	Severity OperationalPolicyFindingSeverity `json:"severity"`
	Ref      *d.EntityRef                     `json:"ref,omitempty"`
	Message  string                           `json:"message"`
	Remedy   string                           `json:"remedy"`
}

type OperationalPolicyCriterionSnapshot struct {
	ID                   d.ID                `json:"id"`
	Revision             d.CriterionRevision `json:"criterion_revision"`
	Title                string              `json:"title"`
	Required             bool                `json:"required"`
	VerificationMode     d.VerificationMode  `json:"verification_mode"`
	Status               d.CriterionStatus   `json:"status"`
	AssessmentCount      int                 `json:"assessment_count"`
	HasCurrentAssessment bool                `json:"has_current_assessment"`
}

type OperationalPolicyOutcomeSnapshot struct {
	Ref                d.EntityRef                          `json:"ref"`
	Version            d.Version                            `json:"version,string"`
	Lifecycle          d.OutcomeLifecycle                   `json:"lifecycle"`
	OperationalProfile d.OperationalProfile                 `json:"operational_profile"`
	CompletionPolicy   d.CompletionPolicy                   `json:"completion_policy"`
	Archived           bool                                 `json:"archived"`
	Owners             []d.ActorRef                         `json:"owner_refs"`
	Criteria           []OperationalPolicyCriterionSnapshot `json:"criteria"`
}

type OperationalPolicyObjectiveSnapshot struct {
	Ref                d.EntityRef                          `json:"ref"`
	Version            d.Version                            `json:"version,string"`
	Lifecycle          d.ObjectiveLifecycle                 `json:"lifecycle"`
	ParentObjectiveID  *d.ID                                `json:"parent_objective_id,omitempty"`
	RequiredForOutcome bool                                 `json:"required_for_outcome"`
	CompletionMode     d.ObjectiveCompletionMode            `json:"completion_mode"`
	CompletionCycle    uint64                               `json:"completion_cycle,string"`
	Owners             []d.ActorRef                         `json:"owner_refs"`
	Criteria           []OperationalPolicyCriterionSnapshot `json:"criteria"`
}

type OperationalPolicyTaskSnapshot struct {
	Ref               d.EntityRef                          `json:"ref"`
	Version           d.Version                            `json:"version,string"`
	Lifecycle         d.WorkItemLifecycle                  `json:"lifecycle"`
	ObjectiveID       *d.ID                                `json:"objective_id,omitempty"`
	Assignees         []d.ActorRef                         `json:"assignee_refs"`
	CurrentLease      *OperationalPolicyLeaseSnapshot      `json:"current_lease,omitempty"`
	CurrentContractID *d.ID                                `json:"current_contract_id,omitempty"`
	Criteria          []OperationalPolicyCriterionSnapshot `json:"criteria"`
}

type OperationalPolicyLeaseSnapshot struct {
	ClaimID         d.ID       `json:"claim_id"`
	PrincipalID     string     `json:"principal_id"`
	Actor           d.ActorRef `json:"actor_ref"`
	AcquiredAt      time.Time  `json:"acquired_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	Active          bool       `json:"active"`
	CredentialBound bool       `json:"credential_bound"`
}

type OperationalPolicyAuthoritySnapshot struct {
	Kind              string           `json:"kind"`
	ID                d.ID             `json:"id"`
	WorkItemRef       d.EntityRef      `json:"work_item_ref"`
	Status            d.ContractStatus `json:"status"`
	HolderPrincipalID string           `json:"holder_principal_id"`
	CredentialID      *d.ID            `json:"credential_id,omitempty"`
	CredentialBound   bool             `json:"credential_bound"`
	Active            bool             `json:"active"`
	ExpiresAt         time.Time        `json:"expires_at"`
}

// OperationalProfileMigrationPreview is an authorized, read-only snapshot.
// It reports the exact policy change under consideration and any records that
// need resolution before the migration may proceed.
type OperationalProfileMigrationPreview struct {
	Scope                    d.Scope                              `json:"scope"`
	Outcome                  OperationalPolicyOutcomeSnapshot     `json:"outcome"`
	OutcomeRevision          d.OutcomeRevision                    `json:"outcome_revision,string"`
	TargetOperationalProfile d.OperationalProfile                 `json:"target_operational_profile"`
	TargetCompletionPolicy   d.CompletionPolicy                   `json:"target_completion_policy"`
	Objectives               []OperationalPolicyObjectiveSnapshot `json:"objectives"`
	Tasks                    []OperationalPolicyTaskSnapshot      `json:"tasks"`
	Authorities              []OperationalPolicyAuthoritySnapshot `json:"authorities"`
	Findings                 []OperationalPolicyFinding           `json:"findings"`
	CanMigrate               bool                                 `json:"can_migrate"`
}

// PreviewOutcomeOperationalProfileMigration reads every relevant fact from a
// single UOW snapshot and never writes state, emits events or reserves an
// idempotency receipt. It is deliberately not registered as an API capability
// until operational_v2's write gates are complete.
func (s *Service) PreviewOutcomeOperationalProfileMigration(ctx context.Context, scope d.Scope) (ReadResult[OperationalProfileMigrationPreview], error) {
	if err := scope.Validate(); err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return ReadResult[OperationalProfileMigrationPreview]{}, d.NewError(d.ErrorCodeForbidden, "authenticated namespace administrator identity is required")
	}
	if err := authorizeOperationalPolicyAdmin(ctx, s.authorizer, scope, identity.PrincipalID); err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	defer uow.Rollback()
	if err := authorizeOperationalPolicyAdminInUnitOfWork(ctx, s.authorizer, uow, scope, identity.PrincipalID); err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	reads, ok := uow.(ports.ContinuityReadUnitOfWork)
	if !ok {
		return ReadResult[OperationalProfileMigrationPreview]{}, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks a read-only Outcome revision snapshot for operational policy audit")
	}
	workspace, ok := reads.ContinuityReads().(ports.WorkspaceReadRepository)
	if !ok {
		return ReadResult[OperationalProfileMigrationPreview]{}, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks a read-only Outcome revision snapshot for operational policy audit")
	}
	revision, err := workspace.WorkspaceRevision(ctx, scope)
	if err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	preview, err := buildOperationalProfileMigrationPreview(ctx, uow, outcome, revision, now)
	if err != nil {
		return ReadResult[OperationalProfileMigrationPreview]{}, err
	}
	return ReadResult[OperationalProfileMigrationPreview]{Value: preview, OutcomeRevision: revision}, nil
}

func buildOperationalProfileMigrationPreview(ctx context.Context, uow ports.UnitOfWork, outcome d.Outcome, revision d.OutcomeRevision, now time.Time) (OperationalProfileMigrationPreview, error) {
	var zero OperationalProfileMigrationPreview
	source, ok := uow.(ports.OperationalPolicyAuditUnitOfWork)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks the coherent work-authority repositories required for operational policy audit")
	}
	preview := OperationalProfileMigrationPreview{
		Scope:                    outcome.Scope(),
		OutcomeRevision:          revision,
		TargetOperationalProfile: d.OperationalProfileOperationalV2,
		TargetCompletionPolicy:   d.CompletionPolicyStrictWork,
		Objectives:               []OperationalPolicyObjectiveSnapshot{},
		Tasks:                    []OperationalPolicyTaskSnapshot{},
		Authorities:              []OperationalPolicyAuthoritySnapshot{},
		Findings:                 []OperationalPolicyFinding{},
	}
	preview.Outcome = OperationalPolicyOutcomeSnapshot{
		Ref:                outcome.Ref(),
		Version:            outcome.Version,
		Lifecycle:          outcome.Lifecycle,
		OperationalProfile: outcome.OperationalProfile.Effective(),
		CompletionPolicy:   outcome.CompletionPolicy.Effective(),
		Archived:           outcome.IsArchived(),
		Owners:             append([]d.ActorRef{}, outcome.OwnerRefs...),
		Criteria:           summarizeOperationalPolicyCriteria(outcome.Criteria),
	}
	if preview.Outcome.OperationalProfile != d.OperationalProfileCompatibilityV1 {
		addOperationalFinding(&preview, "profile_not_compatibility_v1", OperationalPolicyFindingBlocked, &preview.Outcome.Ref, "Only compatibility_v1 Outcomes can use this migration.", "Select the supported upgrade or recovery procedure for the current profile.")
	}
	if outcome.IsArchived() || outcome.Lifecycle == d.OutcomeLifecycleAchieved || outcome.Lifecycle == d.OutcomeLifecycleFailed || outcome.Lifecycle == d.OutcomeLifecycleAbandoned {
		addOperationalFinding(&preview, "outcome_not_open", OperationalPolicyFindingBlocked, &preview.Outcome.Ref, "Archived or terminal Outcomes cannot change operational profile.", "Use the authorized unarchive or reopen operation where supported, then run the preview again.")
	}
	if len(outcome.OwnerRefs) == 0 {
		addOperationalFinding(&preview, "outcome_owner_missing", OperationalPolicyFindingInfo, &preview.Outcome.Ref, "No Outcome owners are recorded.", "Assign the intended Outcome owners through the existing owner operation; migration will not assign owners.")
	}
	if len(activeRequiredCriteria(outcome.Criteria)) == 0 {
		addOperationalFinding(&preview, "outcome_required_criteria_missing", OperationalPolicyFindingInfo, &preview.Outcome.Ref, "The Outcome has no active required success criteria.", "Define required criteria before starting work; migration will not create criteria.")
	}

	objectives, err := uow.Objectives().ListByOutcome(ctx, outcome.Scope())
	if err != nil {
		return zero, err
	}
	for _, objective := range objectives {
		snapshot := OperationalPolicyObjectiveSnapshot{
			Ref:                objective.Ref(),
			Version:            objective.Version,
			Lifecycle:          objective.Lifecycle,
			ParentObjectiveID:  cloneIDPointer(objective.ParentObjectiveID),
			RequiredForOutcome: objective.RequiredForOutcome,
			CompletionMode:     objective.CompletionMode.Effective(),
			CompletionCycle:    objective.EffectiveCompletionCycle(),
			Owners:             append([]d.ActorRef{}, objective.OwnerRefs...),
			Criteria:           summarizeOperationalPolicyCriteria(objective.Criteria),
		}
		preview.Objectives = append(preview.Objectives, snapshot)
		if len(objective.OwnerRefs) == 0 {
			ref := objective.Ref()
			addOperationalFinding(&preview, "objective_owner_missing", OperationalPolicyFindingInfo, &ref, "This Objective has no recorded owners.", "Assign an owner if the operating model requires one; migration will not assign owners.")
		}
		if len(activeRequiredCriteria(objective.Criteria)) == 0 {
			ref := objective.Ref()
			addOperationalFinding(&preview, "objective_required_criteria_missing", OperationalPolicyFindingInfo, &ref, "This Objective has no active required success criteria.", "Define the required criteria before starting the Objective; migration will not create them.")
		}
		if snapshot.CompletionMode == d.ObjectiveCompletionModeAutomatic {
			ref := objective.Ref()
			addOperationalFinding(&preview, "objective_automatic_completion_requires_separate_delegation", OperationalPolicyFindingBlocked, &ref, "An existing automatic completion mode cannot be reinterpreted by an Outcome profile migration.", "Resolve this through the separately authorized Objective completion policy procedure; this migration never changes completion modes.")
		}
	}

	workItems, err := uow.WorkItems().ListByOutcome(ctx, outcome.Scope())
	if err != nil {
		return zero, err
	}
	for _, item := range workItems {
		snapshot := OperationalPolicyTaskSnapshot{
			Ref:               item.Ref(),
			Version:           item.Version,
			Lifecycle:         item.Lifecycle,
			ObjectiveID:       cloneIDPointer(item.ObjectiveID),
			Assignees:         append([]d.ActorRef{}, item.AssigneeRefs...),
			CurrentContractID: cloneIDPointer(item.CurrentContractID),
			Criteria:          summarizeOperationalPolicyCriteria(item.Criteria),
		}
		if item.CurrentLease != nil {
			lease := item.CurrentLease
			snapshot.CurrentLease = &OperationalPolicyLeaseSnapshot{
				ClaimID: lease.ClaimID, PrincipalID: lease.PrincipalID, Actor: lease.Actor,
				AcquiredAt: lease.AcquiredAt, ExpiresAt: lease.ExpiresAt,
				Active: lease.ValidAt(now), CredentialBound: false,
			}
			ref := item.Ref()
			addOperationalFinding(&preview, "legacy_lease_unreconciled", OperationalPolicyFindingBlocked, &ref, "This Task retains a legacy lease that has no credential binding.", "Release or explicitly reconcile the lease under the existing authority protocol; do not infer a credential from the Principal.")
		}
		if len(item.AssigneeRefs) == 0 {
			ref := item.Ref()
			addOperationalFinding(&preview, "task_assignee_missing", OperationalPolicyFindingInfo, &ref, "This Task has no recorded assignee.", "Assign an intended participant if needed; migration will not assign Tasks.")
		}
		if len(activeRequiredCriteria(item.Criteria)) == 0 {
			ref := item.Ref()
			addOperationalFinding(&preview, "task_required_criteria_missing", OperationalPolicyFindingInfo, &ref, "This Task has no active required success criteria.", "Define the required criteria before activation; migration will not create them.")
		}
		preview.Tasks = append(preview.Tasks, snapshot)
	}

	if err := appendOperationalAuthorityAudit(ctx, source, outcome.Scope(), workItems, now, &preview); err != nil {
		return zero, err
	}
	sort.Slice(preview.Objectives, func(i, j int) bool { return preview.Objectives[i].Ref.ID < preview.Objectives[j].Ref.ID })
	sort.Slice(preview.Tasks, func(i, j int) bool { return preview.Tasks[i].Ref.ID < preview.Tasks[j].Ref.ID })
	sort.Slice(preview.Authorities, func(i, j int) bool {
		if preview.Authorities[i].Kind == preview.Authorities[j].Kind {
			return preview.Authorities[i].ID < preview.Authorities[j].ID
		}
		return preview.Authorities[i].Kind < preview.Authorities[j].Kind
	})
	sort.Slice(preview.Findings, func(i, j int) bool {
		left, right := preview.Findings[i], preview.Findings[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		leftRef, rightRef := "", ""
		if left.Ref != nil {
			leftRef = left.Ref.Kind.String() + ":" + left.Ref.ID.String()
		}
		if right.Ref != nil {
			rightRef = right.Ref.Kind.String() + ":" + right.Ref.ID.String()
		}
		return leftRef < rightRef
	})
	preview.CanMigrate = true
	for _, finding := range preview.Findings {
		if finding.Severity == OperationalPolicyFindingBlocked {
			preview.CanMigrate = false
			break
		}
	}
	return preview, nil
}

func appendOperationalAuthorityAudit(ctx context.Context, uow ports.OperationalPolicyAuditUnitOfWork, scope d.Scope, workItems []d.WorkItem, now time.Time, preview *OperationalProfileMigrationPreview) error {
	workItemRefs := make(map[d.ID]d.EntityRef, len(workItems))
	currentContracts := make(map[d.ID]bool, len(workItems))
	for _, item := range workItems {
		workItemRefs[item.ID] = item.Ref()
		if item.CurrentContractID != nil {
			currentContracts[*item.CurrentContractID] = true
		}
	}
	legacyRepo := uow.WorkContracts()
	signedRepo := uow.SignedWorkContracts()
	legacyContracts, err := listOperationalWorkContracts(ctx, legacyRepo, scope)
	if err != nil {
		return err
	}
	signedContracts, err := listOperationalWorkContracts(ctx, signedRepo, scope)
	if err != nil {
		return err
	}
	seenContracts := make(map[d.ID]bool, len(legacyContracts)+len(signedContracts))
	for _, c := range legacyContracts {
		seenContracts[c.ID] = true
		if c.Status != d.ContractActive && !currentContracts[c.ID] {
			continue
		}
		ref, ok := workItemRefs[c.WorkItemID]
		if !ok {
			ref = c.Ref()
		}
		active := c.ValidAt(now)
		bound := c.SignedBinding != nil && c.SignedBinding.CredentialID.Validate() == nil
		var credentialID *d.ID
		if bound {
			credentialID = cloneIDPointer(&c.SignedBinding.CredentialID)
		}
		preview.Authorities = append(preview.Authorities, OperationalPolicyAuthoritySnapshot{Kind: "work_contract", ID: c.ID, WorkItemRef: ref, Status: c.Status, HolderPrincipalID: c.HolderPrincipalID, CredentialID: credentialID, CredentialBound: bound, Active: active, ExpiresAt: c.ExpiresAt})
		if c.Status == d.ContractActive || currentContracts[c.ID] {
			code, message, remedy := "work_contract_in_flight", "A work contract is still active or attached to a Task.", "Finish, revoke, or explicitly reconcile the contract under its existing protocol, then rerun the preview."
			if !bound {
				code, message, remedy = "unsigned_work_authority_unbound", "An active unsigned work contract has no credential binding.", "Close or explicitly reconcile the contract; do not infer a credential from its Principal."
			}
			addOperationalFinding(preview, code, OperationalPolicyFindingBlocked, &ref, message, remedy)
		}
	}
	for _, c := range signedContracts {
		if seenContracts[c.ID] {
			continue
		}
		if c.Status != d.ContractActive && !currentContracts[c.ID] {
			continue
		}
		ref, ok := workItemRefs[c.WorkItemID]
		if !ok {
			ref = c.Ref()
		}
		active := c.ValidAt(now)
		bound := c.SignedBinding != nil && c.SignedBinding.CredentialID.Validate() == nil
		var credentialID *d.ID
		if bound {
			credentialID = cloneIDPointer(&c.SignedBinding.CredentialID)
		}
		preview.Authorities = append(preview.Authorities, OperationalPolicyAuthoritySnapshot{Kind: "signed_work_contract", ID: c.ID, WorkItemRef: ref, Status: c.Status, HolderPrincipalID: c.HolderPrincipalID, CredentialID: credentialID, CredentialBound: bound, Active: active, ExpiresAt: c.ExpiresAt})
		if c.Status == d.ContractActive || currentContracts[c.ID] {
			code, message, remedy := "signed_work_contract_in_flight", "A signed work contract is still active or attached to a Task.", "Finish, revoke, or explicitly reconcile the contract under its existing signed protocol, then rerun the preview."
			if !bound {
				code, message, remedy = "signed_work_authority_binding_invalid", "An active signed work contract has no valid credential binding.", "Resolve the signed contract through protocol recovery; never replace or infer its credential binding."
			}
			addOperationalFinding(preview, code, OperationalPolicyFindingBlocked, &ref, message, remedy)
		}
	}
	for _, item := range workItems {
		if item.CurrentContractID != nil && !seenContracts[*item.CurrentContractID] {
			ref := item.Ref()
			addOperationalFinding(preview, "current_work_contract_unresolved", OperationalPolicyFindingBlocked, &ref, "The Task references a current contract that could not be resolved in this storage snapshot.", "Reconcile the contract reference using the owning protocol before migration.")
		}
	}

	security := uow.SignedContracts()
	openCases, err := listOperationalReviewCases(ctx, security, scope)
	if err != nil {
		return err
	}
	for _, review := range openCases {
		ref := d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: review.WorkItemID}
		preview.Authorities = append(preview.Authorities, OperationalPolicyAuthoritySnapshot{Kind: "open_review_case", ID: review.ID, WorkItemRef: ref, Status: d.ContractActive, Active: true})
		addOperationalFinding(preview, "review_case_in_flight", OperationalPolicyFindingBlocked, &ref, "A delivery review is pending or in progress and must not be reinterpreted during profile migration.", "Complete or explicitly intervene in the existing review case, then rerun the preview.")
	}
	reviewContracts, err := listOperationalReviewContracts(ctx, security, scope)
	if err != nil {
		return err
	}
	for _, review := range reviewContracts {
		if review.Status != d.ContractActive {
			continue
		}
		ref := d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: review.WorkItemID}
		active := review.ValidAt(now)
		bound := review.Binding.CredentialID.Validate() == nil
		var credentialID *d.ID
		if bound {
			credentialID = cloneIDPointer(&review.Binding.CredentialID)
		}
		preview.Authorities = append(preview.Authorities, OperationalPolicyAuthoritySnapshot{Kind: "review_contract", ID: review.ID, WorkItemRef: ref, Status: review.Status, HolderPrincipalID: review.HolderPrincipalID, CredentialID: credentialID, CredentialBound: bound, Active: active, ExpiresAt: review.ExpiresAt})
		code, message, remedy := "review_contract_in_flight", "A review contract is still active and must be closed before profile migration.", "Complete, revoke, or reconcile the review authority using the signed review protocol, then rerun the preview."
		if !bound {
			code, message, remedy = "unsigned_review_authority_unbound", "An active review authority has no valid credential binding.", "Resolve the authority through protocol recovery; never infer or substitute the review credential."
		}
		addOperationalFinding(preview, code, OperationalPolicyFindingBlocked, &ref, message, remedy)
	}
	return nil
}

func listOperationalWorkContracts(ctx context.Context, repo ports.WorkContractRepository, scope d.Scope) ([]d.WorkContract, error) {
	const pageSize = 100
	values := []d.WorkContract{}
	after := d.ID("")
	for {
		page, err := repo.List(ctx, scope, ports.ContractFilter{Status: d.ContractActive, After: after, Limit: pageSize})
		if err != nil {
			return nil, err
		}
		values = append(values, page...)
		if len(page) < pageSize {
			return values, nil
		}
		after = page[len(page)-1].ID
	}
}

func listOperationalReviewCases(ctx context.Context, repo ports.SignedContractRepository, scope d.Scope) ([]d.ReviewCase, error) {
	const pageSize = 100
	values := []d.ReviewCase{}
	after := d.ID("")
	for {
		page, err := repo.Cases(ctx, scope, ports.ReviewFilter{OpenOnly: true, After: after, Limit: pageSize})
		if err != nil {
			return nil, err
		}
		values = append(values, page...)
		if len(page) < pageSize {
			return values, nil
		}
		after = page[len(page)-1].ID
	}
}

func listOperationalReviewContracts(ctx context.Context, repo ports.SignedContractRepository, scope d.Scope) ([]d.ReviewContract, error) {
	const pageSize = 100
	values := []d.ReviewContract{}
	after := d.ID("")
	for {
		page, err := repo.ReviewContracts(ctx, scope, ports.ReviewFilter{Status: string(d.ContractActive), After: after, Limit: pageSize})
		if err != nil {
			return nil, err
		}
		values = append(values, page...)
		if len(page) < pageSize {
			return values, nil
		}
		after = page[len(page)-1].ID
	}
}

func summarizeOperationalPolicyCriteria(criteria d.CriterionSet) []OperationalPolicyCriterionSnapshot {
	result := make([]OperationalPolicyCriterionSnapshot, 0, len(criteria.Items))
	for _, item := range criteria.Items {
		count := 0
		for _, assessment := range criteria.Assessments {
			if assessment.CriterionID == item.ID {
				count++
			}
		}
		_, current := criteria.CurrentAssessments[item.ID]
		result = append(result, OperationalPolicyCriterionSnapshot{ID: item.ID, Revision: item.Revision, Title: item.Title, Required: item.Required, VerificationMode: item.VerificationMode, Status: item.Status, AssessmentCount: count, HasCurrentAssessment: current})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func activeRequiredCriteria(criteria d.CriterionSet) []d.SuccessCriterion {
	result := []d.SuccessCriterion{}
	for _, criterion := range criteria.Items {
		if criterion.Status == d.CriterionStatusActive && criterion.Required {
			result = append(result, criterion)
		}
	}
	return result
}

func addOperationalFinding(preview *OperationalProfileMigrationPreview, code string, severity OperationalPolicyFindingSeverity, ref *d.EntityRef, message, remedy string) {
	for _, existing := range preview.Findings {
		if existing.Code == code && sameOperationalRef(existing.Ref, ref) {
			return
		}
	}
	preview.Findings = append(preview.Findings, OperationalPolicyFinding{Code: code, Severity: severity, Ref: cloneEntityRefPointer(ref), Message: message, Remedy: remedy})
}

func sameOperationalRef(left, right *d.EntityRef) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func cloneEntityRefPointer(value *d.EntityRef) *d.EntityRef {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
