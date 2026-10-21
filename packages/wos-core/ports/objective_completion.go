package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// ObjectiveCompletionAuthorityRepository stores explicit completion-policy
// authorizations as immutable, versioned history.
type ObjectiveCompletionAuthorityRepository interface {
	Append(context.Context, domain.ObjectiveCompletionAuthority) error
	ListByObjective(context.Context, domain.Scope, domain.ID) ([]domain.ObjectiveCompletionAuthority, error)
}

type ObjectiveCompletionAuthorityUnitOfWork interface {
	ObjectiveCompletionAuthorities() ObjectiveCompletionAuthorityRepository
}
