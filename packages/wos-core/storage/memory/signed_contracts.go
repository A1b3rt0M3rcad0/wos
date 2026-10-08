package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"time"
)

type memorySignedReviewState struct {
	Cases     map[string]d.ReviewCase
	Contracts map[string]d.ReviewContract
	Facts     map[string]d.SignedFact
}

func newMemorySignedReviewState() memorySignedReviewState {
	return memorySignedReviewState{Cases: map[string]d.ReviewCase{}, Contracts: map[string]d.ReviewContract{}, Facts: map[string]d.SignedFact{}}
}
func cloneSignedReview(s memorySignedReviewState) memorySignedReviewState { return deepCopy(s) }

type signedContractRepository struct{ tx *transaction }

func (tx *transaction) SignedContracts() ports.SignedContractRepository {
	return signedContractRepository{tx}
}
func (r signedContractRepository) ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.tx.ensureOpen()
}
func (r signedContractRepository) Case(ctx context.Context, scope d.Scope, id d.ID) (d.ReviewCase, error) {
	var zero d.ReviewCase
	if err := r.ready(ctx); err != nil {
		return zero, err
	}
	c, ok := r.tx.signedReview.Cases[entityKey(scope, id)]
	if !ok {
		return zero, missingSigning()
	}
	return deepCopy(c), c.Validate()
}
func (r signedContractRepository) Cases(ctx context.Context, scope d.Scope, f ports.ReviewFilter) ([]d.ReviewCase, error) {
	if err := r.ready(ctx); err != nil {
		return nil, err
	}
	if f.Limit < 1 || f.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "review case page outside 1..101")
	}
	out := []d.ReviewCase{}
	for _, c := range r.tx.signedReview.Cases {
		if (!f.OpenOnly || c.Status == d.ReviewPending || c.Status == d.ReviewInReview) && c.Scope == scope && c.ID > f.After && (f.WorkItemID.IsZero() || f.WorkItemID == c.WorkItemID) && (f.Status == "" || f.Status == string(c.Status)) {
			out = append(out, deepCopy(c))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}
func (r signedContractRepository) OpenCase(ctx context.Context, scope d.Scope, work d.ID) (*d.ReviewCase, error) {
	if err := r.ready(ctx); err != nil {
		return nil, err
	}
	for _, c := range r.tx.signedReview.Cases {
		if c.Scope == scope && c.WorkItemID == work && c.Status.Open() {
			copy := deepCopy(c)
			return &copy, nil
		}
	}
	return nil, nil
}
func (r signedContractRepository) InsertCase(ctx context.Context, c d.ReviewCase) error {
	if err := r.ready(ctx); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Status != d.ReviewPending || c.Version != 1 || c.CurrentContractID != nil {
		return d.NewError(d.ErrorCodeInvalidTransition, "new review must be pending")
	}
	if _, ok := r.tx.signedReview.Cases[entityKey(c.Scope, c.ID)]; ok {
		return d.NewError(d.ErrorCodeAlreadyExists, "review case exists")
	}
	if current, err := r.OpenCase(ctx, c.Scope, c.WorkItemID); err != nil {
		return err
	} else if current != nil {
		return d.NewError(d.ErrorCodeAlreadyExists, "work already has open review")
	}
	work, err := r.tx.SignedWorkContracts().Get(ctx, c.Scope, c.WorkContractID)
	if err != nil {
		return err
	}
	p, err := r.tx.SignedWorkContracts().GetSubmission(ctx, c.Scope, c.SubmissionID)
	if err != nil {
		return err
	}
	if work.Status != d.ContractDelivered || work.WorkItemID != c.WorkItemID || work.LatestSubmissionID == nil || *work.LatestSubmissionID != c.SubmissionID || p.Digest != c.SubmissionDigest || p.Material.ContractID != work.ID {
		return d.NewError(d.ErrorCodeInvalidScope, "review immutable delivery target differs")
	}
	r.tx.signedReview.Cases[entityKey(c.Scope, c.ID)] = deepCopy(c)
	return nil
}
func (r signedContractRepository) SaveCase(ctx context.Context, c d.ReviewCase, expected d.Version) error {
	old, err := r.Case(ctx, c.Scope, c.ID)
	if err != nil {
		return err
	}
	if old.Version != expected {
		return d.NewError(d.ErrorCodeVersionConflict, "review case CAS differs")
	}
	if err = d.ValidateReviewCaseUpdate(old, c); err != nil {
		return err
	}
	r.tx.signedReview.Cases[entityKey(c.Scope, c.ID)] = deepCopy(c)
	return nil
}
func (r signedContractRepository) ReviewContract(ctx context.Context, scope d.Scope, id d.ID) (d.ReviewContract, error) {
	var zero d.ReviewContract
	if err := r.ready(ctx); err != nil {
		return zero, err
	}
	c, ok := r.tx.signedReview.Contracts[entityKey(scope, id)]
	if !ok {
		return zero, missingSigning()
	}
	return deepCopy(c), c.Validate()
}
func (r signedContractRepository) ReviewContracts(ctx context.Context, scope d.Scope, f ports.ReviewFilter) ([]d.ReviewContract, error) {
	if err := r.ready(ctx); err != nil {
		return nil, err
	}
	if f.Limit < 1 || f.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "review contract page outside 1..101")
	}
	out := []d.ReviewContract{}
	for _, c := range r.tx.signedReview.Contracts {
		if c.Scope == scope && c.ID > f.After && (f.WorkItemID.IsZero() || f.WorkItemID == c.WorkItemID) && (f.HolderPrincipalID == "" || f.HolderPrincipalID == c.HolderPrincipalID) && (f.Status == "" || f.Status == string(c.Status)) {
			out = append(out, deepCopy(c))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}
func (r signedContractRepository) InsertReviewContract(ctx context.Context, c d.ReviewContract) error {
	if err := r.ready(ctx); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Status != d.ContractActive || c.Version != 1 || c.LeaseVersion != 1 {
		return d.NewError(d.ErrorCodeInvalidTransition, "new review authority must be active")
	}
	if _, ok := r.tx.signedReview.Contracts[entityKey(c.Scope, c.ID)]; ok {
		return d.NewError(d.ErrorCodeAlreadyExists, "review authority exists")
	}
	for _, old := range r.tx.signedReview.Contracts {
		if old.Scope == c.Scope && old.CaseID == c.CaseID && old.Status == d.ContractActive {
			return d.NewError(d.ErrorCodeAlreadyExists, "case already has active review authority")
		}
	}
	target, err := r.Case(ctx, c.Scope, c.CaseID)
	if err != nil {
		return err
	}
	if err = target.Bind(c, c.AcquiredAt); err != nil {
		return err
	}
	r.tx.signedReview.Contracts[entityKey(c.Scope, c.ID)] = deepCopy(c)
	return nil
}
func (r signedContractRepository) SaveReviewContract(ctx context.Context, c d.ReviewContract, version, lease d.Version) error {
	old, err := r.ReviewContract(ctx, c.Scope, c.ID)
	if err != nil {
		return err
	}
	if old.Version != version || old.LeaseVersion != lease {
		return d.NewError(d.ErrorCodeVersionConflict, "review authority CAS differs")
	}
	if err = d.ValidateReviewContractUpdate(old, c); err != nil {
		return err
	}
	r.tx.signedReview.Contracts[entityKey(c.Scope, c.ID)] = deepCopy(c)
	return nil
}
func (r signedContractRepository) Fact(ctx context.Context, scope d.Scope, id d.ID) (d.SignedFact, error) {
	var zero d.SignedFact
	if err := r.ready(ctx); err != nil {
		return zero, err
	}
	f, ok := r.tx.signedReview.Facts[entityKey(scope, id)]
	if !ok {
		return zero, missingSigning()
	}
	return deepCopy(f), f.Validate()
}
func (r signedContractRepository) Facts(ctx context.Context, scope d.Scope, contract d.ID, kind string, after d.ID, limit int) ([]d.SignedFact, error) {
	if err := r.ready(ctx); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "signed fact page outside 1..101")
	}
	out := []d.SignedFact{}
	for _, f := range r.tx.signedReview.Facts {
		if f.Scope == scope && f.ContractID == contract && (kind == "" || f.Kind == kind) && f.ID > after {
			out = append(out, deepCopy(f))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (r signedContractRepository) InsertFact(ctx context.Context, f d.SignedFact) error {
	if err := r.ready(ctx); err != nil {
		return err
	}
	if err := f.Validate(); err != nil {
		return err
	}
	if f.ContractKind == "execution" {
		if _, err := r.tx.SignedWorkContracts().Get(ctx, f.Scope, f.ContractID); err != nil {
			return err
		}
	} else {
		if _, err := r.ReviewContract(ctx, f.Scope, f.ContractID); err != nil {
			return err
		}
	}
	key, err := r.tx.SigningIdentity().Key(ctx, f.Scope.NamespaceID, f.KeyID)
	if err != nil {
		return err
	}
	if (f.Kind == "return" && (key.Purpose != "agent" || key.PrincipalID != f.PrincipalID)) || (f.Kind != "return" && key.Purpose != "issuer") {
		return d.NewError(d.ErrorCodeForbidden, "signed fact key role differs")
	}
	for _, old := range r.tx.signedReview.Facts {
		if old.ID == f.ID || (old.Scope.NamespaceID == f.Scope.NamespaceID && old.Kind == f.Kind && (f.Kind == "return" || f.Kind == "acceptance") && old.PrincipalID == f.PrincipalID && old.IdempotencyKey == f.IdempotencyKey) || (f.Kind == "return" && old.Kind == "return" && old.Scope.NamespaceID == f.Scope.NamespaceID && old.RequestID != nil && f.RequestID != nil && *old.RequestID == *f.RequestID) || (f.Kind == "specification" && old.Kind == f.Kind && old.Scope == f.Scope && old.ContractID == f.ContractID) {
			return d.NewError(d.ErrorCodeAlreadyExists, "immutable signed fact/request already exists")
		}
	}
	r.tx.signedReview.Facts[entityKey(f.Scope, f.ID)] = deepCopy(f)
	return nil
}
func (r signedContractRepository) Acceptance(ctx context.Context, ns d.ID, principal, key string) (d.SignedFact, error) {
	var zero d.SignedFact
	if err := r.ready(ctx); err != nil {
		return zero, err
	}
	for _, f := range r.tx.signedReview.Facts {
		if f.Scope.NamespaceID == ns && f.PrincipalID == principal && f.IdempotencyKey == key && f.Kind == "acceptance" {
			return deepCopy(f), f.Validate()
		}
	}
	return zero, missingSigning()
}
func (r signedContractRepository) Counts(ctx context.Context, ns d.ID, principal string, credential d.ID, now time.Time) (ports.SignedActiveCounts, error) {
	var result ports.SignedActiveCounts
	if err := r.ready(ctx); err != nil {
		return result, err
	}
	for _, c := range r.tx.signedContracts {
		if c.Scope.NamespaceID == ns && c.ValidAt(now) {
			result.NamespaceWork++
			if c.HolderPrincipalID == principal {
				result.PrincipalWork++
			}
			if c.SignedBinding != nil && c.SignedBinding.CredentialID == credential {
				result.CredentialWork++
			}
		}
	}
	for _, c := range r.tx.signedReview.Contracts {
		if c.Scope.NamespaceID == ns && c.ValidAt(now) {
			result.NamespaceReview++
			if c.HolderPrincipalID == principal {
				result.PrincipalReview++
			}
			if c.Binding.CredentialID == credential {
				result.CredentialReview++
			}
		}
	}
	return result, nil
}

var _ ports.SignedContractUnitOfWork = (*transaction)(nil)
