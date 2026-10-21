package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type TaskDeliveryFilter struct {
	WorkItemID domain.ID
	Pending    bool
	After      domain.ID
	Limit      int
}

type TaskDeliveryRepository interface {
	Insert(context.Context, domain.TaskDelivery) error
	Get(context.Context, domain.Scope, domain.ID) (domain.TaskDelivery, error)
	Pending(context.Context, domain.Scope, domain.ID) (*domain.TaskDelivery, error)
	List(context.Context, domain.Scope, TaskDeliveryFilter) ([]domain.TaskDelivery, error)
	InsertDecision(context.Context, domain.TaskDeliveryDecision) error
	GetDecision(context.Context, domain.Scope, domain.ID) (domain.TaskDeliveryDecision, error)
	DecisionForDelivery(context.Context, domain.Scope, domain.ID) (*domain.TaskDeliveryDecision, error)
	ListDecisions(context.Context, domain.Scope, domain.ID, domain.ID, int) ([]domain.TaskDeliveryDecision, error)
	InsertWithdrawal(context.Context, domain.TaskDeliveryWithdrawal) error
	GetWithdrawal(context.Context, domain.Scope, domain.ID) (domain.TaskDeliveryWithdrawal, error)
	WithdrawalForDelivery(context.Context, domain.Scope, domain.ID) (*domain.TaskDeliveryWithdrawal, error)
}
