package application

import (
	"context"
	"encoding/hex"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) directContinuitySection(ctx context.Context, scope d.Scope, section string, n int, rawCursor string) (SectionPage, bool, error) {
	var zero SectionPage
	if e := scope.Validate(); e != nil {
		return zero, true, e
	}
	if e := s.authorizeScopedRead(ctx, scope); e != nil {
		return zero, true, e
	}
	var cursor queryCursor
	if rawCursor != "" {
		var e error
		cursor, e = decodeCursor(rawCursor, scope.NamespaceID, scope.OutcomeID, section, "")
		if e != nil {
			return zero, true, e
		}
		if cursor.CursorVersion != 0 && cursor.CursorVersion != 2 {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: unsupported section cursor version")
		}
		if cursor.EvaluatedAt.IsZero() {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: evaluation time")
		}
	}
	u, e := s.beginReadSnapshot(ctx)
	if e != nil {
		return zero, true, e
	}
	defer u.Rollback()
	if e = s.authorizeScopedReadInUnitOfWork(ctx, u, scope); e != nil {
		return zero, true, e
	}
	reads, ok := u.(ports.ContinuityReadUnitOfWork)
	if !ok {
		return zero, false, nil
	} // compatibility for embedded adapters; official adapters provide the port
	repo := reads.ContinuityReads()
	if _, e = repo.Outcome(ctx, scope); e != nil {
		return zero, true, e
	}
	coord, e := u.Coordination().LockOutcome(ctx, scope)
	if e != nil {
		return zero, true, e
	}
	now, e := s.transactionTime(ctx, u)
	if e != nil {
		return zero, true, e
	}
	at := now
	if rawCursor != "" {
		if cursor.Revision != coord.Revision {
			return zero, true, d.NewError(d.ErrorCodePreconditionFailed, "snapshot_changed: reload continuity snapshot")
		}
		if cursor.EvaluatedAt.After(now) {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: evaluation time")
		}
		at = cursor.EvaluatedAt
	}
	after := cursor.Key
	if rawCursor != "" && cursor.CursorVersion == 0 {
		digest, e := hex.DecodeString(cursor.Key)
		if e != nil || len(digest) != 32 {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: legacy section digest")
		}
		if section == "external_references" {
			after = "h:" + cursor.Key
		} else {
			// Old clients carry an item digest, not its ID. Stream bounded metadata
			// pages in the same snapshot; never rehydrate histories or other sections.
			after, e = legacyContinuityKey(ctx, repo, scope, section, cursor.Key)
			if e != nil {
				return zero, true, e
			}
		}
	}
	if rawCursor != "" && cursor.CursorVersion == 2 && section == "external_references" {
		if len(after) != 66 || after[:2] != "h:" {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: external reference digest")
		}
		if value, e := hex.DecodeString(after[2:]); e != nil || len(value) != 32 {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: external reference digest")
		}
	}
	page, e := repo.Page(ctx, scope, section, after, n+1)
	if e != nil {
		return zero, true, e
	}
	result := SectionPage{Items: []json.RawMessage{}, Count: page.Count, OutcomeRevision: coord.Revision, EvaluatedAt: at, Omitted: page.Remaining + max(0, len(page.Items)-n)}
	for _, item := range page.Items[:min(n, len(page.Items))] {
		result.Items = append(result.Items, item.Value)
	}
	if len(page.Items) > n {
		last := page.Items[n-1]
		key := last.Key
		if section == "external_references" {
			key = "h:" + filterHash(last.Value)
		}
		result.NextCursor = encodeCursor(queryCursor{CursorVersion: 2, Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: section, Revision: coord.Revision, Key: key, EvaluatedAt: at})
	}
	raw, e := json.Marshal(result)
	if e != nil {
		return zero, true, e
	}
	if len(raw) > MaxSnapshotBytes {
		return zero, true, d.NewError(d.ErrorCodeGraphLimitExceeded, "section exceeds byte limit; reduce limit")
	}
	return result, true, nil
}
func legacyContinuityKey(ctx context.Context, repo ports.ContinuityReadRepository, scope d.Scope, section, digest string) (string, error) {
	after := ""
	for {
		page, e := repo.Page(ctx, scope, section, after, 100)
		if e != nil {
			return "", e
		}
		for _, item := range page.Items {
			if filterHash(item.Value) == digest {
				return item.Key, nil
			}
		}
		if len(page.Items) == 0 || page.Remaining == 0 {
			return "", d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: section key is missing")
		}
		after = page.Items[len(page.Items)-1].Key
	}
}
