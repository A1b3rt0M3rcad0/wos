package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type ContractFilter struct {
	WorkItemID        domain.ID
	HolderPrincipalID string
	Status            domain.ContractStatus
	After             domain.ID
	Limit             int
}
type WorkContractRepository interface {
	Get(context.Context, domain.Scope, domain.ID) (domain.WorkContract, error)
	Current(context.Context, domain.Scope, domain.ID) (*domain.WorkContract, error)
	List(context.Context, domain.Scope, ContractFilter) ([]domain.WorkContract, error)
	Insert(context.Context, domain.WorkContract) error
	Save(context.Context, domain.WorkContract, domain.Version, domain.Version) error
	InsertCheckpoint(context.Context, domain.WorkCheckpoint) error
	GetCheckpoint(context.Context, domain.Scope, domain.ID) (domain.WorkCheckpoint, error)
	ListCheckpoints(context.Context, domain.Scope, domain.ID, domain.ID, int) ([]domain.WorkCheckpoint, error)
	InsertSubmission(context.Context, domain.WorkSubmission) error
	GetSubmission(context.Context, domain.Scope, domain.ID) (domain.WorkSubmission, error)
	ListSubmissions(context.Context, domain.Scope, domain.ID, domain.ID, int) ([]domain.WorkSubmission, error)
}
type WorkContractUnitOfWork interface {
	WorkContracts() WorkContractRepository
}

type ContractCandidate struct {
	ID        domain.ID       `json:"id"`
	Version   domain.Version  `json:"version"`
	Priority  domain.Priority `json:"priority"`
	CreatedAt time.Time       `json:"created_at"`
}
type ContractCandidateQuery struct {
	AfterPriority int
	AfterCreated  time.Time
	AfterID       domain.ID
	Limit         int
}
type ContractCandidateRepository interface {
	ContractCandidates(context.Context, domain.Scope, ContractCandidateQuery) ([]ContractCandidate, error)
}

func ContractPriorityRank(p domain.Priority) int {
	switch p {
	case domain.PriorityCritical:
		return 0
	case domain.PriorityHigh:
		return 1
	case domain.PriorityNormal:
		return 2
	default:
		return 3
	}
}
