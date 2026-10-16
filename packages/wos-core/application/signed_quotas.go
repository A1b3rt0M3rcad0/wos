package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func namespaceAcceptancePolicy(ctx context.Context, registry ports.SigningIdentityRepository, ns d.ID) (d.WorkAcceptancePolicy, error) {
	p, e := registry.AcceptancePolicy(ctx, ns, nil, nil)
	if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
		return d.WorkAcceptancePolicy{}, nil
	}
	return p, e
}

// SigningQuotaView exposes limits and only the caller's activity. It never lists
// another Principal's contracts or credential identity.
type SigningQuotaView struct {
	MaxActiveWorkPerPrincipal    int  `json:"max_active_work_per_principal"`
	MaxActiveReviewPerPrincipal  int  `json:"max_active_review_per_principal"`
	MaxActiveWorkPerCredential   int  `json:"max_active_work_per_credential,omitempty"`
	MaxActiveReviewPerCredential int  `json:"max_active_review_per_credential,omitempty"`
	MaxActiveWorkPerNamespace    int  `json:"max_active_work_per_namespace,omitempty"`
	MaxActiveReviewPerNamespace  int  `json:"max_active_review_per_namespace,omitempty"`
	EffectiveWorkCeiling         int  `json:"effective_work_ceiling"`
	EffectiveReviewCeiling       int  `json:"effective_review_ceiling"`
	CredentialPolicyConfigured   bool `json:"credential_policy_configured"`
	ActivePrincipalWork          int  `json:"active_principal_work"`
	ActivePrincipalReview        int  `json:"active_principal_review"`
	ActiveCredentialWork         int  `json:"active_credential_work"`
	ActiveCredentialReview       int  `json:"active_credential_review"`
}

func signingQuotaView(policy *d.CredentialPolicy, ns d.WorkAcceptancePolicy, counts ports.SignedActiveCounts) SigningQuotaView {
	q := SigningQuotaView{MaxActiveWorkPerPrincipal: ns.PrincipalWorkLimit(), MaxActiveReviewPerPrincipal: ns.PrincipalReviewLimit(), MaxActiveWorkPerNamespace: ns.MaxActiveWorkContracts, MaxActiveReviewPerNamespace: ns.MaxActiveReviewContracts, ActivePrincipalWork: counts.PrincipalWork, ActivePrincipalReview: counts.PrincipalReview, ActiveCredentialWork: counts.CredentialWork, ActiveCredentialReview: counts.CredentialReview}
	if policy != nil {
		q.CredentialPolicyConfigured = true
		q.MaxActiveWorkPerCredential = policy.MaxActiveWorkContracts
		q.MaxActiveReviewPerCredential = policy.MaxActiveReviewContracts
		q.EffectiveWorkCeiling = min(q.MaxActiveWorkPerPrincipal, q.MaxActiveWorkPerCredential)
		q.EffectiveReviewCeiling = min(q.MaxActiveReviewPerPrincipal, q.MaxActiveReviewPerCredential)
		if ns.MaxActiveWorkContracts > 0 {
			q.EffectiveWorkCeiling = min(q.EffectiveWorkCeiling, ns.MaxActiveWorkContracts)
		}
		if ns.MaxActiveReviewContracts > 0 {
			q.EffectiveReviewCeiling = min(q.EffectiveReviewCeiling, ns.MaxActiveReviewContracts)
		}
	}
	return q
}
