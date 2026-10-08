package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"reflect"
	"sort"
)

func contractCopy[T any](v T) T {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}
func cloneContractMap(src map[string]d.WorkContract) map[string]d.WorkContract {
	out := make(map[string]d.WorkContract, len(src))
	for k, v := range src {
		out[k] = contractCopy(v)
	}
	return out
}
func cloneCheckpointMap(src map[string]d.WorkCheckpoint) map[string]d.WorkCheckpoint {
	out := make(map[string]d.WorkCheckpoint, len(src))
	for k, v := range src {
		out[k] = contractCopy(v)
	}
	return out
}
func cloneSubmissionMap(src map[string]d.WorkSubmission) map[string]d.WorkSubmission {
	out := make(map[string]d.WorkSubmission, len(src))
	for k, v := range src {
		out[k] = contractCopy(v)
	}
	return out
}

type workContractRepository struct{ tx *transaction }

func (tx *transaction) WorkContracts() ports.WorkContractRepository {
	return workContractRepository{tx}
}
func (r workContractRepository) Get(ctx context.Context, scope d.Scope, id d.ID) (d.WorkContract, error) {
	if err := ctx.Err(); err != nil {
		return d.WorkContract{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.WorkContract{}, err
	}
	v, ok := r.tx.contracts[entityKey(scope, id)]
	if !ok {
		return d.WorkContract{}, d.NewError(d.ErrorCodeNotFound, "contract not found")
	}
	return contractCopy(v), nil
}
func (r workContractRepository) Current(ctx context.Context, scope d.Scope, work d.ID) (*d.WorkContract, error) {
	list, err := r.List(ctx, scope, ports.ContractFilter{WorkItemID: work, Status: d.ContractActive, Limit: 2})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	c := list[0]
	return &c, nil
}
func (r workContractRepository) List(ctx context.Context, scope d.Scope, f ports.ContractFilter) ([]d.WorkContract, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if f.Limit < 1 || f.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "contract limit outside 1..101")
	}
	out := []d.WorkContract{}
	for _, v := range r.tx.contracts {
		if v.Scope == scope && (f.WorkItemID == "" || f.WorkItemID == v.WorkItemID) && (f.HolderPrincipalID == "" || f.HolderPrincipalID == v.HolderPrincipalID) && (f.Status == "" || f.Status == v.Status) && v.ID > f.After {
			out = append(out, contractCopy(v))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}
func (r workContractRepository) Insert(ctx context.Context, c d.WorkContract) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}
	key := entityKey(c.Scope, c.ID)
	if _, ok := r.tx.contracts[key]; ok {
		return d.NewError(d.ErrorCodeAlreadyExists, "contract already exists")
	}
	if _, err := r.tx.WorkItems().Get(ctx, c.Scope, c.WorkItemID); err != nil {
		return err
	}
	old, err := r.Current(ctx, c.Scope, c.WorkItemID)
	if err != nil {
		return err
	}
	if old != nil {
		return d.NewError(d.ErrorCodeWorkAlreadyClaimed, "an open contract exists")
	}
	r.tx.contracts[key] = contractCopy(c)
	return nil
}
func (r workContractRepository) Save(ctx context.Context, c d.WorkContract, version, lease d.Version) error {
	if err := c.Validate(); err != nil {
		return err
	}
	old, err := r.Get(ctx, c.Scope, c.ID)
	if err != nil {
		return err
	}
	if old.Version != version || old.LeaseVersion != lease {
		return d.NewError(d.ErrorCodeVersionConflict, "contract CAS conflict")
	}
	if err := d.ValidateContractUpdate(old, c); err != nil {
		return err
	}
	r.tx.contracts[entityKey(c.Scope, c.ID)] = contractCopy(c)
	return nil
}
func (r workContractRepository) InsertCheckpoint(ctx context.Context, p d.WorkCheckpoint) error {
	if err := p.Validate(); err != nil {
		return err
	}
	c, err := r.Get(ctx, p.Scope, p.ContractID)
	if err != nil {
		return err
	}
	if c.WorkItemID != p.WorkItemID {
		return d.NewError(d.ErrorCodeInvalidScope, "checkpoint task mismatch")
	}
	for _, old := range r.tx.checkpoints {
		if old.Scope == p.Scope && old.ContractID == p.ContractID && old.Sequence == p.Sequence {
			return d.NewError(d.ErrorCodeAlreadyExists, "checkpoint sequence exists")
		}
	}
	key := entityKey(p.Scope, p.ID)
	if old, ok := r.tx.checkpoints[key]; ok {
		if reflect.DeepEqual(old, p) {
			return nil
		}
		return d.NewError(d.ErrorCodeAlreadyExists, "checkpoint immutable")
	}
	r.tx.checkpoints[key] = contractCopy(p)
	return nil
}
func (r workContractRepository) ListCheckpoints(ctx context.Context, scope d.Scope, contract, after d.ID, limit int) ([]d.WorkCheckpoint, error) {
	if _, err := r.Get(ctx, scope, contract); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "checkpoint limit outside 1..101")
	}
	out := []d.WorkCheckpoint{}
	for _, v := range r.tx.checkpoints {
		if v.Scope == scope && v.ContractID == contract && v.ID > after {
			out = append(out, contractCopy(v))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (r workContractRepository) InsertSubmission(ctx context.Context, p d.WorkSubmission) error {
	if err := p.Validate(); err != nil {
		return err
	}
	c, err := r.Get(ctx, p.Scope, p.Material.ContractID)
	if err != nil {
		return err
	}
	if c.WorkItemID != p.Material.WorkItemID {
		return d.NewError(d.ErrorCodeInvalidScope, "submission task mismatch")
	}
	key := entityKey(p.Scope, p.ID)
	if old, ok := r.tx.submissions[key]; ok {
		if reflect.DeepEqual(old, p) {
			return nil
		}
		return d.NewError(d.ErrorCodeAlreadyExists, "submission immutable")
	}
	r.tx.submissions[key] = contractCopy(p)
	return nil
}
func (r workContractRepository) GetSubmission(ctx context.Context, scope d.Scope, id d.ID) (d.WorkSubmission, error) {
	if err := ctx.Err(); err != nil {
		return d.WorkSubmission{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.WorkSubmission{}, err
	}
	v, ok := r.tx.submissions[entityKey(scope, id)]
	if !ok {
		return d.WorkSubmission{}, d.NewError(d.ErrorCodeNotFound, "submission not found")
	}
	return contractCopy(v), nil
}
func (r workContractRepository) ListSubmissions(ctx context.Context, scope d.Scope, contract, after d.ID, limit int) ([]d.WorkSubmission, error) {
	if _, err := r.Get(ctx, scope, contract); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "submission limit outside 1..101")
	}
	out := []d.WorkSubmission{}
	for _, v := range r.tx.submissions {
		if v.Scope == scope && v.Material.ContractID == contract && v.ID > after {
			out = append(out, contractCopy(v))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r workItemRepository) ContractCandidates(ctx context.Context, scope d.Scope, q ports.ContractCandidateQuery) ([]ports.ContractCandidate, error) {
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if q.Limit < 1 || q.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "candidate limit outside 1..101")
	}
	out := []ports.ContractCandidate{}
	for _, w := range r.tx.workItems {
		if w.Scope != scope || !w.ContractsEnabled || (w.Lifecycle != d.WorkItemLifecycleTodo && w.Lifecycle != d.WorkItemLifecycleInProgress) {
			continue
		}
		rank := ports.ContractPriorityRank(w.Priority)
		if q.AfterID != "" && !(rank > q.AfterPriority || rank == q.AfterPriority && (w.CreatedAt.After(q.AfterCreated) || w.CreatedAt.Equal(q.AfterCreated) && w.ID > q.AfterID)) {
			continue
		}
		out = append(out, ports.ContractCandidate{ID: w.ID, Version: w.Version, Priority: w.Priority, CreatedAt: w.CreatedAt})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		ar, br := ports.ContractPriorityRank(a.Priority), ports.ContractPriorityRank(b.Priority)
		if ar != br {
			return ar < br
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.ID < b.ID
	})
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (r idempotencyStore) LookupReceipt(ctx context.Context, ns d.ID, principal string, command d.ID) (d.StoredCommandResult, error) {
	if err := ctx.Err(); err != nil {
		return d.StoredCommandResult{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.StoredCommandResult{}, err
	}
	for _, v := range r.tx.idempotency {
		if v.Identity.NamespaceID == ns && v.Identity.PrincipalID == principal && v.Completed && v.Result.CommandID == command {
			return contractCopy(v.Result), nil
		}
	}
	return d.StoredCommandResult{}, d.NewError(d.ErrorCodeNotFound, "command receipt not retained")
}
