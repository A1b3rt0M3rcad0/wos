package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type ReviewFilter struct {
	WorkItemID        domain.ID
	HolderPrincipalID string
	Status            string
	After             domain.ID
	Limit             int
}
type SignedActiveCounts struct {
	PrincipalWork    int `json:"principal_work"`
	PrincipalReview  int `json:"principal_review"`
	CredentialWork   int `json:"credential_work"`
	CredentialReview int `json:"credential_review"`
	NamespaceWork    int `json:"namespace_work"`
	NamespaceReview  int `json:"namespace_review"`
}
type SignedContractRepository interface {
	ExecutionParticipants(context.Context, domain.Scope, domain.ID, int) ([]domain.ExecutionParticipant, error)
	Case(context.Context, domain.Scope, domain.ID) (domain.ReviewCase, error)
	Cases(context.Context, domain.Scope, ReviewFilter) ([]domain.ReviewCase, error)
	OpenCase(context.Context, domain.Scope, domain.ID) (*domain.ReviewCase, error)
	InsertCase(context.Context, domain.ReviewCase) error
	SaveCase(context.Context, domain.ReviewCase, domain.Version) error
	ReviewContract(context.Context, domain.Scope, domain.ID) (domain.ReviewContract, error)
	ReviewContracts(context.Context, domain.Scope, ReviewFilter) ([]domain.ReviewContract, error)
	InsertReviewContract(context.Context, domain.ReviewContract) error
	SaveReviewContract(context.Context, domain.ReviewContract, domain.Version, domain.Version) error
	Fact(context.Context, domain.Scope, domain.ID) (domain.SignedFact, error)
	Facts(context.Context, domain.Scope, domain.ID, string, domain.ID, int) ([]domain.SignedFact, error)
	InsertFact(context.Context, domain.SignedFact) error
	Acceptance(context.Context, domain.ID, string, string) (domain.SignedFact, error)
	Counts(context.Context, domain.ID, string, domain.ID, time.Time) (SignedActiveCounts, error)
}
type SignedContractUnitOfWork interface {
	SignedWorkContracts() WorkContractRepository
	SignedContracts() SignedContractRepository
}
