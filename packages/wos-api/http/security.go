package httptransport

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net/http"
	"time"
)

func (h *Handler) listNamespaces(w http.ResponseWriter, r *http.Request) {
	v, err := h.security.Namespaces(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": v})
}
func (h *Handler) setGrant(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var b struct {
		Permissions []ports.Permission `json:"permissions"`
	}
	if err = decodeJSON(w, r, &b); err != nil {
		writeError(w, r, err)
		return
	}
	err = h.security.SetGrant(r.Context(), ports.NamespaceGrant{NamespaceID: ns, PrincipalID: r.PathValue("principal_id"), Permissions: b.Permissions})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}
func (h *Handler) issueCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var b struct {
		PrincipalID string          `json:"principal_id"`
		Actor       domain.ActorRef `json:"actor_ref"`
		ExpiresAt   time.Time       `json:"expires_at"`
	}
	if err = decodeJSON(w, r, &b); err != nil {
		writeError(w, r, err)
		return
	}
	c, token, err := h.security.IssueCredential(r.Context(), ns, b.PrincipalID, b.Actor, b.ExpiresAt)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"credential": c, "token": token})
}
func (h *Handler) revokeCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "credential_id")
	if err == nil {
		err = h.security.RevokeCredential(r.Context(), ns, id)
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}
