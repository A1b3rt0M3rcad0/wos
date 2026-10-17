package application

import (
	"context"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func lockAcquisitionQuota(ctx context.Context, u ports.UnitOfWork, identity Identity, command string) error {
	kind := ""
	switch command {
	case "AcquireSignedWorkContract", "AcquireNextSignedWorkContract":
		kind = "work"
	case "AcquireSignedReviewContract", "AcquireNextSignedReviewContract":
		kind = "review"
	default:
		return nil
	}
	guard, ok := u.(ports.SignedQuotaUnitOfWork)
	if !ok {
		return nil // legacy adapters retain the exclusive Namespace barrier
	}
	registry, err := signingRepository(u)
	if err != nil {
		return err
	}
	policy, err := namespaceAcceptancePolicy(ctx, registry, identity.NamespaceID)
	if err != nil {
		return err
	}
	shared := policy.MaxActiveWorkContracts > 0
	if kind == "review" {
		shared = policy.MaxActiveReviewContracts > 0
	}
	return guard.LockSignedQuota(ctx, identity.NamespaceID, identity.PrincipalID, kind, shared)
}

func countAcquisitionQuota(ctx context.Context, repo ports.SignedContractRepository, ns d.ID, principal string, credential d.ID, now time.Time, includeNamespace bool) (ports.SignedActiveCounts, error) {
	if indexed, ok := repo.(ports.SignedQuotaCountsRepository); ok {
		return indexed.CountQuota(ctx, ns, principal, credential, now, includeNamespace)
	}
	return repo.Counts(ctx, ns, principal, credential, now)
}
