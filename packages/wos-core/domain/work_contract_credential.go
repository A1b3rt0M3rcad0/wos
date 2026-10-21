package domain

import "time"

// WorkContractCredentialBindingV1 binds an unsigned contracts_v1 authority
// to the canonical credential that acquired it. Signed-v2 keeps its own
// SignedContractBinding and does not use this version.
const WorkContractCredentialBindingV1 uint8 = 1

// NewWorkContractForCredential creates the same immutable contracts_v1
// snapshot as NewWorkContract and adds authenticated holder provenance.
func NewWorkContractForCredential(
	id, execution ID,
	w WorkItem,
	principal string,
	holderCredentialID ID,
	sessionCredentialID *ID,
	actor ActorRef,
	policy LeasePolicy,
	ttl int,
	spec WorkContractSpec,
	revision OutcomeRevision,
	now time.Time,
) (WorkContract, error) {
	c, err := NewWorkContract(id, execution, w, principal, actor, policy, ttl, spec, revision, now)
	if err != nil {
		return WorkContract{}, err
	}
	c.CredentialBindingVersion = WorkContractCredentialBindingV1
	c.HolderCredentialID = holderCredentialID
	c.SessionCredentialID = cloneID(sessionCredentialID)
	if err := c.Validate(); err != nil {
		return WorkContract{}, err
	}
	return c, nil
}

func (c WorkContract) validateCredentialBinding() error {
	switch c.CredentialBindingVersion {
	case 0:
		if !c.HolderCredentialID.IsZero() || c.SessionCredentialID != nil {
			return NewError(ErrorCodeLease, "unversioned contract cannot contain credential binding")
		}
	case WorkContractCredentialBindingV1:
		if c.SignedBinding != nil {
			return NewError(ErrorCodeLease, "signed-v2 contract cannot use contracts_v1 credential binding")
		}
		if err := c.HolderCredentialID.Validate(); err != nil {
			return WrapError(ErrorCodeLease, "contract holder credential id is invalid", err)
		}
		if c.SessionCredentialID != nil {
			if err := c.SessionCredentialID.Validate(); err != nil {
				return WrapError(ErrorCodeLease, "contract session credential id is invalid", err)
			}
		}
	default:
		return NewError(ErrorCodeLease, "contract credential binding version is unsupported")
	}
	return nil
}
