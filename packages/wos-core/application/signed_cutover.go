package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
	"time"
)

// Readiness is advisory. Activation repeats every check while holding the
// Namespace protocol guard and all affected Outcome coordination guards.
// WOS cannot attest that an operator has retired unrelated SQL processes.
type SignedProtocolReadiness struct {
	Protocol                d.NamespaceWorkProtocol `json:"protocol"`
	ValidLegacyClaims       int                     `json:"valid_legacy_claims"`
	ValidUnsignedContracts  int                     `json:"valid_unsigned_contracts"`
	IssuerReady             bool                    `json:"issuer_ready"`
	NamespacePolicyReady    bool                    `json:"namespace_policy_ready"`
	NamespacePolicyRevision signing.Decimal         `json:"namespace_policy_revision"`
	ReadyWithOperatorDrain  bool                    `json:"ready_with_operator_drain"`
	RequiresOperatorDrain   bool                    `json:"requires_operator_drain"`
	EvaluatedAt             time.Time               `json:"evaluated_at"`
}

func (s *Service) signedProtocolReadiness(ctx context.Context, u ports.UnitOfWork, ns d.ID, now time.Time) (SignedProtocolReadiness, error) {
	r := SignedProtocolReadiness{RequiresOperatorDrain: true, EvaluatedAt: now}
	protocol, e := protocolRepository(u)
	if e != nil {
		return r, e
	}
	r.Protocol, e = protocol.Lock(ctx, ns, false)
	if e != nil {
		return r, e
	}
	r.ValidLegacyClaims, e = protocol.ValidLegacyLeases(ctx, ns, now)
	if e != nil {
		return r, e
	}
	r.ValidUnsignedContracts, e = protocol.ValidUnsignedContracts(ctx, ns, now)
	if e != nil {
		return r, e
	}
	registry, e := signingRepository(u)
	if e != nil {
		return r, e
	}
	policy, e := registry.AcceptancePolicy(ctx, ns, nil, nil)
	if e == nil {
		r.NamespacePolicyReady = policy.Validate() == nil
		r.NamespacePolicyRevision = signing.Decimal(policy.Version)
	} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound {
		return r, e
	}
	if s.signed != nil && s.signed.issuer != nil {
		serverRepository, e := serverIdentityRepository(u)
		if e != nil {
			return r, e
		}
		server, e := serverRepository.Server(ctx)
		if e != nil {
			return r, e
		}
		if reflect.DeepEqual(server, s.signed.issuer.Identity()) {
			key, e := registry.Key(ctx, ns, server.IssuerKeyID)
			if e == nil {
				r.IssuerReady = key.Purpose == "issuer" && key.Status == "active" && key.PrincipalID == server.PrincipalID() && key.PublicKey == server.PublicKey && key.Fingerprint == server.Fingerprint
			} else if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
				r.IssuerReady = true
			} else {
				return r, e
			}
		}
	}
	r.ReadyWithOperatorDrain = r.Protocol.Phase == d.WorkProtocolSignedDraining && r.ValidLegacyClaims == 0 && r.ValidUnsignedContracts == 0 && r.IssuerReady && r.NamespacePolicyReady
	return r, nil
}
