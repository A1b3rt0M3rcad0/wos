package application

import (
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type AcquireWorkContractCommand struct {
	Scope                   d.Scope
	WorkItemID              d.ID
	ExpectedWorkItemVersion d.Version
	TTLSeconds              int
}
type ContractAuthority struct {
	ExecutionID  d.ID
	FencingToken d.FencingToken
	SpecDigest   string
}
type RenewWorkContractCommand struct {
	Scope      d.Scope
	ContractID d.ID
	ContractAuthority
	ExpectedLeaseVersion d.Version
	TTLSeconds           int
}
type ResumeWorkContractCommand struct {
	Scope      d.Scope
	ContractID d.ID
	ContractAuthority
	ExpectedLeaseVersion d.Version
}
type RevokeWorkContractCommand struct {
	Scope                   d.Scope
	ContractID              d.ID
	ExpectedContractVersion d.Version
	Reason                  string
}
type WorkContractResult struct {
	EvaluatedAt      time.Time       `json:"evaluated_at"`
	Contract         d.WorkContract  `json:"contract"`
	WorkItem         d.WorkItem      `json:"work_item"`
	PreviousContract *d.WorkContract `json:"previous_contract,omitempty"`
	Recovery         bool            `json:"recovery"`
}
