package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Historical acceptance remains replayable after key retirement, but not by a
// different credential sharing the same Principal. This check is needed when
// the original operation registry/short cache did not yet exist or has no row.
func (s *Service) requireSignedAcceptedCredential(ctx context.Context, u ports.UnitOfWork, scope d.Scope, kind string, id d.ID) error {
	permission := ports.PermissionWorkContractReturn
	if kind == "review" {
		permission = ports.PermissionWorkReviewDecide
	}
	identity, _, e := s.signedAccess(ctx, u, scope, permission, "")
	if e != nil {
		return e
	}
	repo, work, e := signedRepository(u)
	if e != nil {
		return e
	}
	var credential d.ID
	switch kind {
	case "execution":
		c, e := work.Get(ctx, scope, id)
		if e != nil {
			return e
		}
		if c.SignedBinding == nil {
			return d.NewError(d.ErrorCodeSignedProtocolRequired, "signed accepted execution required")
		}
		credential = c.SignedBinding.CredentialID
	case "review":
		c, e := repo.ReviewContract(ctx, scope, id)
		if e != nil {
			return e
		}
		credential = c.Binding.CredentialID
	default:
		return d.NewError(d.ErrorCodeInvalidArgument, "accepted contract kind differs")
	}
	if credential != identity.CredentialID {
		return d.NewError(d.ErrorCodeForbidden, "historical acceptance belongs to another credential")
	}
	return nil
}
