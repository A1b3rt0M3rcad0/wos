package httptransport

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net/http"
	"strconv"
)

func queryLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid limit")
	}
	return n, nil
}
func (h *Handler) searchOutcomes(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := queryLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	q := r.URL.Query()
	f := ports.OutcomeFilter{ExternalProvider: q.Get("external_provider"), ExternalKind: q.Get("external_kind"), ExternalID: q.Get("external_id"), Text: q.Get("text"), Lifecycle: q.Get("lifecycle"), Priority: domain.Priority(q.Get("priority"))}
	if raw := q.Get("archived"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid archived filter"))
			return
		}
		f.Archived = &v
	}
	v, err := h.service.SearchOutcomes(r.Context(), ns, f, limit, q.Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
func (h *Handler) getContinuity(w http.ResponseWriter, r *http.Request) {
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
	v, err := h.service.GetContinuity(r.Context(), scope, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
func (h *Handler) getContinuitySection(w http.ResponseWriter, r *http.Request) {
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
	v, err := h.service.GetContinuitySection(r.Context(), scope, r.PathValue("section"), limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
func (h *Handler) getTimeline(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()
	v, err := h.service.GetTimeline(r.Context(), scope, application.TimelineQuery{Limit: limit, Cursor: q.Get("cursor"), PrincipalID: q.Get("principal_id"), EntityID: domain.ID(q.Get("entity_id")), CommandID: domain.ID(q.Get("command_id")), EventType: q.Get("event_type")})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
func (h *Handler) getGraph(w http.ResponseWriter, r *http.Request) {
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
	depth := 2
	if raw := r.URL.Query().Get("depth"); raw != "" {
		depth, err = strconv.Atoi(raw)
		if err != nil {
			writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid depth"))
			return
		}
	}
	v, err := h.service.GetOutcomeGraph(r.Context(), scope, application.GraphQuery{Limit: limit, Depth: depth, Cursor: r.URL.Query().Get("cursor"), Direction: r.URL.Query().Get("direction")})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
func (h *Handler) getWorkContext(w http.ResponseWriter, r *http.Request) {
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
	v, err := h.service.GetWorkContext(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
