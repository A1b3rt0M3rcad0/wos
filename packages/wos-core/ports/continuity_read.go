package ports

import (
	"container/heap"
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"sort"
	"unicode/utf8"
)

// ContinuitySummary is metadata, never a hydrated aggregate. Within one section
// scope and kind are constant: the first variable serialized field is Ref.ID.
// Thus UUID ordering exactly matches the legacy serialized summary ordering.
type ContinuitySummary struct {
	Ref           d.EntityRef `json:"ref"`
	Version       d.Version   `json:"version,omitempty"`
	Title         string      `json:"title,omitempty"`
	Lifecycle     string      `json:"lifecycle,omitempty"`
	Priority      d.Priority  `json:"priority,omitempty"`
	DetailOmitted bool        `json:"detail_omitted"`
}

func Summarize(ref d.EntityRef, version d.Version, title, lifecycle string, priority d.Priority) ContinuitySummary {
	if len(title) > 512 {
		title = title[:512]
		for !utf8.ValidString(title) {
			title = title[:len(title)-1]
		}
	}
	return ContinuitySummary{ref, version, title, lifecycle, priority, true}
}

// Keys are strict byte-order keys, not offsets or authority. Relations and
// evidence links start with their unique ID in the legacy JSON. External
// references require the exact Go-encoded JSON as key (escaping is significant).
type ContinuityReadItem struct {
	Key   string
	Value json.RawMessage
}
type ContinuityReadPage struct {
	Items     []ContinuityReadItem
	Count     int
	Remaining int
}
type ContinuityReadRepository interface {
	Outcome(context.Context, d.Scope) (ContinuitySummary, error)
	Page(context.Context, d.Scope, string, string, int) (ContinuityReadPage, error)
}
type ContinuityReadUnitOfWork interface {
	ContinuityReads() ContinuityReadRepository
}

func DirectContinuitySection(section string) bool {
	switch section {
	case "work_items", "objectives", "issues", "blockers", "artifacts", "evidence", "decisions", "relations", "evidence_links", "external_references":
		return true
	}
	return false
}

// A bounded builder lets Memory and legacy-cursor fallback keep only one page.
// SQL adapters use scoped COUNT and LIMIT; they do not feed all rows here.
type ContinuityPageBuilder struct {
	cursorFound              bool
	items                    continuityHeap
	count, afterCount, limit int
	after                    string
}

func NewContinuityPageBuilder(after string, limit int) (*ContinuityPageBuilder, error) {
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid continuity repository limit")
	}
	return &ContinuityPageBuilder{after: after, limit: limit}, nil
}
func (b *ContinuityPageBuilder) Add(key string, value any) error {
	b.count++
	if key == b.after {
		b.cursorFound = true
	}
	if key <= b.after {
		return nil
	}
	b.afterCount++
	if len(b.items) == b.limit && key >= b.items[0].Key {
		return nil
	}
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	item := ContinuityReadItem{key, raw}
	if len(b.items) < b.limit {
		heap.Push(&b.items, item)
	} else {
		b.items[0] = item
		heap.Fix(&b.items, 0)
	}
	return nil
}
func (b *ContinuityPageBuilder) Page() ContinuityReadPage {

	items := []ContinuityReadItem{}
	items = append(items, b.items...)
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return ContinuityReadPage{Items: items, Count: b.count, Remaining: b.afterCount - len(items)}
}

type continuityHeap []ContinuityReadItem

func (h continuityHeap) Len() int           { return len(h) }
func (h continuityHeap) Less(i, j int) bool { return h[i].Key > h[j].Key }
func (h continuityHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *continuityHeap) Push(v any)        { *h = append(*h, v.(ContinuityReadItem)) }
func (h *continuityHeap) Pop() any          { old := *h; v := old[len(old)-1]; *h = old[:len(old)-1]; return v }

func (b *ContinuityPageBuilder) HasAfter() bool { return b.after == "" || b.cursorFound }
