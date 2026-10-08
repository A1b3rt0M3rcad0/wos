package httptransport

import (
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
)

func (h *Handler) signedState(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	query := a.SignedStateQuery{Scope: scope, Resource: r.PathValue("resource"), ContractKind: r.URL.Query().Get("contract_kind"), Digest: r.URL.Query().Get("digest"), IdempotencyKey: r.URL.Query().Get("idempotency_key"), Cursor: r.URL.Query().Get("cursor")}
	if value := r.PathValue("entity_id"); value != "" {
		query.ID, err = d.ParseID(value)
		if err != nil {
			writeError(w, r, err)
			return
		}
	}
	if r.URL.Query().Get("limit") != "" {
		query.Limit, err = queryLimit(r)
		if err != nil {
			writeError(w, r, err)
			return
		}
	}
	value, err := h.service.ReadSignedState(r.Context(), query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, value)
}
func (h *Handler) signedTrust(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	value, err := h.service.ReadSignedState(r.Context(), a.SignedStateQuery{Scope: d.Scope{NamespaceID: ns}, Resource: "trust"})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, value)
}
