package httptransport

import (
	"net/http"
)

func (h *Handler) listTriggers(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items, revision, err := h.service.ListTriggers(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "outcome_revision": revision})
}
func (h *Handler) listDeliveries(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := queryLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	page, err := h.service.ListDeliveries(r.Context(), scope, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, page)
}
func (h *Handler) listFirings(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := queryLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	page, err := h.service.ListTriggerFirings(r.Context(), scope, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, page)
}
