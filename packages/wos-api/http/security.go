package httptransport

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
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
		Permissions     []ports.Permission `json:"permissions"`
		ExpectedVersion domain.Version     `json:"expected_namespace_version"`
	}
	if err = decodeJSON(w, r, &b); err != nil {
		writeError(w, r, err)
		return
	}
	result, _, err := h.security.AdministrativeMutation(r.Context(), r.Header.Get("Idempotency-Key"), ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: b.ExpectedVersion, Operation: "set_grant", Grant: &ports.NamespaceGrant{NamespaceID: ns, PrincipalID: r.PathValue("principal_id"), Permissions: b.Permissions}})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (h *Handler) issueCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var b struct {
		PrincipalID     string          `json:"principal_id"`
		Actor           domain.ActorRef `json:"actor_ref"`
		ExpiresAt       time.Time       `json:"expires_at"`
		ExpectedVersion domain.Version  `json:"expected_namespace_version"`
	}
	if err = decodeJSON(w, r, &b); err != nil {
		writeError(w, r, err)
		return
	}
	result, token, err := h.security.AdministrativeMutation(r.Context(), r.Header.Get("Idempotency-Key"), ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: b.ExpectedVersion, Operation: "issue_credential", Credential: &ports.Credential{NamespaceID: ns, PrincipalID: b.PrincipalID, Actor: b.Actor, ExpiresAt: b.ExpiresAt}})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"result": result, "token": token})
}
func (h *Handler) revokeCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "credential_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var body struct {
		ExpectedVersion domain.Version `json:"expected_namespace_version"`
	}
	if err = decodeJSON(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	result, _, err := h.security.AdministrativeMutation(r.Context(), r.Header.Get("Idempotency-Key"), ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: body.ExpectedVersion, Operation: "revoke_credential", CredentialID: id})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) securityAdministration(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.security.AdministrativeSnapshot(r.Context(), ns)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) administer(w http.ResponseWriter, r *http.Request) {
	var intent application.AdministrativeIntent
	if err := decodeJSON(w, r, &intent); err != nil {
		writeError(w, r, err)
		return
	}
	result, token, err := h.security.Administer(r.Context(), r.Header.Get("Idempotency-Key"), intent)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"result": result, "token": token})
}
