package postgres

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// insertWorkItemFixture keeps storage fixtures aligned with the current schema:
// persisted Tasks must name an Objective owner. Existing tests that do not
// exercise hierarchy reuse their sole Objective or get one stable fixture
// Objective per Outcome. Tests with multiple possible owners must set the
// intended ObjectiveID explicitly.
func insertWorkItemFixture(ctx context.Context, objectives ports.ObjectiveRepository, workItems ports.WorkItemRepository, item *domain.WorkItem) error {
	if item == nil {
		return fmt.Errorf("nil work item fixture")
	}
	if item.ObjectiveID == nil {
		owners, err := objectives.ListByOutcome(ctx, item.Scope)
		if err != nil {
			return err
		}
		switch len(owners) {
		case 0:
			digest := fmt.Sprintf("%x", sha256.Sum256([]byte(item.Scope.OutcomeID.String())))
			id := domain.MustParseID("0199fffe-0000-7000-8000-" + digest[:12])
			owner, err := domain.NewObjective(id, item.Scope, "Storage fixture owner", "", domain.PriorityNormal, false, item.CreatedAt)
			if err != nil {
				return err
			}
			if err := objectives.Insert(ctx, owner); err != nil {
				return err
			}
			item.ObjectiveID = &owner.ID
		case 1:
			item.ObjectiveID = &owners[0].ID
		default:
			return fmt.Errorf("work item fixture %s needs an explicit Objective owner; found %d candidates", item.ID, len(owners))
		}
	}
	return workItems.Insert(ctx, *item)
}
