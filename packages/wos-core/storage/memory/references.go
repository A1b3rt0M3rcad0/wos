package memory

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"strings"
)

func (r outcomeRepository) SearchReferences(ctx context.Context, scope domain.Scope, f ports.ReferenceFilter) ([]ports.ReferenceCandidate, error) {
	if f.Limit < 1 || f.Limit > 101 {
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid reference repository limit")
	}
	allowed := map[domain.EntityKind]bool{}
	for _, k := range f.Kinds {
		allowed[k] = true
	}
	result := []ports.ReferenceCandidate{}
	add := func(ref domain.EntityRef, version domain.Version, title, lifecycle, context string) {
		if ref.Scope != scope || !allowed[ref.Kind] || (!f.ID.IsZero() && ref.ID != f.ID) || (!f.AfterID.IsZero() && (ref.Kind < f.AfterKind || (ref.Kind == f.AfterKind && ref.ID <= f.AfterID))) || (f.Lifecycle != "" && lifecycle != f.Lifecycle) || !strings.Contains(foldReferenceASCII(title), foldReferenceASCII(f.Query)) {
			return
		}
		item := ports.ReferenceCandidate{Ref: ref, Version: version, Title: referenceTitle(title), Lifecycle: lifecycle, DisplayContext: referenceTitle(context)}
		index := sort.Search(len(result), func(i int) bool {
			return result[i].Ref.Kind > ref.Kind || (result[i].Ref.Kind == ref.Kind && result[i].Ref.ID >= ref.ID)
		})
		if index >= f.Limit {
			return
		}
		result = append(result, ports.ReferenceCandidate{})
		copy(result[index+1:], result[index:])
		result[index] = item
		if len(result) > f.Limit {
			result = result[:f.Limit]
		}
	}
	for _, v := range r.tx.outcomes {
		add(v.Ref(), v.Version, string(v.Title), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.objectives {
		context := ""
		if v.ParentObjectiveID != nil {
			if o, ok := r.tx.objectives[entityKey(scope, *v.ParentObjectiveID)]; ok {
				context = o.Title
			}
		}
		add(v.Ref(), v.Version, string(v.Title), string(v.Lifecycle), context)
	}
	for _, v := range r.tx.workItems {
		context := ""
		if v.ObjectiveID != nil {
			if o, ok := r.tx.objectives[entityKey(scope, *v.ObjectiveID)]; ok {
				context = o.Title
			}
		}
		add(v.Ref(), v.Version, string(v.Title), string(v.Lifecycle), context)
	}
	for _, v := range r.tx.issues {
		add(v.Ref(), v.Version, string(v.Title), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.blockers {
		add(v.Ref(), v.Version, string(v.Description), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.artifacts {
		add(v.Ref(), v.Version, string(v.Name), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.evidence {
		add(v.Ref(), v.Version, string(v.Description), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.decisions {
		add(v.Ref(), v.Version, string(v.Title), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.roadmaps {
		add(v.Ref(), v.Version, string(v.Title), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.relations {
		add(v.Ref(), v.Version, string(v.RelationType), string(v.Lifecycle), "")
	}
	for _, v := range r.tx.evidenceLinks {
		add(v.Ref(), v.Version, string(v.Rationale), string(v.Lifecycle), "")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
func referenceTitle(s string) string {
	r := []rune(s)
	if len(r) > 128 {
		return string(r[:128])
	}
	return s
}

func foldReferenceASCII(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return r
	}, s)
}
