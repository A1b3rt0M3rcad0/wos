package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// FencingToken is exact on every new wire representation, including JavaScript.
type FencingToken uint64

func (f FencingToken) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatUint(uint64(f), 10))
}
func (f *FencingToken) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return NewError(ErrorCodeInvalidArgument, "fencing_token must be a decimal string")
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || s != strconv.FormatUint(v, 10) || v == 0 {
		return NewError(ErrorCodeInvalidArgument, "fencing_token must be a positive canonical uint64 string")
	}
	*f = FencingToken(v)
	return nil
}
func NextFencing(f uint64) (FencingToken, error) {
	if f == math.MaxUint64 {
		return 0, NewError(ErrorCodeLease, "fencing token is exhausted")
	}
	return FencingToken(f + 1), nil
}
func SemanticDigest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

type ExecutionSpec struct {
	Instructions []string    `json:"instructions"`
	Constraints  []string    `json:"constraints"`
	Deliverables []string    `json:"deliverables"`
	ScopeHints   []string    `json:"scope_hints"`
	ContextRefs  []EntityRef `json:"context_refs"`
}

func (s ExecutionSpec) Validate() error {
	for _, list := range [][]string{s.Instructions, s.Constraints, s.Deliverables, s.ScopeHints} {
		if len(list) > 100 {
			return NewError(ErrorCodeInvalidArgument, "execution spec collection exceeds 100 items")
		}
		for _, v := range list {
			if len(v) > 16384 || strings.TrimSpace(v) == "" {
				return NewError(ErrorCodeInvalidArgument, "execution spec entry must be nonempty and bounded")
			}
		}
	}
	if len(s.ContextRefs) > 100 {
		return NewError(ErrorCodeInvalidArgument, "too many context references")
	}
	for _, ref := range s.ContextRefs {
		if err := ref.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type WorkContractSpec struct {
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	ObjectiveID   *ID                `json:"objective_id"`
	ExecutionSpec ExecutionSpec      `json:"execution_spec"`
	Criteria      []SuccessCriterion `json:"criteria"`
	Dependencies  []EntityRef        `json:"dependencies"`
}

func NormalizeContractSpec(s WorkContractSpec) WorkContractSpec {
	if s.Criteria == nil {
		s.Criteria = []SuccessCriterion{}
	}
	if s.Dependencies == nil {
		s.Dependencies = []EntityRef{}
	}
	if s.ExecutionSpec.Instructions == nil {
		s.ExecutionSpec.Instructions = []string{}
	}
	if s.ExecutionSpec.Constraints == nil {
		s.ExecutionSpec.Constraints = []string{}
	}
	if s.ExecutionSpec.Deliverables == nil {
		s.ExecutionSpec.Deliverables = []string{}
	}
	if s.ExecutionSpec.ScopeHints == nil {
		s.ExecutionSpec.ScopeHints = []string{}
	}
	if s.ExecutionSpec.ContextRefs == nil {
		s.ExecutionSpec.ContextRefs = []EntityRef{}
	}
	return s
}

type ContractStatus string

const (
	ContractActive    ContractStatus = "active"
	ContractExpired   ContractStatus = "expired"
	ContractRevoked   ContractStatus = "revoked"
	ContractCompleted ContractStatus = "completed"
)

type LeasePolicy struct {
	Revision          Version `json:"revision"`
	DefaultTTLSeconds int     `json:"default_ttl_seconds"`
	MinTTLSeconds     int     `json:"min_ttl_seconds"`
	MaxTTLSeconds     int     `json:"max_ttl_seconds"`
}

func DefaultContractLeasePolicy() LeasePolicy { return LeasePolicy{InitialVersion, 300, 30, 3600} }
func (p LeasePolicy) TTL(seconds int) (time.Duration, error) {
	if p.Revision == 0 || p.MinTTLSeconds < 1 || p.MaxTTLSeconds < p.MinTTLSeconds || p.MaxTTLSeconds > 86400 || p.DefaultTTLSeconds < p.MinTTLSeconds || p.DefaultTTLSeconds > p.MaxTTLSeconds {
		return 0, NewError(ErrorCodeInvalidConfig, "invalid lease policy")
	}
	if seconds == 0 {
		seconds = p.DefaultTTLSeconds
	}
	if seconds < p.MinTTLSeconds || seconds > p.MaxTTLSeconds {
		return 0, NewError(ErrorCodeLease, "lease ttl outside configured policy")
	}
	return time.Duration(seconds) * time.Second, nil
}

type WorkContract struct {
	ID                       ID               `json:"id"`
	Scope                    Scope            `json:"scope"`
	WorkItemID               ID               `json:"work_item_id"`
	HolderPrincipalID        string           `json:"holder_principal_id"`
	Actor                    ActorRef         `json:"actor_ref"`
	Status                   ContractStatus   `json:"status"`
	Version                  Version          `json:"version"`
	LeaseVersion             Version          `json:"lease_version"`
	ExecutionID              ID               `json:"execution_id"`
	FencingToken             FencingToken     `json:"fencing_token"`
	AcquiredAt               time.Time        `json:"acquired_at"`
	ExpiresAt                time.Time        `json:"expires_at"`
	LastRenewedAt            time.Time        `json:"last_renewed_at"`
	LeasePolicy              LeasePolicy      `json:"lease_policy"`
	WorkItemVersionAtAcquire Version          `json:"work_item_version_at_acquire"`
	OutcomeRevisionAtAcquire OutcomeRevision  `json:"outcome_revision_at_acquire"`
	Spec                     WorkContractSpec `json:"spec"`
	SpecDigest               string           `json:"spec_digest"`
	LatestCheckpointID       *ID              `json:"latest_checkpoint_id,omitempty"`
	LatestSubmissionID       *ID              `json:"latest_submission_id,omitempty"`
	ClosedAt                 *time.Time       `json:"closed_at,omitempty"`
	CloseReason              string           `json:"close_reason,omitempty"`
	ClosedBy                 string           `json:"closed_by,omitempty"`
}

func NewWorkContract(id, execution ID, w WorkItem, principal string, actor ActorRef, policy LeasePolicy, ttl int, spec WorkContractSpec, revision OutcomeRevision, now time.Time) (WorkContract, error) {
	duration, err := policy.TTL(ttl)
	if err != nil {
		return WorkContract{}, err
	}
	fencing, err := NextFencing(w.LastFencingToken)
	if err != nil {
		return WorkContract{}, err
	}
	spec = NormalizeContractSpec(spec)
	raw, err := json.Marshal(spec)
	if err != nil {
		return WorkContract{}, err
	}
	if len(raw) > 128*1024 {
		return WorkContract{}, NewError(ErrorCodeGraphLimitExceeded, "contract spec exceeds 128 KiB; reduce material and use references")
	}

	digest, err := SemanticDigest(spec)
	if err != nil {
		return WorkContract{}, err
	}
	c := WorkContract{ID: id, Scope: w.Scope, WorkItemID: w.ID, HolderPrincipalID: principal, Actor: actor, Status: ContractActive, Version: InitialVersion, LeaseVersion: InitialVersion, ExecutionID: execution, FencingToken: fencing, AcquiredAt: now.UTC(), ExpiresAt: now.UTC().Add(duration), LastRenewedAt: now.UTC(), LeasePolicy: policy, WorkItemVersionAtAcquire: w.Version, OutcomeRevisionAtAcquire: revision, Spec: spec, SpecDigest: digest}
	if err := c.Validate(); err != nil {
		return WorkContract{}, err
	}
	return c, nil
}
func (c WorkContract) Validate() error {
	for _, id := range []ID{c.ID, c.ExecutionID, c.WorkItemID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if err := c.Scope.Validate(); err != nil {
		return err
	}
	if err := c.Actor.Validate(); err != nil {
		return err
	}
	if err := c.Version.Validate(); err != nil {
		return err
	}
	if err := c.LeaseVersion.Validate(); err != nil {
		return err
	}
	if err := c.WorkItemVersionAtAcquire.Validate(); err != nil {
		return err
	}
	if _, err := c.LeasePolicy.TTL(0); err != nil {
		return err
	}
	if strings.TrimSpace(c.HolderPrincipalID) == "" || c.FencingToken == 0 || c.AcquiredAt.IsZero() || !c.ExpiresAt.After(c.AcquiredAt) || c.LastRenewedAt.Before(c.AcquiredAt) {
		return NewError(ErrorCodeInvalidArgument, "invalid contract authority")
	}
	if strings.TrimSpace(c.Spec.Title) == "" {
		return NewError(ErrorCodeInvalidArgument, "contract spec title is required")
	}
	if err := c.Spec.ExecutionSpec.Validate(); err != nil {
		return err
	}
	for _, criterion := range c.Spec.Criteria {
		if err := criterion.Validate(); err != nil {
			return err
		}
		if criterion.OwnerRef != (EntityRef{Scope: c.Scope, Kind: EntityKindWorkItem, ID: c.WorkItemID}) {
			return NewError(ErrorCodeInvalidScope, "contract criterion scope mismatch")
		}
	}
	for _, ref := range append(append([]EntityRef{}, c.Spec.Dependencies...), c.Spec.ExecutionSpec.ContextRefs...) {
		if err := ref.Validate(); err != nil {
			return err
		}
		if ref.Scope != c.Scope {
			return NewError(ErrorCodeInvalidScope, "contract reference scope mismatch")
		}
	}
	if c.Spec.ObjectiveID != nil {
		if err := c.Spec.ObjectiveID.Validate(); err != nil {
			return err
		}
	}
	digest, err := SemanticDigest(NormalizeContractSpec(c.Spec))
	if err != nil {
		return err
	}
	if digest != c.SpecDigest {
		return NewError(ErrorCodeContractSpecMismatch, "contract digest mismatch")
	}
	switch c.Status {
	case ContractActive:
		if c.ClosedAt != nil || c.CloseReason != "" || c.ClosedBy != "" {
			return NewError(ErrorCodeInvalidArgument, "active contract cannot have closure")
		}
	case ContractExpired, ContractRevoked, ContractCompleted:
		if c.ClosedAt == nil || c.ClosedAt.IsZero() || strings.TrimSpace(c.CloseReason) == "" || strings.TrimSpace(c.ClosedBy) == "" {
			return NewError(ErrorCodeInvalidArgument, "terminal contract requires closure provenance")
		}
	default:
		return NewError(ErrorCodeInvalidArgument, "unknown contract status")
	}
	return nil
}
func (c WorkContract) EffectiveStatus(now time.Time) ContractStatus {
	if c.Status == ContractActive && !now.Before(c.ExpiresAt) {
		return ContractExpired
	}
	return c.Status
}
func (c WorkContract) ValidAt(now time.Time) bool {
	return c.Status == ContractActive && !now.Before(c.AcquiredAt) && now.Before(c.ExpiresAt)
}
func (c WorkContract) Authorize(principal string, execution ID, token FencingToken, now time.Time) error {
	if c.EffectiveStatus(now) == ContractExpired {
		return NewError(ErrorCodeContractExpired, "contract has expired")
	}
	if c.Status == ContractRevoked {
		return NewError(ErrorCodeContractRevoked, "contract has been revoked")
	}
	if !c.ValidAt(now) {
		return NewError(ErrorCodeInvalidTransition, "contract is not active")
	}
	if principal != c.HolderPrincipalID {
		return NewError(ErrorCodeForbidden, "principal is not contract holder")
	}
	if execution != c.ExecutionID || token != c.FencingToken {
		return NewError(ErrorCodeStaleExecution, "execution generation is stale")
	}
	return nil
}
func (c *WorkContract) Renew(principal string, execution ID, token FencingToken, expected Version, ttl int, now time.Time) error {
	if err := c.Authorize(principal, execution, token, now); err != nil {
		return err
	}
	if expected != c.LeaseVersion {
		return NewError(ErrorCodeVersionConflict, "lease version conflict")
	}
	duration, err := c.LeasePolicy.TTL(ttl)
	if err != nil {
		return err
	}
	next, err := c.LeaseVersion.Next()
	if err != nil {
		return err
	}
	c.LeaseVersion = next
	c.ExpiresAt = now.UTC().Add(duration)
	c.LastRenewedAt = now.UTC()
	return nil
}
func (c *WorkContract) Resume(principal string, execution ID, token FencingToken, expected Version, newExecution ID, highWater uint64, now time.Time) error {
	if err := c.Authorize(principal, execution, token, now); err != nil {
		return err
	}
	if expected != c.LeaseVersion {
		return NewError(ErrorCodeVersionConflict, "lease version conflict")
	}
	if err := newExecution.Validate(); err != nil {
		return err
	}
	if newExecution == c.ExecutionID {
		return NewError(ErrorCodeInvalidArgument, "takeover requires a new execution id")
	}
	if highWater < uint64(c.FencingToken) {
		return NewError(ErrorCodeInvalidArgument, "fencing high water mark is inconsistent")
	}
	fencing, err := NextFencing(highWater)
	if err != nil {
		return err
	}
	next, err := c.LeaseVersion.Next()
	if err != nil {
		return err
	}
	c.ExecutionID = newExecution
	c.FencingToken = fencing
	c.LeaseVersion = next
	return nil
}
func (c *WorkContract) Expire(now time.Time) error {
	if c.Status != ContractActive || now.Before(c.ExpiresAt) {
		return NewError(ErrorCodeInvalidTransition, "contract is not pending expiration")
	}
	return c.close(ContractExpired, "lease expired", "system", c.ExpiresAt)
}
func (c *WorkContract) Revoke(principal, reason string, expected Version, now time.Time) error {
	if !c.ValidAt(now) {
		return NewError(ErrorCodeInvalidTransition, "only effectively active contracts can be revoked")
	}
	if expected != c.Version {
		return NewError(ErrorCodeVersionConflict, "contract version conflict")
	}
	return c.close(ContractRevoked, reason, principal, now)
}
func (c *WorkContract) close(status ContractStatus, reason, principal string, now time.Time) error {
	if c.Status != ContractActive {
		return NewError(ErrorCodeInvalidTransition, "terminal contract cannot close again")
	}
	if strings.TrimSpace(reason) == "" || strings.TrimSpace(principal) == "" {
		return NewError(ErrorCodeInvalidArgument, "closure reason and principal required")
	}
	next, err := c.Version.Next()
	if err != nil {
		return err
	}
	instant := now.UTC()
	c.Status = status
	c.Version = next
	c.ClosedAt = &instant
	c.CloseReason = reason
	c.ClosedBy = principal
	return nil
}
func (w *WorkItem) requireUncontracted() error {
	if w.CurrentContractID != nil {
		return NewError(ErrorCodeContractProtocolRequired, "revoke or reconcile contract before changing specification")
	}
	return nil
}
func (w *WorkItem) BindContract(c WorkContract, now time.Time) error {
	if !w.ContractsEnabled || c.Scope != w.Scope || c.WorkItemID != w.ID || !c.ValidAt(now) || uint64(c.FencingToken) <= w.LastFencingToken || w.CurrentContractID != nil || w.CurrentLease != nil {
		return NewError(ErrorCodeInvalidTransition, "work item cannot bind this contract")
	}
	if w.Lifecycle != WorkItemLifecycleTodo && w.Lifecycle != WorkItemLifecycleInProgress {
		return NewError(ErrorCodeInvalidTransition, "work item is not eligible for contract")
	}
	next, err := w.Version.Next()
	if err != nil {
		return err
	}
	id := c.ID
	w.CurrentContractID = &id
	w.LastFencingToken = uint64(c.FencingToken)
	w.Lifecycle = WorkItemLifecycleInProgress
	w.Version = next
	w.UpdatedAt = now.UTC()
	return nil
}
func (w *WorkItem) DetachContract(c WorkContract, now time.Time) error {
	if w.CurrentContractID == nil || *w.CurrentContractID != c.ID || c.Status == ContractActive {
		return NewError(ErrorCodeInvalidTransition, "contract must be terminal before detach")
	}
	next, err := w.Version.Next()
	if err != nil {
		return err
	}
	w.CurrentContractID = nil
	w.Version = next
	w.UpdatedAt = now.UTC()
	return nil
}

func (c WorkContract) Ref() EntityRef {
	return EntityRef{Scope: c.Scope, Kind: EntityKindWorkItem, ID: c.WorkItemID}
}

// ValidateContractUpdate is shared by all adapters. Snapshot/acquisition are immutable.
func ValidateContractUpdate(old, next WorkContract) error {
	if old.Status != ContractActive {
		return NewError(ErrorCodeInvalidTransition, "terminal contract is immutable")
	}
	if old.ID != next.ID || old.Scope != next.Scope || old.WorkItemID != next.WorkItemID || old.HolderPrincipalID != next.HolderPrincipalID || old.Actor != next.Actor || old.SpecDigest != next.SpecDigest || !old.AcquiredAt.Equal(next.AcquiredAt) || old.WorkItemVersionAtAcquire != next.WorkItemVersionAtAcquire || old.OutcomeRevisionAtAcquire != next.OutcomeRevisionAtAcquire {
		return NewError(ErrorCodeContractSpecMismatch, "contract acquisition is immutable")
	}
	contentChanged := next.Version != old.Version
	leaseChanged := next.LeaseVersion != old.LeaseVersion
	if contentChanged == leaseChanged {
		return NewError(ErrorCodeVersionConflict, "save must advance either content or lease once")
	}
	if contentChanged {
		v, err := old.Version.Next()
		if err != nil {
			return err
		}
		if next.Version != v || next.ExecutionID != old.ExecutionID || next.FencingToken != old.FencingToken || !next.ExpiresAt.Equal(old.ExpiresAt) || !next.LastRenewedAt.Equal(old.LastRenewedAt) || next.LeasePolicy != old.LeasePolicy {
			return NewError(ErrorCodeVersionConflict, "content save changed lease")
		}
	}
	if leaseChanged {
		v, err := old.LeaseVersion.Next()
		if err != nil {
			return err
		}
		if next.LeaseVersion != v || next.Status != old.Status || !reflect.DeepEqual(next.LatestCheckpointID, old.LatestCheckpointID) || !reflect.DeepEqual(next.LatestSubmissionID, old.LatestSubmissionID) || next.ClosedAt != nil {
			return NewError(ErrorCodeVersionConflict, "lease save changed content")
		}
	}
	return next.Validate()
}
