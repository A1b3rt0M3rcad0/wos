package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// requireTaskDeliveryReviewForCompletion keeps operational_v2 completion on
// the exact-delivery, independent-review path. Legacy profiles retain their
// existing completion commands; profile policy is centralized in
// workItemRequiresIndependentReview so all protocol entry points use the same
// acceptance floor.
func requireTaskDeliveryReviewForCompletion(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, workID domain.ID) error {
	required, err := workItemRequiresIndependentReview(ctx, uow, scope, workID)
	if err != nil || !required {
		return err
	}
	ref := domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: workID}
	return domain.NewDetailedError(
		domain.ErrorCodePreconditionFailed,
		"operational_v2 Tasks must be submitted as an exact TaskDelivery and independently accepted; direct completion is disabled",
		domain.ErrorDetails{
			ReasonCode: "independent_task_delivery_review_required",
			SubjectRef: &ref,
			Remedies:   []domain.ErrorRemedy{{ActionID: "SubmitTaskForReview", SubjectRef: &ref}},
		},
	)
}
