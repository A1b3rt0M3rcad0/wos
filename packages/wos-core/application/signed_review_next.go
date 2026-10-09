package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type AcquireNextSignedReviewContractCommand struct {
	Scope       d.Scope
	SignerKeyID d.ID
	TTLSeconds  int    `wos:"optional"`
	Limit       int    `wos:"optional"`
	Cursor      string `wos:"optional"`
}
type SignedReviewAcquisition struct {
	Acquired       bool                        `json:"acquired"`
	Result         *SignedReviewContractResult `json:"result,omitempty"`
	SearchComplete bool                        `json:"search_complete"`
	NextCursor     string                      `json:"next_cursor,omitempty"`
	Reasons        []string                    `json:"reasons"`
}

func (s *Service) AcquireNextSignedReviewContract(ctx context.Context, cc d.CommandContext, cmd AcquireNextSignedReviewContractCommand) (MutationResult[SignedReviewAcquisition], error) {
	var zero MutationResult[SignedReviewAcquisition]
	if e := cc.Validate(); e != nil {
		return zero, e
	}
	if cc.IdempotencyKey == "" || cmd.Scope.Validate() != nil || cmd.SignerKeyID.Validate() != nil {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "review search requires independent intention, scope and agent key")
	}
	limit, e := queryLimit(cmd.Limit)
	if e != nil {
		return zero, e
	}
	filter := ports.ReviewFilter{Limit: limit + 1, OpenOnly: true}
	if cmd.Cursor != "" {
		cursor, e := decodeCursor(cmd.Cursor, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID, "signed_review_candidates", "")
		if e != nil {
			return zero, e
		}
		filter.After, e = d.ParseID(cursor.Key)
		if e != nil {
			return zero, e
		}
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedReviewAcquisition, d.OutcomeRevision, error) {
		result := SignedReviewAcquisition{Reasons: []string{}}
		identity, _, e := s.signedAccess(ctx, u, cmd.Scope, ports.PermissionWorkReviewAcquire, cmd.SignerKeyID)
		if e != nil {
			return result, 0, e
		}
		if _, e = s.requireSignedIssuer(ctx, u, cmd.Scope.NamespaceID); e != nil {
			return result, 0, e
		}
		if e = requireActiveOutcome(ctx, u, cmd.Scope); e != nil {
			return result, 0, e
		}
		protocol, e := protocolRepository(u)
		if e != nil {
			return result, 0, e
		}
		state, e := protocol.Lock(ctx, cmd.Scope.NamespaceID, true)
		if e != nil {
			return result, 0, e
		}
		if _, e = state.LeasePolicy.TTL(cmd.TTLSeconds); e != nil {
			return result, 0, e
		}
		coord, e := u.Coordination().LockOutcome(ctx, cmd.Scope)
		if e != nil {
			return result, 0, e
		}
		repo, _, e := signedRepository(u)
		if e != nil {
			return result, 0, e
		}
		registry, e := signingRepository(u)
		if e != nil {
			return result, 0, e
		}
		group, e := registry.PrincipalGroup(ctx, cmd.Scope.NamespaceID, identity.PrincipalID)
		if e != nil {
			return result, 0, e
		}
		cases, e := repo.Cases(ctx, cmd.Scope, filter)
		if e != nil {
			return result, 0, e
		}
		result.SearchComplete = len(cases) <= limit
		if len(cases) > limit {
			cases = cases[:limit]
			result.NextCursor = encodeCursor(queryCursor{Namespace: cmd.Scope.NamespaceID, Outcome: cmd.Scope.OutcomeID, Section: "signed_review_candidates", Key: cases[len(cases)-1].ID.String()})
		}
		now, e := securityTransactionTime(ctx, u, s.clock)
		if e != nil {
			return result, 0, e
		}
		for _, review := range cases {
			if review.CurrentContractID != nil {
				active, e := repo.ReviewContract(ctx, cmd.Scope, *review.CurrentContractID)
				if e != nil {
					return result, 0, e
				}
				if active.ValidAt(now) {
					continue
				}
			}
			if e = s.requireSignedReviewIndependence(ctx, u, review, identity.PrincipalID, group); e != nil {
				if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeForbidden {
					continue
				}
				return result, 0, e
			}
			acquired, revision, e := s.acquireSignedReview(ctx, u, cc, AcquireSignedReviewContractCommand{Scope: cmd.Scope, ReviewCaseID: review.ID, ExpectedReviewCaseVersion: review.Version, SignerKeyID: cmd.SignerKeyID, TTLSeconds: cmd.TTLSeconds})
			if e != nil {
				return result, 0, e
			}
			result.Acquired = true
			result.Result = &acquired
			return result, revision, nil
		}
		if result.SearchComplete {
			result.Reasons = append(result.Reasons, "no_eligible_review")
		} else {
			result.Reasons = append(result.Reasons, "review_search_incomplete")
		}
		return result, coord.Revision, nil
	})
}
