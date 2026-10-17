package domain

import "time"

type WorkProtocolPhase string

const (
	WorkProtocolLegacy         WorkProtocolPhase = "legacy"
	WorkProtocolDraining       WorkProtocolPhase = "draining"
	WorkProtocolContracts      WorkProtocolPhase = "contracts_v1"
	WorkProtocolSignedDraining WorkProtocolPhase = "draining_to_signed_v3"
	WorkProtocolSigned         WorkProtocolPhase = "signed_contracts_v3"
)

type TenantWorkProtocol struct {
	TenantID       ID                `json:"tenant_id"`
	Version        Version           `json:"protocol_version"`
	Phase          WorkProtocolPhase `json:"phase"`
	WriterEpoch    int               `json:"writer_epoch"`
	LeasePolicy    LeasePolicy       `json:"lease_policy"`
	UpdatedAt      time.Time         `json:"updated_at"`
	UpdatedBy      string            `json:"updated_by"`
	Reason         string            `json:"reason"`
	WritersDrained bool              `json:"writers_drained"`
}

func DefaultWorkProtocol(ns ID) TenantWorkProtocol {
	return TenantWorkProtocol{TenantID: ns, Version: 1, Phase: WorkProtocolLegacy, LeasePolicy: DefaultContractLeasePolicy()}
}
func (p TenantWorkProtocol) Validate() error {
	if err := p.TenantID.Validate(); err != nil {
		return err
	}
	if err := p.Version.Validate(); err != nil {
		return err
	}
	if p.Phase != WorkProtocolLegacy && p.Phase != WorkProtocolDraining && p.Phase != WorkProtocolContracts && p.Phase != WorkProtocolSignedDraining && p.Phase != WorkProtocolSigned {
		return NewError(ErrorCodeInvalidArgument, "unknown work protocol")
	}
	if p.WriterEpoch < 0 || p.WriterEpoch > 3 || p.WriterEpoch == 2 {
		return NewError(ErrorCodeContractProtocolRequired, "unsupported protocol writer epoch")
	}
	if p.Phase == WorkProtocolSigned && (p.WriterEpoch != 3 || !p.WritersDrained) {
		return NewError(ErrorCodePreconditionFailed, "signed cutover requires epoch 3 and explicit v1 writer drain")
	}
	if p.Phase != WorkProtocolSigned && p.WriterEpoch == 3 {
		return NewError(ErrorCodeSignedProtocolRequired, "epoch 3 cannot represent an unsigned protocol")
	}
	if p.Phase == WorkProtocolSignedDraining && p.WriterEpoch != 1 {
		return NewError(ErrorCodePreconditionFailed, "signed draining requires v1 writer epoch")
	}
	if p.Phase == WorkProtocolContracts && (p.WriterEpoch != 1 || !p.WritersDrained) {
		return NewError(ErrorCodePreconditionFailed, "cutover requires explicit old-writer drain")
	}
	_, err := p.LeasePolicy.TTL(0)
	return err
}
