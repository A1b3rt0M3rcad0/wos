package ports

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

const MaxTaskVerificationBatch = 100

// TaskVerificationDeliveryFacts is a compact header of a validated immutable
// delivery. Adapters validate the complete payload and its typed references
// before projecting this header; omitted material is not evidence of absence.
type TaskVerificationDeliveryFacts struct {
	ID                   domain.ID
	Scope                domain.Scope
	WorkItemID           domain.ID
	Authority            domain.TaskDeliveryAuthority
	Round                uint64
	SupersedesDeliveryID *domain.ID
	DefinitionDigest     string
	Digest               string
	Summary              string
	DeliveredAt          time.Time
	SubmissionRef        *domain.TaskDeliverySubmissionRef
	ArtifactCount        int
	EvidenceCount        int
	CriterionCount       int
}

// TaskVerificationFacts contains only current navigation and the real records
// it references. Native deliveries and signed review cases remain different
// sources. Closed records are included for history, never inferred as pending.
type TaskVerificationFacts struct {
	WorkItemID                 domain.ID
	Version                    domain.Version
	Lifecycle                  domain.WorkItemLifecycle
	ContractsEnabled           bool
	CurrentLease               *domain.WorkLease
	CurrentContractID          *domain.ID
	PendingDeliveryID          *domain.ID
	LatestDeliveryID           *domain.ID
	LatestDeliveryDecisionID   *domain.ID
	LatestDeliveryWithdrawalID *domain.ID
	DeliveryRound              uint64
	PendingReviewCaseID        *domain.ID
	LatestReviewCaseID         *domain.ID
	CorrectionReviewCaseID     *domain.ID
	NativeDelivery             *TaskVerificationDeliveryFacts
	NativeDecision             *domain.TaskDeliveryDecision
	NativeWithdrawal           *domain.TaskDeliveryWithdrawal
	SignedReview               *domain.ReviewCase
	SignedCorrection           *domain.ReviewCase
	ExecutionContract          *domain.WorkContract
	ReviewContract             *domain.ReviewContract
}

type TaskVerificationReadRepository interface {
	// TaskVerificationFacts returns one scoped result per requested Task using a
	// bounded number of batch reads, not one repository lookup per card. Missing,
	// foreign, duplicate, or internally inconsistent sources fail closed. The
	// caller supplies at most MaxTaskVerificationBatch distinct Task IDs.
	TaskVerificationFacts(context.Context, domain.Scope, []domain.ID) ([]TaskVerificationFacts, error)
}

type TaskVerificationReadUnitOfWork interface {
	TaskVerificationReads() TaskVerificationReadRepository
}
