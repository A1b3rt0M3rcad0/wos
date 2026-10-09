package httptransport

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
	"strings"
)

func (h *Handler) searchReferences(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	n, err := queryLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if r.URL.Query().Has("limit") && n == 0 {
		writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "limit must be between 1 and 100"))
		return
	}
	kinds := []domain.EntityKind{}
	for _, kind := range strings.Split(r.URL.Query().Get("kind"), ",") {
		kinds = append(kinds, domain.EntityKind(kind))
	}
	q := application.ReferenceQuery{Kinds: kinds, Query: r.URL.Query().Get("query"), Lifecycle: r.URL.Query().Get("lifecycle"), Priority: domain.Priority(r.URL.Query().Get("priority")), Limit: n, Cursor: r.URL.Query().Get("cursor")}
	if raw := r.URL.Query().Get("id"); raw != "" {
		q.ID, err = domain.ParseID(raw)
		if err != nil {
			writeError(w, r, err)
			return
		}
	}
	v, err := h.service.SearchReferences(r.Context(), scope, q)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
