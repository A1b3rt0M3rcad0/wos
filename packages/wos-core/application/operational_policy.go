package application

import (
	"context"
	"strings"
	"unicode/utf8"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const maxOperationalProfileMigrationReasonRunes = 1024

// ConfigureOutcomeOperationalProfileCommand is intentionally a narrow,
// one-way migration command. It cannot select a weaker policy or configure
// Objective completion delegation.
type ConfigureOutcomeOperationalProfileCommand struct {
	Scope                   d.Scope            `json:"scope"`
	ExpectedVersion         d.Version          `json:"expected_version,string"`
	ExpectedOutcomeRevision *d.OutcomeRevision `json:"expected_outcome_revision,string"`
	Reason                  string             `json:"reason"`
}

// ConfigureOutcomeOperationalProfile upgrades exactly one existing Outcome
// from compatibility_v1 to operational_v2 with strict_work. The preview is
// re-evaluated after the single Outcome guard is acquired, so the migration
// and its blockers use the same transaction snapshot.
func (s *Service) ConfigureOutcomeOperationalProfile(ctx context.Context, commandContext d.CommandContext, cmd ConfigureOutcomeOperationalProfileCommand) (MutationResult[d.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[d.Outcome]{}, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return MutationResult[d.Outcome]{}, err
	}
	if err := cmd.ExpectedVersion.Validate(); err != nil {
		return MutationResult[d.Outcome]{}, err
	}
	if cmd.ExpectedOutcomeRevision == nil {
		return MutationResult[d.Outcome]{}, d.NewError(d.ErrorCodeInvalidArgument, "expected_outcome_revision is required")
	}
	cmd.Reason = strings.TrimSpace(cmd.Reason)
	if cmd.Reason == "" || !utf8.ValidString(cmd.Reason) || strings.ContainsRune(cmd.Reason, 0) || utf8.RuneCountInString(cmd.Reason) > maxOperationalProfileMigrationReasonRunes {
		return MutationResult[d.Outcome]{}, d.NewError(d.ErrorCodeInvalidArgument, "migration reason is required and must not exceed 1024 characters")
	}
	if commandContext.IdempotencyKey == "" {
		return MutationResult[d.Outcome]{}, d.NewError(d.ErrorCodeInvalidArgument, "operational profile migration requires an idempotency key")
	}
	if err := authorizeOperationalPolicyAdmin(ctx, s.authorizer, cmd.Scope, commandContext.PrincipalID); err != nil {
		return MutationResult[d.Outcome]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (d.Outcome, d.OutcomeRevision, error) {
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return d.Outcome{}, 0, err
		}
		if !s.requireIdentity {
			if err := authorizeOperationalPolicyAdminInUnitOfWork(ctx, s.authorizer, uow, cmd.Scope, commandContext.PrincipalID); err != nil {
				return d.Outcome{}, 0, err
			}
		}
		if coordination.Revision != *cmd.ExpectedOutcomeRevision {
			return d.Outcome{}, 0, d.NewError(d.ErrorCodeVersionConflict, "outcome expected_revision is stale")
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return d.Outcome{}, 0, err
		}
		if outcome.Version != cmd.ExpectedVersion {
			return d.Outcome{}, 0, d.NewError(d.ErrorCodeVersionConflict, "outcome expected_version is stale")
		}
		transactionTime, err := s.transactionTime(ctx, uow)
		if err != nil {
			return d.Outcome{}, 0, err
		}
		preview, err := buildOperationalProfileMigrationPreview(ctx, uow, outcome, coordination.Revision, transactionTime)
		if err != nil {
			return d.Outcome{}, 0, err
		}
		if !preview.CanMigrate {
			return d.Outcome{}, 0, d.NewError(d.ErrorCodePreconditionFailed, "operational profile migration is blocked; resolve preview findings and retry")
		}
		if outcome.OperationalProfile.Effective() != d.OperationalProfileCompatibilityV1 {
			return d.Outcome{}, 0, d.NewError(d.ErrorCodeInvalidTransition, "only compatibility_v1 can be migrated to operational_v2")
		}
		nextVersion, err := outcome.Version.Next()
		if err != nil {
			return d.Outcome{}, 0, err
		}
		outcome.OperationalProfile = d.OperationalProfileOperationalV2
		outcome.CompletionPolicy = d.CompletionPolicyStrictWork
		outcome.Version = nextVersion
		outcome.UpdatedAt = transactionTime.UTC()
		if err := outcome.Validate(); err != nil {
			return d.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return d.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func authorizeOperationalPolicyAdmin(ctx context.Context, authorizer ports.Authorizer, scope d.Scope, principal string) error {
	identity, ok := IdentityFromContext(ctx)
	if !ok || principal == "" || identity.PrincipalID != principal || identity.NamespaceID != scope.NamespaceID {
		return d.NewError(d.ErrorCodeForbidden, "authenticated namespace administrator identity is required")
	}
	request := ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: principal, Permission: ports.PermissionNamespaceAdmin}
	if err := request.Validate(); err != nil {
		return err
	}
	return authorizer.Authorize(ctx, request)
}

func authorizeOperationalPolicyAdminInUnitOfWork(ctx context.Context, authorizer ports.Authorizer, uow ports.UnitOfWork, scope d.Scope, principal string) error {
	identity, ok := IdentityFromContext(ctx)
	if !ok || principal == "" || identity.PrincipalID != principal || identity.NamespaceID != scope.NamespaceID {
		return d.NewError(d.ErrorCodeForbidden, "authenticated namespace administrator identity is required")
	}
	request := ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: principal, Permission: ports.PermissionNamespaceAdmin}
	if err := request.Validate(); err != nil {
		return err
	}
	if dynamic, ok := authorizer.(ports.TransactionalAuthorizer); ok {
		return dynamic.AuthorizeInUnitOfWork(ctx, uow, request)
	}
	return nil
}
