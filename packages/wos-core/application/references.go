package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type ReferenceQuery struct {
	Kinds     []domain.EntityKind `json:"kinds"`
	Query     string              `json:"query,omitempty"`
	Lifecycle string              `json:"lifecycle,omitempty"`
	ID        domain.ID           `json:"id,omitempty"`
	Limit     int                 `json:"limit,omitempty"`
	Cursor    string              `json:"cursor,omitempty"`
}
type ReferencePage struct {
	Items           []ports.ReferenceCandidate `json:"items"`
	NextCursor      string                     `json:"next_cursor,omitempty"`
	OutcomeRevision domain.OutcomeRevision     `json:"outcome_revision"`
	Limit           int                        `json:"limit"`
	Consistency     string                     `json:"consistency"`
}

func referenceKindAllowed(k domain.EntityKind) bool {
	switch k {
	case domain.EntityKindOutcome, domain.EntityKindObjective, domain.EntityKindWorkItem,
		domain.EntityKindIssue, domain.EntityKindBlocker, domain.EntityKindEvidence,
		domain.EntityKindArtifact, domain.EntityKindDecision, domain.EntityKindRoadmap,
		domain.EntityKindRelation, domain.EntityKindEvidenceLink:
		return true
	}
	return false
}

// SearchReferences binds each page to the current Outcome revision. It returns
// metadata only; the eventual mutation rechecks reference/state/authority.
func (s *Service) SearchReferences(ctx context.Context, scope domain.Scope, q ReferenceQuery) (result ReferencePage, queryErr error) {
	started := time.Now()
	returnedItems := 0
	defer s.observeQuery(ctx, "reference_search", scope, &result.OutcomeRevision, &queryErr, started, &returnedItems)
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return result, err
	}
	if err := scope.Validate(); err != nil {
		return result, err
	}
	n, err := queryLimit(q.Limit)
	if err != nil {
		return result, err
	}
	if len(q.Query) > 512 || !utf8.ValidString(q.Query) || len(q.Lifecycle) > 64 || strings.ContainsRune(q.Query, 0) || strings.ContainsRune(q.Lifecycle, 0) {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid reference search text")
	}
	if len(q.Kinds) == 0 || len(q.Kinds) > 11 {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "reference kinds are required")
	}
	kinds := append([]domain.EntityKind(nil), q.Kinds...)
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	for i, k := range kinds {
		if !referenceKindAllowed(k) || (i > 0 && k == kinds[i-1]) {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid or duplicate reference kind")
		}
	}
	if !q.ID.IsZero() {
		if err := q.ID.Validate(); err != nil {
			return result, err
		}
		if q.Cursor != "" || q.Query != "" {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "reference resolution cannot use search or cursor")
		}
	}
	f := ports.ReferenceFilter{Kinds: kinds, Query: strings.TrimSpace(q.Query), Lifecycle: q.Lifecycle, ID: q.ID, Limit: n + 1}
	hash := filterHash(struct {
		Kinds                []domain.EntityKind
		Query, Lifecycle, ID string
	}{kinds, f.Query, q.Lifecycle, q.ID.String()})
	var cursor queryCursor
	if q.Cursor != "" {
		cursor, err = decodeCursor(q.Cursor, scope.NamespaceID, scope.OutcomeID, "references", hash)
		if err != nil {
			return result, err
		}
		parts := strings.Split(cursor.Key, "/")
		if len(parts) != 2 {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor: reference key")
		}
		f.AfterKind = domain.EntityKind(parts[0])
		f.AfterID, err = domain.ParseID(parts[1])
		if err != nil || !referenceKindAllowed(f.AfterKind) {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor: reference key")
		}
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer uow.Rollback()
	if err = s.authorizeScopedReadInUnitOfWork(ctx, uow, scope); err != nil {
		return result, err
	}
	if _, err = uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID); err != nil {
		return result, err
	}
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return result, err
	}
	if q.Cursor != "" && cursor.Revision != coord.Revision {
		return result, domain.NewError(domain.ErrorCodePreconditionFailed, "snapshot_changed: reload reference search")
	}
	repo, ok := uow.Outcomes().(ports.ReferenceSearchRepository)
	if !ok {
		return result, domain.NewError(domain.ErrorCodeInvalidConfig, "reference search repository required")
	}
	items, err := repo.SearchReferences(ctx, scope, f)
	if err != nil {
		return result, err
	}
	result = ReferencePage{Items: items, OutcomeRevision: coord.Revision, Limit: n, Consistency: "outcome_revision"}
	if len(items) > n {
		result.Items = items[:n]
		last := result.Items[n-1]
		result.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "references", Filter: hash, Revision: coord.Revision, Key: string(last.Ref.Kind) + "/" + last.Ref.ID.String()})
	}
	if result.Items == nil {
		result.Items = []ports.ReferenceCandidate{}
	}
	returnedItems = len(result.Items)
	return result, nil
}
