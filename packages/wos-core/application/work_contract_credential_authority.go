package application

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// resolveWorkContractCredentialAuthority applies the credential checks used
// for legacy leases to contracts_v1. Historical contracts remain usable by a
// trusted embedded service, while an authenticated caller cannot inherit an
// unbound Principal-only contract.
func (s *Service) resolveWorkContractCredentialAuthority(
	ctx context.Context,
	uow ports.UnitOfWork,
	commandContext domain.CommandContext,
	scope domain.Scope,
	contract *domain.WorkContract,
	commandName string,
	now time.Time,
) (*leaseCredentialAuthority, error) {
	var binding *domain.WorkLease
	if contract != nil && contract.CredentialBindingVersion != 0 {
		binding = &domain.WorkLease{
			CredentialBindingVersion: contract.CredentialBindingVersion,
			HolderCredentialID:       contract.HolderCredentialID,
			SessionCredentialID:      cloneApplicationID(contract.SessionCredentialID),
		}
	}
	authority, err := s.resolveLeaseCredentialAuthority(ctx, uow, commandContext, scope, binding, commandName, now)
	if err != nil {
		return nil, err
	}
	if contract == nil {
		return authority, nil
	}
	if contract.CredentialBindingVersion == 0 {
		if authority != nil || s.requireIdentity {
			return nil, leaseCredentialBindingUnavailable()
		}
		return nil, nil
	}
	if authority == nil {
		return nil, leaseCredentialBindingUnavailable()
	}
	if authority.holderID != contract.HolderCredentialID {
		return nil, leaseOwnerMismatch()
	}
	return authority, nil
}

func cloneApplicationID(value *domain.ID) *domain.ID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
