package httptransport

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
)

func (h *Handler) effectivePermissions(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	scope := domain.Scope{NamespaceID: ns}
	if raw := r.URL.Query().Get("outcome_id"); raw != "" {
		scope.OutcomeID, err = domain.ParseID(raw)
		if err != nil {
			writeError(w, r, err)
			return
		}
	}
	permissions, err := h.service.EffectivePermissions(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"permissions": permissions, "execution_authority": false})
}
