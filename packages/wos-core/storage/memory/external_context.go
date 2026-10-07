package memory

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"strings"
)

type externalContextRepository struct{ tx *transaction }

func (t *transaction) ExternalContexts() ports.ExternalContextRepository {
	return externalContextRepository{tx: t}
}
func contextKey(scope domain.Scope) string {
	return scope.NamespaceID.String() + "/" + scope.OutcomeID.String()
}
func (r externalContextRepository) List(ctx context.Context, scope domain.Scope) ([]domain.ExternalReference, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v := append([]domain.ExternalReference{}, r.tx.externalContexts[contextKey(scope)]...)
	sort.Slice(v, func(i, j int) bool {
		return v[i].Provider+"\x00"+v[i].Kind+"\x00"+v[i].ExternalID < v[j].Provider+"\x00"+v[j].Kind+"\x00"+v[j].ExternalID
	})
	return v, nil
}
func (r externalContextRepository) Add(ctx context.Context, scope domain.Scope, v domain.ExternalReference) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	k := contextKey(scope)
	for key, refs := range r.tx.externalContexts {
		if !strings.HasPrefix(key, scope.NamespaceID.String()+"/") {
			continue
		}
		for _, ref := range refs {
			if ref.Provider == v.Provider && ref.Kind == v.Kind && ref.ExternalID == v.ExternalID {
				return domain.NewError(domain.ErrorCodeAlreadyExists, "external address already linked")
			}
		}
	}
	r.tx.externalContexts[k] = append(r.tx.externalContexts[k], v)
	return nil
}
func (r externalContextRepository) Remove(ctx context.Context, scope domain.Scope, v domain.ExternalReference) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	k := contextKey(scope)
	values := r.tx.externalContexts[k]
	for i, ref := range values {
		if ref.Provider == v.Provider && ref.Kind == v.Kind && ref.ExternalID == v.ExternalID {
			r.tx.externalContexts[k] = append(values[:i], values[i+1:]...)
			return nil
		}
	}
	return domain.NewError(domain.ErrorCodeNotFound, "external reference not found")
}
func cloneContexts(src map[string][]domain.ExternalReference) map[string][]domain.ExternalReference {
	dst := map[string][]domain.ExternalReference{}
	for k, v := range src {
		dst[k] = append([]domain.ExternalReference{}, v...)
	}
	return dst
}
