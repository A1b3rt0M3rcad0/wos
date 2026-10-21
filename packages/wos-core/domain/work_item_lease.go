package domain

import (
	"math"
	"strings"
	"time"
)

// RenewLease extends the current lease without changing its claim identity or
// fencing token. Once a lease expires it cannot be renewed; the WorkItem must
// be reclaimed so stale claimants are fenced out by a new token.
func (w *WorkItem) RenewLease(principalID string, claimID ID, fencingToken uint64, ttl time.Duration, now time.Time) error {
	if w.ContractsEnabled {
		return NewError(ErrorCodeContractProtocolRequired, "use contract protocol")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.CurrentLease == nil {
		return NewError(ErrorCodeInvalidTransition, "work item is not currently claimed")
	}
	if err := w.validateLeaseOwnership(principalID, claimID, fencingToken); err != nil {
		return err
	}
	now = now.UTC()
	if !w.CurrentLease.ValidAt(now) {
		return NewError(ErrorCodeLease, "lease is not active at renewal time")
	}
	ttl, err := normalizeLeaseTTL(ttl)
	if err != nil {
		return err
	}
	w.CurrentLease.ExpiresAt = now.Add(ttl)
	return w.touch(now)
}

// RenewLeaseForCredential extends a credential-bound lease without changing
// its claim identity or fencing token.
func (w *WorkItem) RenewLeaseForCredential(principalID string, credentialID ID, claimID ID, fencingToken uint64, ttl time.Duration, now time.Time) error {
	if w.ContractsEnabled {
		return NewError(ErrorCodeContractProtocolRequired, "use contract protocol")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.CurrentLease == nil {
		return NewError(ErrorCodeInvalidTransition, "work item is not currently claimed")
	}
	if err := w.validateLeaseCredentialOwnership(principalID, credentialID, claimID, fencingToken); err != nil {
		return err
	}
	now = now.UTC()
	if !w.CurrentLease.ValidAt(now) {
		return NewError(ErrorCodeLease, "lease is not active at renewal time")
	}
	ttl, err := normalizeLeaseTTL(ttl)
	if err != nil {
		return err
	}
	w.CurrentLease.ExpiresAt = now.Add(ttl)
	return w.touch(now)
}

// Reclaim replaces an expired lease with a new claim identity and a strictly
// newer fencing token. The WorkItem remains in_progress: expiration makes it
// recoverable, but never rewrites lifecycle state by the passage of time alone.
func (w *WorkItem) Reclaim(claimID ID, principalID string, actor ActorRef, ttl time.Duration, now time.Time) error {
	if w.ContractsEnabled {
		return NewError(ErrorCodeContractProtocolRequired, "use contract protocol")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.CurrentLease == nil {
		return NewError(ErrorCodeInvalidTransition, "work item is not currently claimed")
	}
	if err := w.CurrentLease.Validate(); err != nil {
		return err
	}
	if w.CurrentLease.CredentialBindingVersion != 0 {
		return NewError(ErrorCodeLease, "credential-bound lease requires credential identity to reclaim")
	}
	now = now.UTC()
	if now.Before(w.CurrentLease.ExpiresAt) {
		return NewError(ErrorCodeLease, "current lease has not expired")
	}
	if err := claimID.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "claim id is invalid", err)
	}
	if claimID == w.CurrentLease.ClaimID {
		return NewError(ErrorCodeLease, "reclaim requires a new claim id")
	}
	if strings.TrimSpace(principalID) == "" {
		return NewError(ErrorCodeLease, "lease principal_id is required")
	}
	if err := actor.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "lease actor is invalid", err)
	}
	ttl, err := normalizeLeaseTTL(ttl)
	if err != nil {
		return err
	}
	if w.LastFencingToken == math.MaxUint64 {
		return NewError(ErrorCodeLease, "fencing token is exhausted")
	}
	w.LastFencingToken++
	w.CurrentLease = &WorkLease{
		ClaimID:      claimID,
		PrincipalID:  principalID,
		Actor:        actor,
		FencingToken: w.LastFencingToken,
		AcquiredAt:   now,
		ExpiresAt:    now.Add(ttl),
	}
	return w.touch(now)
}

// ReclaimForCredential takes over an expired legacy-protocol lease and binds
// its fresh claim and fencing token to the authenticated holder identity.
func (w *WorkItem) ReclaimForCredential(claimID ID, principalID string, holderCredentialID ID, sessionCredentialID *ID, actor ActorRef, ttl time.Duration, now time.Time) error {
	if w.ContractsEnabled {
		return NewError(ErrorCodeContractProtocolRequired, "use contract protocol")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.CurrentLease == nil {
		return NewError(ErrorCodeInvalidTransition, "work item is not currently claimed")
	}
	if err := w.CurrentLease.Validate(); err != nil {
		return err
	}
	binding := WorkLease{CredentialBindingVersion: WorkLeaseCredentialBindingV1, HolderCredentialID: holderCredentialID, SessionCredentialID: cloneID(sessionCredentialID)}
	if err := binding.validateCredentialBinding(); err != nil {
		return err
	}
	now = now.UTC()
	if now.Before(w.CurrentLease.ExpiresAt) {
		return NewError(ErrorCodeLease, "current lease has not expired")
	}
	if err := claimID.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "claim id is invalid", err)
	}
	if claimID == w.CurrentLease.ClaimID {
		return NewError(ErrorCodeLease, "reclaim requires a new claim id")
	}
	if strings.TrimSpace(principalID) == "" {
		return NewError(ErrorCodeLease, "lease principal_id is required")
	}
	if err := actor.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "lease actor is invalid", err)
	}
	ttl, err := normalizeLeaseTTL(ttl)
	if err != nil {
		return err
	}
	if w.LastFencingToken == math.MaxUint64 {
		return NewError(ErrorCodeLease, "fencing token is exhausted")
	}
	w.LastFencingToken++
	w.CurrentLease = &WorkLease{
		ClaimID:                  claimID,
		PrincipalID:              principalID,
		Actor:                    actor,
		FencingToken:             w.LastFencingToken,
		AcquiredAt:               now,
		ExpiresAt:                now.Add(ttl),
		CredentialBindingVersion: binding.CredentialBindingVersion,
		HolderCredentialID:       binding.HolderCredentialID,
		SessionCredentialID:      cloneID(binding.SessionCredentialID),
	}
	return w.touch(now)
}

func normalizeLeaseTTL(ttl time.Duration) (time.Duration, error) {
	if ttl == 0 {
		ttl = DefaultLeaseTTL
	}
	if ttl < MinLeaseTTL || ttl > MaxLeaseTTL {
		return 0, NewError(ErrorCodeLease, "lease ttl is outside supported range")
	}
	return ttl, nil
}
