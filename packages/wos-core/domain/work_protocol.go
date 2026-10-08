package domain

import "time"

type WorkProtocolPhase string

const (
	WorkProtocolLegacy    WorkProtocolPhase = "legacy"
	WorkProtocolDraining  WorkProtocolPhase = "draining"
	WorkProtocolContracts WorkProtocolPhase = "contracts_v1"
)

type NamespaceWorkProtocol struct {
	NamespaceID    ID                `json:"namespace_id"`
	Version        Version           `json:"protocol_version"`
	Phase          WorkProtocolPhase `json:"phase"`
	WriterEpoch    int               `json:"writer_epoch"`
	LeasePolicy    LeasePolicy       `json:"lease_policy"`
	UpdatedAt      time.Time         `json:"updated_at"`
	UpdatedBy      string            `json:"updated_by"`
	Reason         string            `json:"reason"`
	WritersDrained bool              `json:"writers_drained"`
}

func DefaultWorkProtocol(ns ID) NamespaceWorkProtocol {
	return NamespaceWorkProtocol{NamespaceID: ns, Version: 1, Phase: WorkProtocolLegacy, LeasePolicy: DefaultContractLeasePolicy()}
}
func (p NamespaceWorkProtocol) Validate() error {
	if err := p.NamespaceID.Validate(); err != nil {
		return err
	}
	if err := p.Version.Validate(); err != nil {
		return err
	}
	if p.Phase != WorkProtocolLegacy && p.Phase != WorkProtocolDraining && p.Phase != WorkProtocolContracts {
		return NewError(ErrorCodeInvalidArgument, "unknown work protocol")
	}
	if p.WriterEpoch < 0 || p.WriterEpoch > 1 {
		return NewError(ErrorCodeContractProtocolRequired, "unsupported protocol writer epoch")
	}
	if p.Phase == WorkProtocolContracts && (p.WriterEpoch != 1 || !p.WritersDrained) {
		return NewError(ErrorCodePreconditionFailed, "cutover requires explicit old-writer drain")
	}
	_, err := p.LeasePolicy.TTL(0)
	return err
}
