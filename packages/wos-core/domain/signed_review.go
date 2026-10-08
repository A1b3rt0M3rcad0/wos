package domain

import (
	"encoding/hex"
	"reflect"
	"strings"
	"time"
)

// SignedContractBinding freezes acquisition policy and operational identity.
// It is public metadata, not proof that an envelope has been verified.
type SignedContractBinding struct {
	ProtocolVersion      int             `json:"protocol_version"`
	SpecificationDigest  string          `json:"specification_digest,omitempty"`
	SeparationGroup      string          `json:"separation_group,omitempty"`
	ServerID             string          `json:"server_id"`
	CredentialID         ID              `json:"credential_id"`
	SignerKeyID          ID              `json:"signer_key_id"`
	AcceptanceFloor      AcceptanceMode  `json:"acceptance_floor"`
	CorrectionFindings   []SignedFinding `json:"correction_findings,omitempty"`
	PolicyRevision       Version         `json:"policy_revision,string"`
	AllowedSigningKeyIDs []ID            `json:"allowed_signing_key_ids"`
	PreviousSubmissionID *ID             `json:"previous_submission_id,omitempty"`
	PreviousReviewCaseID *ID             `json:"previous_review_case_id,omitempty"`
}

func (b SignedContractBinding) Validate() error {
	if len(b.CorrectionFindings) > 100 {
		return NewError(ErrorCodeInvalidArgument, "correction finding limit is 100")
	}
	if len(b.SeparationGroup) > 256 || (b.SpecificationDigest != "" && !ValidSignedDigest(b.SpecificationDigest)) {
		return NewError(ErrorCodeInvalidArgument, "invalid frozen specification/group binding")
	}
	if b.ProtocolVersion != 2 || strings.TrimSpace(b.ServerID) == "" || len(b.ServerID) > 256 || b.CredentialID.Validate() != nil || b.SignerKeyID.Validate() != nil || b.PolicyRevision.Validate() != nil || !b.AcceptanceFloor.Valid() || len(b.AllowedSigningKeyIDs) < 1 || len(b.AllowedSigningKeyIDs) > 100 {
		return NewError(ErrorCodeInvalidArgument, "invalid signed contract binding")
	}
	seen := map[ID]bool{}
	found := false
	for _, id := range b.AllowedSigningKeyIDs {
		if id.Validate() != nil || seen[id] {
			return NewError(ErrorCodeInvalidArgument, "invalid/duplicate signing key scope")
		}
		seen[id] = true
		found = found || id == b.SignerKeyID
	}
	if !found {
		return NewError(ErrorCodeForbidden, "acquisition key is outside frozen signing scope")
	}
	if (b.PreviousSubmissionID == nil) != (b.PreviousReviewCaseID == nil) {
		return NewError(ErrorCodeInvalidArgument, "correction requires both immutable targets")
	}
	if b.PreviousSubmissionID != nil && (b.PreviousSubmissionID.Validate() != nil || b.PreviousReviewCaseID.Validate() != nil) {
		return NewError(ErrorCodeInvalidArgument, "invalid correction targets")
	}
	return nil
}
func ValidSignedDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil && strings.ToLower(value) == value
}

// Deliver closes executor authority without certifying the Task. Its immutable
// submission is subsequently reviewed under a separate ReviewContract.
func (c *WorkContract) Deliver(submission WorkSubmission, principal string, execution ID, fence FencingToken, expected Version, now time.Time) error {
	if c.SignedBinding == nil || c.SignedBinding.ProtocolVersion != 2 {
		return NewError(ErrorCodeSignedProtocolRequired, "delivery requires a signed-v2 contract")
	}
	if err := c.Authorize(principal, execution, fence, now); err != nil {
		return err
	}
	if c.Version != expected {
		return NewError(ErrorCodeVersionConflict, "contract version changed")
	}
	if err := submission.Validate(); err != nil {
		return err
	}
	if submission.Material.ContractID != c.ID || submission.Scope != c.Scope || submission.Material.WorkItemID != c.WorkItemID || submission.Material.SpecDigest != c.SubmissionSpecDigest() || submission.PrincipalID != principal || submission.Actor != c.Actor || submission.SubmittedAt.Before(c.AcquiredAt) || submission.SubmittedAt.After(now) {
		return NewError(ErrorCodeInvalidScope, "submission does not bind this authority")
	}
	if err := c.close(ContractDelivered, "accepted for independent review", principal, now); err != nil {
		return err
	}
	id := submission.ID
	c.LatestSubmissionID = &id
	return c.Validate()
}

type ReviewCaseStatus string

const (
	ReviewPending          ReviewCaseStatus = "pending"
	ReviewInReview         ReviewCaseStatus = "in_review"
	ReviewApproved         ReviewCaseStatus = "approved"
	ReviewChangesRequested ReviewCaseStatus = "changes_requested"
	ReviewCancelled        ReviewCaseStatus = "cancelled"
	ReviewSuperseded       ReviewCaseStatus = "superseded"
)

func (s ReviewCaseStatus) Open() bool { return s == ReviewPending || s == ReviewInReview }
func (s ReviewCaseStatus) Valid() bool {
	return s.Open() || s == ReviewApproved || s == ReviewChangesRequested || s == ReviewCancelled || s == ReviewSuperseded
}

type ExecutionParticipant struct {
	PrincipalID     string `json:"principal_id"`
	SeparationGroup string `json:"separation_group,omitempty"`
}

type ReviewCase struct {
	LatestDecisionID    *ID              `json:"latest_decision_id,omitempty"`
	ID                  ID               `json:"id"`
	Scope               Scope            `json:"scope"`
	WorkItemID          ID               `json:"work_item_id"`
	WorkContractID      ID               `json:"work_contract_id"`
	SubmissionID        ID               `json:"submission_id"`
	SubmissionDigest    string           `json:"submission_digest"`
	IssuedSpecDigest    string           `json:"issued_spec_digest"`
	PolicyRevision      Version          `json:"policy_revision,string"`
	AcceptanceFloor     AcceptanceMode   `json:"acceptance_floor"`
	Round               uint64           `json:"round,string"`
	Version             Version          `json:"version,string"`
	Status              ReviewCaseStatus `json:"status"`
	CurrentContractID   *ID              `json:"current_contract_id,omitempty"`
	LastFencingToken    uint64           `json:"last_fencing_token,string"`
	ExecutionPrincipals []string         `json:"execution_principals"`
	ExecutionGroups     []string         `json:"execution_groups"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	ClosedAt            *time.Time       `json:"closed_at,omitempty"`
	CloseReason         string           `json:"close_reason,omitempty"`
	ClosedBy            string           `json:"closed_by,omitempty"`
}

func (r ReviewCase) Validate() error {
	if r.LatestDecisionID != nil && r.LatestDecisionID.Validate() != nil {
		return NewError(ErrorCodeInvalidArgument, "invalid review decision pointer")
	}
	for _, id := range []ID{r.ID, r.WorkItemID, r.WorkContractID, r.SubmissionID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if r.Scope.Validate() != nil || r.Version.Validate() != nil || r.PolicyRevision.Validate() != nil || r.Round < 1 || !r.Status.Valid() || !ValidSignedDigest(r.SubmissionDigest) || !ValidSignedDigest(r.IssuedSpecDigest) || r.AcceptanceFloor != AcceptanceIndependentReview || len(r.ExecutionPrincipals) < 1 || len(r.ExecutionPrincipals) > 100 || len(r.ExecutionGroups) > 100 || r.CreatedAt.IsZero() || r.UpdatedAt.Before(r.CreatedAt) {
		return NewError(ErrorCodeInvalidArgument, "invalid review case")
	}
	for _, list := range [][]string{r.ExecutionPrincipals, r.ExecutionGroups} {
		seen := map[string]bool{}
		for _, v := range list {
			if strings.TrimSpace(v) == "" || len(v) > 256 || seen[v] {
				return NewError(ErrorCodeInvalidArgument, "invalid/duplicate review participant")
			}
			seen[v] = true
		}
	}
	if r.CurrentContractID != nil && r.CurrentContractID.Validate() != nil {
		return NewError(ErrorCodeInvalidArgument, "invalid review authority pointer")
	}
	if (r.Status == ReviewInReview) != (r.CurrentContractID != nil) {
		return NewError(ErrorCodeInvalidTransition, "review case authority pointer differs from status")
	}
	if r.Status.Open() {
		if r.ClosedAt != nil || r.CloseReason != "" || r.ClosedBy != "" {
			return NewError(ErrorCodeInvalidTransition, "open review has closure")
		}
	} else if r.ClosedAt == nil || r.ClosedAt.Before(r.CreatedAt) || strings.TrimSpace(r.CloseReason) == "" || strings.TrimSpace(r.ClosedBy) == "" {
		return NewError(ErrorCodeInvalidArgument, "terminal review requires closure provenance")
	}
	return nil
}

// The complete execution/correction Principal history and frozen stable groups
// participate in independence, rather than only the latest bearer/key/Actor.
func (r ReviewCase) RequireIndependent(principal, group string) error {
	if strings.TrimSpace(principal) == "" {
		return NewError(ErrorCodeForbidden, "review Principal required")
	}
	for _, p := range r.ExecutionPrincipals {
		if p == principal {
			return NewError(ErrorCodeForbidden, "execution Principal cannot review its Task")
		}
	}
	if group != "" {
		for _, g := range r.ExecutionGroups {
			if g == group {
				return NewError(ErrorCodeForbidden, "reviewer shares an execution separation group")
			}
		}
	}
	return nil
}
func (r *ReviewCase) Bind(c ReviewContract, now time.Time) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Status != ReviewPending || c.CaseID != r.ID || c.Scope != r.Scope || c.WorkItemID != r.WorkItemID || c.SubmissionID != r.SubmissionID || c.SubmissionDigest != r.SubmissionDigest || c.IssuedSpecDigest != r.IssuedSpecDigest || c.CaseVersionAtAcquire != r.Version || !c.ValidAt(now) || uint64(c.FencingToken) <= uint64(r.LastFencingToken) {
		return NewError(ErrorCodeInvalidTransition, "review authority cannot bind immutable target")
	}
	if err := r.RequireIndependent(c.HolderPrincipalID, c.SeparationGroup); err != nil {
		return err
	}
	v, err := r.Version.Next()
	if err != nil {
		return err
	}
	id := c.ID
	r.CurrentContractID = &id
	r.LastFencingToken = uint64(c.FencingToken)
	r.Status = ReviewInReview
	r.Version = v
	r.UpdatedAt = now.UTC()
	return r.Validate()
}
func (r *ReviewCase) Release(c ReviewContract, now time.Time) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Scope != r.Scope || c.CaseID != r.ID || c.WorkItemID != r.WorkItemID || c.SubmissionID != r.SubmissionID || c.SubmissionDigest != r.SubmissionDigest || c.IssuedSpecDigest != r.IssuedSpecDigest {
		return NewError(ErrorCodeInvalidScope, "review target differs")
	}
	if r.Status != ReviewInReview || r.CurrentContractID == nil || *r.CurrentContractID != c.ID || c.Status == ContractActive || (c.Status != ContractExpired && c.Status != ContractRevoked && c.Status != ContractCompleted) {
		return NewError(ErrorCodeInvalidTransition, "review authority must close before pending")
	}
	v, err := r.Version.Next()
	if err != nil {
		return err
	}
	r.CurrentContractID = nil
	if uint64(c.FencingToken) > r.LastFencingToken {
		r.LastFencingToken = uint64(c.FencingToken)
	}
	r.Status = ReviewPending
	r.Version = v
	r.UpdatedAt = now.UTC()
	return r.Validate()
}
func (r *ReviewCase) Decide(c ReviewContract, status ReviewCaseStatus, principal, reason string, now time.Time) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Scope != r.Scope || c.CaseID != r.ID || c.SubmissionID != r.SubmissionID || c.SubmissionDigest != r.SubmissionDigest || c.IssuedSpecDigest != r.IssuedSpecDigest {
		return NewError(ErrorCodeInvalidScope, "review target differs")
	}
	if status != ReviewApproved && status != ReviewChangesRequested {
		return NewError(ErrorCodeInvalidArgument, "review decision must approve or request changes")
	}
	if r.Status != ReviewInReview || r.CurrentContractID == nil || *r.CurrentContractID != c.ID || c.Status != ContractCompleted || c.HolderPrincipalID != principal || strings.TrimSpace(reason) == "" {
		return NewError(ErrorCodeInvalidTransition, "review decision lacks closed bound authority")
	}
	v, err := r.Version.Next()
	if err != nil {
		return err
	}
	at := now.UTC()
	r.CurrentContractID = nil
	if uint64(c.FencingToken) > r.LastFencingToken {
		r.LastFencingToken = uint64(c.FencingToken)
	}
	r.Status = status
	r.Version = v
	r.UpdatedAt = at
	r.ClosedAt = &at
	r.CloseReason = reason
	r.ClosedBy = principal
	return r.Validate()
}
func (r *ReviewCase) Intervene(status ReviewCaseStatus, principal, reason string, expected Version, now time.Time) error {
	if status != ReviewCancelled && status != ReviewSuperseded {
		return NewError(ErrorCodeInvalidArgument, "explicit review intervention required")
	}
	if !r.Status.Open() || r.Version != expected || r.CurrentContractID != nil || strings.TrimSpace(principal) == "" || strings.TrimSpace(reason) == "" {
		return NewError(ErrorCodeInvalidTransition, "close review authority before intervention")
	}
	v, err := r.Version.Next()
	if err != nil {
		return err
	}
	at := now.UTC()
	r.Status = status
	r.Version = v
	r.UpdatedAt = at
	r.ClosedAt = &at
	r.CloseReason = reason
	r.ClosedBy = principal
	return r.Validate()
}
func ValidateReviewCaseUpdate(old, next ReviewCase) error {
	if !old.Status.Open() {
		return NewError(ErrorCodeInvalidTransition, "terminal review case is immutable")
	}
	v, err := old.Version.Next()
	if err != nil {
		return err
	}
	if next.Version != v {
		return NewError(ErrorCodeVersionConflict, "review case version must advance once")
	}
	o := old
	o.Version = next.Version
	o.Status = next.Status
	o.CurrentContractID = next.CurrentContractID
	o.LastFencingToken = next.LastFencingToken
	o.UpdatedAt = next.UpdatedAt
	o.ClosedAt = next.ClosedAt
	o.CloseReason = next.CloseReason
	o.ClosedBy = next.ClosedBy
	if !reflect.DeepEqual(old.LatestDecisionID, next.LatestDecisionID) && (old.Status != ReviewInReview || next.LatestDecisionID == nil) {
		return NewError(ErrorCodeInvalidTransition, "review decision pointer changes only on an accepted active review decision")
	}
	o.LatestDecisionID = next.LatestDecisionID
	if !reflect.DeepEqual(o, next) {
		return NewError(ErrorCodeInvalidTransition, "review submission/spec/policy/round/participants are immutable")
	}
	switch old.Status {
	case ReviewPending:
		if next.Status != ReviewInReview && next.Status != ReviewCancelled && next.Status != ReviewSuperseded {
			return NewError(ErrorCodeInvalidTransition, "invalid pending review transition")
		}
	case ReviewInReview:
		if next.Status != ReviewPending && next.Status != ReviewApproved && next.Status != ReviewChangesRequested {
			return NewError(ErrorCodeInvalidTransition, "invalid active review transition")
		}
	}
	return next.Validate()
}

// ReviewContract has its own holder, execution, lease and fencing state. It
// never contains executor credentials or requires its process to remain alive.
type ReviewContract struct {
	IssuedSpecificationID *ID                   `json:"issued_specification_id,omitempty"`
	LatestAuthorityID     *ID                   `json:"latest_authority_id,omitempty"`
	ID                    ID                    `json:"id"`
	Scope                 Scope                 `json:"scope"`
	CaseID                ID                    `json:"review_case_id"`
	WorkItemID            ID                    `json:"work_item_id"`
	SubmissionID          ID                    `json:"submission_id"`
	SubmissionDigest      string                `json:"submission_digest"`
	IssuedSpecDigest      string                `json:"issued_spec_digest"`
	HolderPrincipalID     string                `json:"holder_principal_id"`
	Actor                 ActorRef              `json:"actor_ref"`
	SeparationGroup       string                `json:"separation_group,omitempty"`
	Binding               SignedContractBinding `json:"binding"`
	Version               Version               `json:"version,string"`
	LeaseVersion          Version               `json:"lease_version,string"`
	CaseVersionAtAcquire  Version               `json:"case_version_at_acquire,string"`
	ExecutionID           ID                    `json:"execution_id"`
	FencingToken          FencingToken          `json:"fencing_token"`
	Status                ContractStatus        `json:"status"`
	AcquiredAt            time.Time             `json:"acquired_at"`
	ExpiresAt             time.Time             `json:"expires_at"`
	LastRenewedAt         time.Time             `json:"last_renewed_at"`
	LeasePolicy           LeasePolicy           `json:"lease_policy"`
	ClosedAt              *time.Time            `json:"closed_at,omitempty"`
	CloseReason           string                `json:"close_reason,omitempty"`
	ClosedBy              string                `json:"closed_by,omitempty"`
}

func (c ReviewContract) Validate() error {
	for _, id := range []*ID{c.IssuedSpecificationID, c.LatestAuthorityID} {
		if id != nil && id.Validate() != nil {
			return NewError(ErrorCodeInvalidArgument, "invalid review issuance pointer")
		}
	}
	for _, id := range []ID{c.ID, c.CaseID, c.WorkItemID, c.SubmissionID, c.ExecutionID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if c.Scope.Validate() != nil || c.Binding.Validate() != nil || c.Binding.AcceptanceFloor != AcceptanceIndependentReview || c.Version.Validate() != nil || c.LeaseVersion.Validate() != nil || c.CaseVersionAtAcquire.Validate() != nil || c.Actor.Validate() != nil || strings.TrimSpace(c.HolderPrincipalID) == "" || len(c.HolderPrincipalID) > 256 || len(c.SeparationGroup) > 256 || c.FencingToken == 0 || !ValidSignedDigest(c.SubmissionDigest) || !ValidSignedDigest(c.IssuedSpecDigest) || c.AcquiredAt.IsZero() || !c.ExpiresAt.After(c.AcquiredAt) || c.LastRenewedAt.Before(c.AcquiredAt) {
		return NewError(ErrorCodeInvalidArgument, "invalid review authority")
	}
	if _, err := c.LeasePolicy.TTL(0); err != nil {
		return err
	}
	if c.Status == ContractActive {
		if c.ClosedAt != nil || c.CloseReason != "" || c.ClosedBy != "" {
			return NewError(ErrorCodeInvalidTransition, "active review authority has closure")
		}
	} else if c.Status != ContractCompleted && c.Status != ContractExpired && c.Status != ContractRevoked {
		return NewError(ErrorCodeInvalidArgument, "invalid terminal review authority")
	} else if c.ClosedAt == nil || c.ClosedAt.Before(c.AcquiredAt) || strings.TrimSpace(c.CloseReason) == "" || strings.TrimSpace(c.ClosedBy) == "" {
		return NewError(ErrorCodeInvalidArgument, "closed review authority requires provenance")
	}
	return nil
}
func NewReviewContract(id, execution ID, r ReviewCase, principal string, actor ActorRef, group string, binding SignedContractBinding, policy LeasePolicy, ttl int, now time.Time) (ReviewContract, error) {
	var zero ReviewContract
	if err := r.Validate(); err != nil {
		return zero, err
	}
	if r.Status != ReviewPending {
		return zero, NewError(ErrorCodePreconditionFailed, "review case not pending")
	}
	if err := r.RequireIndependent(principal, group); err != nil {
		return zero, err
	}
	duration, err := policy.TTL(ttl)
	if err != nil {
		return zero, err
	}
	fence, err := NextFencing(uint64(r.LastFencingToken))
	if err != nil {
		return zero, err
	}
	c := ReviewContract{ID: id, Scope: r.Scope, CaseID: r.ID, WorkItemID: r.WorkItemID, SubmissionID: r.SubmissionID, SubmissionDigest: r.SubmissionDigest, IssuedSpecDigest: r.IssuedSpecDigest, HolderPrincipalID: principal, Actor: actor, SeparationGroup: group, Binding: binding, Version: 1, LeaseVersion: 1, CaseVersionAtAcquire: r.Version, ExecutionID: execution, FencingToken: fence, Status: ContractActive, AcquiredAt: now.UTC(), ExpiresAt: now.UTC().Add(duration), LastRenewedAt: now.UTC(), LeasePolicy: policy}
	return c, c.Validate()
}
func (c ReviewContract) ValidAt(now time.Time) bool {
	return c.Status == ContractActive && !now.Before(c.AcquiredAt) && now.Before(c.ExpiresAt)
}
func (c ReviewContract) Authorize(principal string, execution ID, fence FencingToken, now time.Time) error {
	if c.Status == ContractExpired || (c.Status == ContractActive && !now.Before(c.ExpiresAt)) {
		return NewError(ErrorCodeContractExpired, "review authority expired")
	}
	if !c.ValidAt(now) || c.HolderPrincipalID != principal || c.ExecutionID != execution || c.FencingToken != fence {
		return NewError(ErrorCodeForbidden, "review authority differs/inactive")
	}
	return nil
}
func (c *ReviewContract) Renew(principal string, execution ID, fence FencingToken, expected Version, ttl int, now time.Time) error {
	if err := c.Authorize(principal, execution, fence, now); err != nil {
		return err
	}
	if c.LeaseVersion != expected {
		return NewError(ErrorCodeVersionConflict, "review lease version changed")
	}
	duration, err := c.LeasePolicy.TTL(ttl)
	if err != nil {
		return err
	}
	v, err := c.LeaseVersion.Next()
	if err != nil {
		return err
	}
	c.LeaseVersion = v
	c.ExpiresAt = now.UTC().Add(duration)
	c.LastRenewedAt = now.UTC()
	return nil
}
func (c *ReviewContract) Resume(principal string, execution ID, fence FencingToken, expected Version, newExecution ID, now time.Time) error {
	if err := c.Authorize(principal, execution, fence, now); err != nil {
		return err
	}
	if c.LeaseVersion != expected || newExecution.Validate() != nil || newExecution == execution {
		return NewError(ErrorCodeVersionConflict, "review takeover CAS/execution differs")
	}
	f, err := NextFencing(uint64(c.FencingToken))
	if err != nil {
		return err
	}
	v, err := c.LeaseVersion.Next()
	if err != nil {
		return err
	}
	c.ExecutionID = newExecution
	c.FencingToken = f
	c.LeaseVersion = v
	return nil
}
func (c *ReviewContract) Close(status ContractStatus, principal, reason string, expected Version, now time.Time) error {
	if status == ContractCompleted && principal != c.HolderPrincipalID {
		return NewError(ErrorCodeForbidden, "only review holder may decide")
	}
	if c.Status != ContractActive || (status != ContractCompleted && status != ContractExpired && status != ContractRevoked) || strings.TrimSpace(principal) == "" || strings.TrimSpace(reason) == "" {
		return NewError(ErrorCodeInvalidTransition, "review authority cannot close")
	}
	if c.Version != expected {
		return NewError(ErrorCodeVersionConflict, "review content version changed")
	}
	if status == ContractExpired {
		if now.Before(c.ExpiresAt) {
			return NewError(ErrorCodeInvalidTransition, "review not expired")
		}
		now = c.ExpiresAt
	} else if !c.ValidAt(now) {
		return NewError(ErrorCodeContractExpired, "review expired before closure")
	}
	v, err := c.Version.Next()
	if err != nil {
		return err
	}
	at := now.UTC()
	c.Version = v
	c.Status = status
	c.ClosedAt = &at
	c.CloseReason = reason
	c.ClosedBy = principal
	return c.Validate()
}
func ValidateReviewContractUpdate(old, next ReviewContract) error {
	if old.Status != ContractActive {
		return NewError(ErrorCodeInvalidTransition, "terminal review authority immutable")
	}
	content := old.Version != next.Version
	lease := old.LeaseVersion != next.LeaseVersion
	if content == lease {
		return NewError(ErrorCodeVersionConflict, "review save advances content or lease exactly once")
	}
	o := old
	if content {
		v, err := old.Version.Next()
		if err != nil {
			return err
		}
		if v != next.Version || next.Status == ContractActive {
			return NewError(ErrorCodeVersionConflict, "review closure version/state differs")
		}
		o.Version = next.Version
		o.Status = next.Status
		o.ClosedAt = next.ClosedAt
		o.CloseReason = next.CloseReason
		o.ClosedBy = next.ClosedBy
	}
	if lease {
		v, err := old.LeaseVersion.Next()
		if err != nil {
			return err
		}
		if next.LeaseVersion != v {
			return NewError(ErrorCodeVersionConflict, "review lease version differs")
		}
		o.LatestAuthorityID = next.LatestAuthorityID
		o.LeaseVersion = next.LeaseVersion
		o.ExecutionID = next.ExecutionID
		o.FencingToken = next.FencingToken
		o.ExpiresAt = next.ExpiresAt
		o.LastRenewedAt = next.LastRenewedAt
		if next.ExecutionID != old.ExecutionID {
			if uint64(old.FencingToken) == ^uint64(0) || next.FencingToken != old.FencingToken+1 || !next.ExpiresAt.Equal(old.ExpiresAt) || !next.LastRenewedAt.Equal(old.LastRenewedAt) {
				return NewError(ErrorCodeInvalidTransition, "review takeover must fence without extending TTL")
			}
		} else if next.FencingToken != old.FencingToken || next.LastRenewedAt.Before(old.LastRenewedAt) {
			return NewError(ErrorCodeInvalidTransition, "review renewal changed fencing")
		}
	}
	if !reflect.DeepEqual(o, next) {
		return NewError(ErrorCodeInvalidTransition, "review acquisition/target/policy immutable")
	}
	return next.Validate()
}

// A v2 issuance binds its exact signed specification digest. Historical design
// fixtures without issuance metadata still use the unchanged v1 semantic digest.
func (c WorkContract) SubmissionSpecDigest() string {
	if c.SignedBinding != nil && c.SignedBinding.SpecificationDigest != "" {
		return c.SignedBinding.SpecificationDigest
	}
	return c.SpecDigest
}
