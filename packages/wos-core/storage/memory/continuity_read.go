package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type continuityReadRepository struct{ tx *transaction }

func (tx *transaction) ContinuityReads() ports.ContinuityReadRepository {
	return continuityReadRepository{tx}
}
func (r continuityReadRepository) Page(ctx context.Context, scope d.Scope, section, after string, limit int) (ports.ContinuityReadPage, error) {
	var zero ports.ContinuityReadPage
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return zero, e
	}
	if e := scope.Validate(); e != nil {
		return zero, e
	}
	if !ports.DirectContinuitySection(section) {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "unknown direct continuity section")
	}
	b, e := ports.NewContinuityPageBuilder(after, limit)
	if e != nil {
		return zero, e
	}
	summary := func(ref d.EntityRef, v d.Version, title, lifecycle string, priority d.Priority) error {
		return b.Add(ref.ID.String(), ports.Summarize(ref, v, title, lifecycle, priority))
	}
	switch section {
	case "work_items":
		for _, v := range r.tx.workItems {
			if v.Scope == scope {
				if e = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority); e != nil {
					return zero, e
				}
			}
		}
	case "objectives":
		for _, v := range r.tx.objectives {
			if v.Scope == scope {
				if e = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority); e != nil {
					return zero, e
				}
			}
		}
	case "issues":
		for _, v := range r.tx.issues {
			if v.Scope == scope {
				if e = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), ""); e != nil {
					return zero, e
				}
			}
		}
	case "blockers":
		for _, v := range r.tx.blockers {
			if v.Scope == scope {
				if e = summary(v.Ref(), v.Version, v.Description, string(v.Lifecycle), ""); e != nil {
					return zero, e
				}
			}
		}
	case "artifacts":
		for _, v := range r.tx.artifacts {
			if v.Scope == scope {
				if e = summary(v.Ref(), v.Version, v.Name, string(v.Lifecycle), ""); e != nil {
					return zero, e
				}
			}
		}
	case "evidence":
		for _, v := range r.tx.evidence {
			if v.Scope == scope {
				if e = summary(v.Ref(), v.Version, v.Description, string(v.Lifecycle), ""); e != nil {
					return zero, e
				}
			}
		}
	case "decisions":
		for _, v := range r.tx.decisions {
			if v.Scope == scope {
				if e = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), ""); e != nil {
					return zero, e
				}
			}
		}
	case "relations":
		for _, v := range r.tx.relations {
			if v.Scope == scope {
				if e = b.Add(v.ID.String(), v); e != nil {
					return zero, e
				}
			}
		}
	case "evidence_links":
		for _, v := range r.tx.evidenceLinks {
			if v.Scope == scope {
				if e = b.Add(v.ID.String(), v); e != nil {
					return zero, e
				}
			}
		}
	case "external_references":
		for _, v := range r.tx.externalContexts[contextKey(scope)] {
			raw, e := json.Marshal(v)
			if e != nil {
				return zero, e
			}
			if e = b.Add(string(raw), v); e != nil {
				return zero, e
			}
		}
	}
	if !b.HasAfter() {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: section key is missing")
	}
	return b.Page(), nil
}

var _ ports.ContinuityReadUnitOfWork = (*transaction)(nil)
