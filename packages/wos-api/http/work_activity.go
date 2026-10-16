package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (h *Handler) listWorkActivity(w http.ResponseWriter, r *http.Request) {
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
	if r.URL.Query().Has("limit") && limit == 0 {
		writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "limit must be between 1 and 100"))
		return
	}
	query := r.URL.Query()
	var filter ports.WorkActivityFilter
	if query.Has("descending") {
		switch query.Get("descending") {
		case "true":
			filter.Descending = true
		case "false":
		default:
			writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "descending must be true or false"))
			return
		}
	}
	for name, target := range map[string]*domain.ID{"work_item_id": &filter.WorkItemID, "contract_id": &filter.ContractID} {
		if query.Get(name) != "" {
			*target, err = domain.ParseID(query.Get(name))
			if err != nil {
				writeError(w, r, err)
				return
			}
		}
	}
	page, err := h.service.ListWorkActivity(r.Context(), scope, filter, limit, query.Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) getWorkAttention(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	value, err := h.service.GetWorkAttention(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, value)
}
