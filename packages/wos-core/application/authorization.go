package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
)

// Identity is resolved by a trusted adapter, never decoded from a domain payload.
type Identity struct {
	CredentialID             domain.ID
	CredentialPolicyRevision domain.Version
	PrincipalID              string
	NamespaceID              domain.ID
	CredentialDigest         string
	Actor                    domain.ActorRef
}
type identityKey struct{}

func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}

// NewAuthorizedService requires a trusted request identity for every public command and query.
// NewService remains the explicit trusted embedded composition used by local applications.
func NewAuthorizedService(tx ports.TransactionManager, clock ports.Clock, ids ports.IDGenerator, auth ports.Authorizer) (*Service, error) {
	s, err := NewServiceWithAuthorizer(tx, clock, ids, auth)
	if err != nil {
		return nil, err
	}
	s.requireIdentity = true
	return s, nil
}
func (s *Service) Authorize(ctx context.Context, namespace domain.ID, permission ports.Permission) error {
	id, ok := IdentityFromContext(ctx)
	if !ok || id.PrincipalID == "" {
		return domain.NewError(domain.ErrorCodeForbidden, "authenticated identity is required")
	}
	return s.authorizer.Authorize(ctx, ports.AuthorizationRequest{NamespaceID: namespace, PrincipalID: id.PrincipalID, Permission: permission})
}
func (s *Service) authorizeRead(ctx context.Context, namespace domain.ID) error {
	if err := s.authorizeNamespaceMetadataRead(ctx, namespace); err != nil {
		return err
	}
	if !s.requireIdentity {
		return nil
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer u.Rollback()
	allowed, err := s.readOutcomeRestrictions(ctx, u, namespace)
	if err != nil {
		return err
	}
	if len(allowed) != 0 {
		return domain.NewError(domain.ErrorCodeForbidden, "Outcome-restricted credential requires a scoped read")
	}
	return nil
}

func (s *Service) authorizeMutation(ctx context.Context, cc domain.CommandContext, meta commandMetadata) error {
	if !s.requireIdentity {
		return nil
	}
	id, ok := IdentityFromContext(ctx)
	if !ok || id.PrincipalID != cc.PrincipalID || id.Actor != cc.Actor {
		return domain.NewError(domain.ErrorCodeForbidden, "command identity does not match authenticated identity")
	}
	return s.Authorize(ctx, meta.NamespaceID, commandPermission(meta.Name))
}
func commandPermission(name string) ports.Permission {
	switch name {
	case "InterveneSignedReviewCase", "SetNamespaceWorkProtocol", "ReconcileExpiredWorkContracts":
		return ports.PermissionNamespaceAdmin
	case "AcquireSignedWorkContract", "AcquireNextSignedWorkContract", "AcquireWorkContract", "AcquireNextWorkContract":
		return ports.PermissionWorkContractAcquire
	case "RevokeSignedContract", "RevokeWorkContract":
		return ports.PermissionWorkContractRevoke
	case "RenewSignedWorkContract", "ResumeSignedWorkContract", "RenewWorkContract", "ResumeWorkContract", "SyncWorkContract", "SubmitWorkResult":
		return ports.PermissionWorkWrite
	case "ReturnSignedWork":
		return ports.PermissionWorkContractReturn
	case "ReturnSignedReview":
		return ports.PermissionWorkReviewDecide
	case "AcquireSignedReviewContract", "AcquireNextSignedReviewContract", "RenewSignedReviewContract", "ResumeSignedReviewContract":
		return ports.PermissionWorkReviewAcquire
	case "FinalizeWorkContract":
		return ports.PermissionConclusionWrite
	case "AdministrativeCancelWorkItem":
		return ports.PermissionWorkAdminCancel
	case "AdministrativeCompleteWorkItem":
		return ports.PermissionWorkAdminComplete
	case "RecordCriterionAssessment", "AttestCriterion":
		return ports.PermissionAssessmentWrite
	case "AchieveOutcome", "AchieveObjective", "ReopenOutcome", "ReopenObjective":
		return ports.PermissionConclusionWrite
	}
	if strings.Contains(name, "Trigger") || name == "RedeliverDelivery" {
		return ports.PermissionIntegrationWrite
	}
	if strings.Contains(name, "Roadmap") {
		return ports.PermissionPlanningWrite
	}
	if strings.Contains(name, "WorkItem") || strings.Contains(name, "Issue") || strings.Contains(name, "Blocker") {
		return ports.PermissionWorkWrite
	}
	if strings.Contains(name, "Artifact") || strings.Contains(name, "Evidence") || strings.Contains(name, "Decision") {
		return ports.PermissionRecordsWrite
	}
	return ports.PermissionOutcomeWrite
}

// CommandPermissionForPresentation exposes existing authorization metadata only.
// It neither grants authority nor replaces command authorization.
func CommandPermissionForPresentation(name string) ports.Permission { return commandPermission(name) }
