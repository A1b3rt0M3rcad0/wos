package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func (h *Handler) listReadyWork(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ListReadyWork(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

var _ = domain.Readiness{}
