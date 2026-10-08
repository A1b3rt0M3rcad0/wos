package application

import (
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"time"
)

type AcquireWorkContractCommand struct {
	Scope                   d.Scope
	WorkItemID              d.ID
	ExpectedWorkItemVersion d.Version
	TTLSeconds              int `wos:"optional"`
}
type ContractAuthority struct {
	ExecutionID  d.ID
	FencingToken d.FencingToken
	SpecDigest   string
}
type RenewWorkContractCommand struct {
	Scope                d.Scope
	ContractID           d.ID
	Authority            ContractAuthority
	ExpectedLeaseVersion d.Version
	TTLSeconds           int `wos:"optional"`
}
type ResumeWorkContractCommand struct {
	Scope                d.Scope
	ContractID           d.ID
	Authority            ContractAuthority
	ExpectedLeaseVersion d.Version
}
type RevokeWorkContractCommand struct {
	Scope                   d.Scope
	ContractID              d.ID
	ExpectedContractVersion d.Version
	Reason                  string
}
type WorkContractResult struct {
	IssuedSpecification *signing.Document `json:"issued_specification,omitempty"`
	IssuedAuthority     *signing.Document `json:"issued_authority,omitempty"`
	LocalKeys           map[string]d.ID   `json:"local_keys,omitempty"`
	Artifacts           []d.Artifact      `json:"artifacts,omitempty"`
	Evidence            []d.Evidence      `json:"evidence,omitempty"`
	EvidenceLinks       []d.EvidenceLink  `json:"evidence_links,omitempty"`
	EvaluatedAt         time.Time         `json:"evaluated_at"`
	Contract            d.WorkContract    `json:"contract"`
	WorkItem            d.WorkItem        `json:"work_item"`
	PreviousContract    *d.WorkContract   `json:"previous_contract,omitempty"`
	Checkpoint          *d.WorkCheckpoint `json:"checkpoint,omitempty"`
	Submission          *d.WorkSubmission `json:"submission,omitempty"`
	Recovery            bool              `json:"recovery"`
}
